// @ts-check
import { defineConfig } from 'astro/config';

// Custom-domain site published at https://peculium.dev/, served at the root
// (base '/'). `public/CNAME` holds `peculium.dev` so GitHub Pages keeps the
// domain on every deploy.
// Every internal link/asset in the site goes through import.meta.env.BASE_URL
// (via withBase() in src/lib/base.ts), so the URLs stay correct under any base.
export default defineConfig({
  output: 'static',
  site: 'https://peculium.dev',
  base: '/',
  trailingSlash: 'ignore',
  i18n: {
    locales: ['en', 'it'],
    defaultLocale: 'en',
    routing: {
      // English lives at `/`, `/install`, ...; Italian at `/it`, `/it/install`, ...
      prefixDefaultLocale: false,
    },
  },
});
