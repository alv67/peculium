#!/usr/bin/env node
// Capture a Peculium page as a docs/website screenshot using headless Chrome
// driven over the DevTools Protocol.
//
// It reproduces the reference spec of website/public/screenshots: a 1280x800
// viewport at 2x DPR (2560x1600 px) in the LIGHT theme, captured from the
// isolated test stack — never the dev/prod one.
//
// Prerequisites:
//   1. Isolated test stack up (backend on :8081) with a seeded user, e.g.
//        podman-compose -p peculium-test -f docker-compose.test.yml up -d --build backend
//        # seed a user via POST /api/v1/auth/register + login (see tests/api-test.http)
//      Note: the test compose has no worker, so enqueue/run jobs by hand if the
//      page needs activity rows (a worker image on the same network works).
//   2. Frontend dev server up, its /api proxy pointed at :8081, e.g.
//        # temporarily set `server.proxy['/api'].target` to http://localhost:8081
//        cd frontend && npm run dev -- --host
//   3. Headless Chrome with a remote debugging port:
//        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
//          --headless=new --disable-gpu --hide-scrollbars --no-first-run \
//          --remote-debugging-port=9222 --user-data-dir=/tmp/pec-chrome about:blank &
//
// Usage:
//   node scripts/capture-screenshots.mjs <out.png> <en|it> [route]
//   # e.g. node scripts/capture-screenshots.mjs website/public/screenshots/en/price-health.png en /admin/health
//
// Env overrides:
//   PECULIUM_SHOT_FRONT     frontend base URL      (default http://localhost:5173)
//   PECULIUM_SHOT_API       API base URL           (default http://localhost:8081/api/v1)
//   PECULIUM_SHOT_EMAIL     login email            (default shot@example.com)
//   PECULIUM_SHOT_PASSWORD  login password         (default Password123!)
//   PECULIUM_SHOT_CDP_PORT  Chrome debug port      (default 9222)
import { writeFileSync } from 'node:fs'

const FRONT = process.env.PECULIUM_SHOT_FRONT || 'http://localhost:5173'
const API = process.env.PECULIUM_SHOT_API || 'http://localhost:8081/api/v1'
const EMAIL = process.env.PECULIUM_SHOT_EMAIL || 'shot@example.com'
const PASSWORD = process.env.PECULIUM_SHOT_PASSWORD || 'Password123!'
const PORT = process.env.PECULIUM_SHOT_CDP_PORT || '9222'

const [out, locale = 'en', route = '/admin/health'] = process.argv.slice(2)
if (!out) {
  console.error('usage: node scripts/capture-screenshots.mjs <out.png> <en|it> [route]')
  process.exit(2)
}

const login = await fetch(`${API}/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
})
const { access_token, refresh_token } = await login.json()
if (!access_token) throw new Error(`login failed (${login.status}) — is the stack up and the user seeded?`)

const targets = await (await fetch(`http://127.0.0.1:${PORT}/json/list`)).json()
const page = targets.find((t) => t.type === 'page')
if (!page) throw new Error('no Chrome page target — is headless Chrome running with --remote-debugging-port?')

const ws = new WebSocket(page.webSocketDebuggerUrl)
await new Promise((r) => (ws.onopen = r))
let id = 0
const pending = new Map()
ws.onmessage = (ev) => {
  const m = JSON.parse(ev.data)
  if (m.id && pending.has(m.id)) {
    pending.get(m.id)(m)
    pending.delete(m.id)
  }
}
const send = (method, params = {}) =>
  new Promise((res) => {
    const mid = ++id
    pending.set(mid, res)
    ws.send(JSON.stringify({ id: mid, method, params }))
  })
const evalJS = async (expression) => {
  const r = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  return r.result?.result?.value
}
const waitFor = async (expr, tries = 80) => {
  for (let i = 0; i < tries; i++) {
    if (await evalJS(expr)) return true
    await new Promise((r) => setTimeout(r, 500))
  }
  return false
}

await send('Page.enable')
await send('Runtime.enable')
// The reference spec: 1280x800 CSS px at 2x DPR => 2560x1600 px.
await send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 800, deviceScaleFactor: 2, mobile: false })

await send('Page.navigate', { url: `${FRONT}/login` })
await waitFor('document.readyState === "complete"')
await evalJS(
  `localStorage.setItem('access_token', ${JSON.stringify(access_token)});` +
    `localStorage.setItem('refresh_token', ${JSON.stringify(refresh_token)});` +
    `localStorage.setItem('peculium-locale', ${JSON.stringify(locale)});` +
    `localStorage.setItem('peculium-theme', 'light'); 'ok'`,
)

await send('Page.navigate', { url: FRONT + route })
const ok = await waitFor('document.querySelectorAll("tbody tr").length > 0')
await new Promise((r) => setTimeout(r, 1200))
const shot = await send('Page.captureScreenshot', { format: 'png' })
writeFileSync(out, Buffer.from(shot.result.data, 'base64'))
console.log(`wrote ${out} (${locale}, ${route}) content_loaded=${ok}`)
ws.close()
process.exit(0)
