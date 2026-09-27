/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,ts}'],
  theme: {
    extend: {
      colors: {
        // Brand accent for the site (matches the app's semantic role of
        // `accent` without importing the app's CSS token machinery).
        brand: {
          DEFAULT: '#059669',
          dark: '#34d399',
        },
      },
    },
  },
  plugins: [],
};
