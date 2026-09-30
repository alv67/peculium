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
npm run dev        # http://localhost:4321/
npm run build      # static output in dist/
npm run preview    # serve the built dist/ locally at the root (/)
npm run check      # astro check (TypeScript diagnostics)
```

Requirements: Node **22.12+** (Astro 7's minimum).

## Site structure

| Route                  | Source                                   | Status                 |
| ---------------------- | ---------------------------------------- | ---------------------- |
| `/` and `/it/`         | `src/pages/index.astro`, `src/pages/it/index.astro` | real content |
| `/install` and `/it/install` | `src/pages/install.astro`, `src/pages/it/install.astro` | real content (pull-only guide) |
| `/releases` and `/it/releases` | `src/pages/releases.astro`, `src/pages/it/releases.astro` | real content (notes rendered from `docs/RELEASE-NOTES.*` at build time) |
| `/features` and `/it/features` | `src/pages/features.astro`, `src/pages/it/features.astro` | real content, with per-locale screenshots from `public/screenshots/en/` and `public/screenshots/it/` |
| `/manual` and `/it/manual` | `src/pages/manual/`, `src/pages/it/manual/` over the Content Layer collection `manual` (`src/content/docs/{en,it}/manual/*.md`) | user manual: landing lists the chapters by `order`; each chapter renders in `ManualLayout.astro` with the sidebar TOC and the language switcher (EN/IT chapters share their file slug) |

Manual content conventions:

- Frontmatter schema: `title`, `order`, `description` (`src/content.config.ts`).
- Images and internal links in the markdown are authored as full paths of
  the file's own locale — EN: `/install`, `/screenshots/en/manual/<file>.png`;
  IT: `/it/install`, `/screenshots/it/manual/<file>.png` — so every content
  link stays inside its locale. The Sätteri hast plugin
  `src/lib/satteri-site-urls.ts` prefixes them with the site base at build
  time, so content never hardcodes `/peculium`. The base passed to the
  plugin is the same `base` constant in `astro.config.mjs`.
- Headings get stable slug `id`s automatically (Astro's Sätteri pipeline), so
  in-app help can deep-link to sections.
- Screenshots for the manual live under `public/screenshots/{en,it}/manual/`,
  one file per locale, same slug in both.

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
  `docs/RELEASE-NOTES.en.md` / `.it.md`, so the build reads those files directly
  — never the network. `src/lib/releases.ts` locates them by walking up the
  directory tree from the module (Astro bundling makes fixed relative paths
  unreliable) and parses the version headings into entries; the EN pages use
  the EN notes, the IT pages the IT notes. `## Unreleased` sections are skipped
  (the site publishes released versions only). When a notes file is missing the
  build still succeeds: the pages degrade to a plain link to GitHub Releases.
  The site never stores hand-copied release notes.
- Keep dependencies light: Astro + Tailwind only. No CMS, no UI kit.

## Deployment (GitHub Pages, custom domain)

The site is published at **https://peculium.dev/** as a custom-domain Pages
site. `astro.config.mjs` uses:

```js
// astro.config.mjs
site: 'https://peculium.dev',
base: '/',
```

`website/public/CNAME` contains a single line, `peculium.dev` — it is copied
into `dist/` at build time and GitHub Pages uses it to keep the custom domain
on every deploy.

`.github/workflows/deploy-site.yml` builds on every push to `main` (and on
`v*` tags) and publishes with `actions/deploy-pages`. Repo settings, under
**Settings → Pages → Build and deployment**: **Source: GitHub Actions** and
**Custom domain: peculium.dev** (the setting and the `CNAME` file agree). At
the registrar, the domain's DNS records (the `A` records to the GitHub Pages
IPs and the `www` `CNAME` to `alv67.github.io`) point the domain at GitHub.

Every internal link and asset is generated through `import.meta.env.BASE_URL`
(now `/`), so pages and assets resolve at the domain root.
