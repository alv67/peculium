// @ts-check
import { defineConfig } from 'astro/config';

// Project Pages site (https://<owner>.github.io/<repo>/).
//
// When the custom domain (peculium.dev) lands, change ONLY this block to:
//   site: 'https://peculium.dev',
//   base: '/',
// and add a `CNAME` file (`peculium.dev`) under `public/` — see README.md.
// Every internal link/asset in the site goes through import.meta.env.BASE_URL,
// so nothing else needs to change.
export default defineConfig({
  output: 'static',
  site: 'https://alv67.github.io',
  base: '/peculium',
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
