/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#0c1017',
        surface: '#161b22',
        border: '#2a2d3a',
        accent: '#1ebe8a',
        textPrimary: '#ffffff',
        textSecondary: '#8b8fa8',
      },
      fontFamily: {
        sans: ['Space Grotesk', 'sans-serif'],
        mono: ['DM Mono', 'monospace'],
      },
      borderRadius: {
        DEFAULT: '8px',
      }
    },
  },
  plugins: [],
}
