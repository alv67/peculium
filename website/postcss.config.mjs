/**
 * Tailwind v3 is wired through Astro's native PostCSS support (this file +
 * tailwind.config.mjs + the @tailwind directives in src/styles/global.css).
 * This is exactly what the old @astrojs/tailwind integration did internally;
 * the integration is not used because its peer range does not cover Astro 7.
 * The version matches frontend/ (Tailwind v3) on purpose.
 */
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
};
