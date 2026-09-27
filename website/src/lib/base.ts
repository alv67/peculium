/**
 * Base-aware URL helpers.
 *
 * The site is deployed under a base path (project GitHub Pages: `/peculium`).
 * Every internal link and asset in the site MUST be produced through
 * `withBase()` (or go through `import.meta.env.BASE_URL` directly) so that
 * moving to a custom domain (`base: '/'`) is only an astro.config change.
 */

/** The configured site base without a trailing slash, e.g. `/peculium` or ``. */
function basePrefix(): string {
  return import.meta.env.BASE_URL.replace(/\/+$/, '');
}

/**
 * Prefix a root-relative site path (e.g. `/`, `/install`, `/peculium.svg`)
 * with the deployment base. Paths that already start with the base are
 * returned unchanged; absolute URLs (http/https/mailto) pass through.
 */
export function withBase(path: string): string {
  if (/^(https?:|mailto:)/.test(path)) return path;
  const base = basePrefix();
  const normalized = path.startsWith('/') ? path : `/${path}`;
  if (base === '' || normalized === '/') return `${base}/`;
  if (normalized === base || normalized.startsWith(`${base}/`)) return normalized;
  return `${base}${normalized}`;
}

/** Locale-root-relative path for the current URL path (with or without base). */
export function toRootPath(pathname: string): string {
  let p = pathname;
  const base = basePrefix();
  if (base && (p === base || p.startsWith(`${base}/`))) p = p.slice(base.length);
  if (p === '/it' || p.startsWith('/it/')) p = p.slice(3);
  return p === '' ? '/' : p;
}

/** Path (without base) for the same page in the other locale. */
export function localeSwitchPath(pathname: string, target: 'en' | 'it'): string {
  const root = toRootPath(pathname);
  return target === 'en' ? root : withIt(root);
}

/**
 * Build a root-relative page path for a specific locale, e.g.
 * `localePath('/install', 'it')` -> `/it/install`. Intended for locale-root
 * paths like the nav entries; `toRootPath` is idempotent on those.
 */
export function localePath(root: string, locale: 'en' | 'it'): string {
  return localeSwitchPath(root, locale);
}

function withIt(root: string): string {
  return root === '/' ? '/it' : `/it${root}`;
}
