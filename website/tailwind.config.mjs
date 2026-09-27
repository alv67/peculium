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
      boxShadow: {
        // Matches the app's `shadow-card` resting elevation.
        card: '0 1px 2px 0 rgb(0 0 0 / 0.05), 0 1px 3px 0 rgb(0 0 0 / 0.1)',
      },
    },
  },
  plugins: [],
};
