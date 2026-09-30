/**
 * Sätteri hast plugin for rendered markdown (the manual chapters): makes
 * authored root-relative URLs base-aware.
 *
 * `img[src]` and `a[href]` written as `/…` (e.g. `/screenshots/en/manual/x.png`,
 * `/install`) are prefixed with the site base so content stays correct under
 * any deployment base (`/peculium` on project Pages, `/` on the custom
 * domain). Idempotent: already-prefixed paths, protocol-relative (`//…`) and
 * absolute URLs pass through untouched.
 *
 * Heading anchor ids do NOT need a plugin: Astro's Sätteri processor adds
 * stable GitHub-style slug ids to every heading (and exposes the `headings`
 * list) out of the box.
 *
 * The base is injected by astro.config from the same constant that sets
 * `base:`, so the plugin and the build can never disagree.
 */

type HastElement = { tagName?: string; properties?: Record<string, unknown> };
type Ctx = { setProperty(node: HastElement, key: string, value: unknown): void };

export default function satteriSiteUrls(base: string) {
  const prefix = base.replace(/\/+$/, '');
  const underBase = (p: string): string =>
    prefix === '' || p === prefix || p.startsWith(`${prefix}/`) ? p : `${prefix}${p}`;

  return {
    name: 'site-urls',
    element: {
      filter: ['img', 'a'],
      visit(node: HastElement, ctx: Ctx): void {
        const key = node.tagName === 'img' ? 'src' : 'href';
        const value = node.properties?.[key];
        // `/…` but not `//…` (protocol-relative).
        if (typeof value === 'string' && value.startsWith('/') && !value.startsWith('//')) {
          ctx.setProperty(node, key, underBase(value));
        }
      },
    },
  };
}
