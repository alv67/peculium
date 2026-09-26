# Peculium website

The project website: a bilingual (English / Italian), zero-JS static site built
with [Astro](https://astro.build) and Tailwind CSS, published on GitHub Pages
via `.github/workflows/deploy-site.yml`.

This package is **independent** from `frontend/` (the SvelteKit app) — it has
its own `package.json`, lockfile and toolchain. It is documentation/presentation
content only: no API access, no auth.

## Local development

```sh
cd website
npm install        # once
npm run dev        # http://localhost:4321/peculium/
npm run build      # static output in dist/
npm run preview    # serve the built dist/ locally (with the /peculium base)
npm run check      # astro check (TypeScript diagnostics)
```

Requirements: Node **22.12+** (Astro 7's minimum).

## Site structure

| Route                  | Source                                   | Status                 |
| ---------------------- | ---------------------------------------- | ---------------------- |
| `/` and `/it/`         | `src/pages/index.astro`, `src/pages/it/index.astro` | real content |
| `/install` and `/it/install` | `src/pages/install.astro`, `src/pages/it/install.astro` | stub (issue #102, slice 2) |
| `/releases` and `/it/releases` | `src/pages/releases.astro`, `src/pages/it/releases.astro` | stub + latest-release card (full notes in slice 2) |
| `/features` and `/it/features` | `src/pages/features.astro`, `src/pages/it/features.astro` | stub (screenshots in slice 3) |

English is the default locale and is **not** prefixed (`prefixDefaultLocale:
false`); Italian pages live under `/it/...`. The nav has an EN/IT switcher that
links the equivalent page in the other locale.

Key conventions:

- **Base-aware everything.** Every internal link and asset goes through
  `withBase()` in `src/lib/base.ts` (or `import.meta.env.BASE_URL` directly).
  Never hardcode root-absolute URLs like `/install` in markup.
- **Tailwind v3** (same major as `frontend/`), wired through Astro's native
  PostCSS support: `postcss.config.mjs` + `tailwind.config.mjs` + the
  `@tailwind` directives in `src/styles/global.css`. The `@astrojs/tailwind`
  integration is not used because its peer range does not cover Astro 7.
- **Single source of truth for releases.** GitHub Releases are created from
  `docs/RELEASE-NOTES.en.md` / `.it.md`, so the build reads that file directly —
  never the network. The latest-release card parses the first version heading
  and degrades gracefully to a plain link to the releases page when the file or
  heading is missing; the releases page will render the notes in full (slice 2).
  The site never stores hand-copied release notes.
- Keep dependencies light: Astro + Tailwind only. No CMS, no UI kit.

## Deployment (GitHub Pages, project site)

The site currently deploys as a **project Pages site**
(`https://alv67.github.io/peculium/`):

```js
// astro.config.mjs
site: 'https://alv67.github.io',
base: '/peculium',
```

`.github/workflows/deploy-site.yml` builds on every push to `main` (and on
`v*` tags) and publishes with `actions/deploy-pages`. One-time repo setting:
**Settings → Pages → Build and deployment → Source: GitHub Actions**.

## Moving to a custom domain (peculium.dev)

The site is portable by design — switching to `peculium.dev` is config + DNS
only, no content refactor:

1. **Buy/point the domain.** At your registrar, add the DNS records GitHub
   asks for the domain (typically four `A` records to GitHub Pages IPs and/or
   a `CNAME` for `www`, e.g. to `alv67.github.io`).
2. **`website/astro.config.mjs`** — change to:

   ```js
   site: 'https://peculium.dev',
   base: '/',
   ```

3. **`website/public/CNAME`** — create the file with a single line:

   ```text
   peculium.dev
   ```

   (it is copied into `dist/` at build time and GitHub Pages picks it up;
   do not enable "Enforce HTTPS" until GitHub reports the domain verified).
4. **Repo settings** — if a `CNAME` already existed in the Pages settings,
   update it there too (the file and the setting should agree).
5. Re-run **Deploy site** (push or `workflow_dispatch`). Every internal link
   keeps working because they are all generated through
   `import.meta.env.BASE_URL` (now `/`).
