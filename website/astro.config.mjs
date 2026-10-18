// @ts-check
import { defineConfig } from 'astro/config';
import { satteri } from '@astrojs/markdown-satteri';
import satteriSiteUrls from './src/lib/satteri-site-urls';

// Custom-domain site published at https://peculium.dev/, served at the root
// (base '/'). `public/CNAME` holds `peculium.dev` so GitHub Pages keeps the
// domain on every deploy.
// Every internal link/asset in the site goes through import.meta.env.BASE_URL
// (via withBase() in src/lib/base.ts), so the URLs stay correct under any base.
// Markdown content (the manual) can't call withBase(), so the same base is
// handed to the Sätteri site-urls plugin below: change `base` here and the
// two stay in sync. Sätteri also adds stable slug ids to every heading out
// of the box, which is what manual deep links rely on.
const base = '/';

export default defineConfig({
  output: 'static',
  site: 'https://peculium.dev',
  base,
  trailingSlash: 'ignore',
  markdown: {
    processor: satteri({ hastPlugins: [satteriSiteUrls(base)] }),
  },
  i18n: {
    locales: ['en', 'it'],
    defaultLocale: 'en',
    routing: {
      // English lives at `/`, `/install`, ...; Italian at `/it`, `/it/install`, ...
      prefixDefaultLocale: false,
    },
  },
});
