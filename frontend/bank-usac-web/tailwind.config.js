/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        bank: {
          primary: '#0f172a',
          accent: '#2563eb',
          secondary: '#1e293b',
          gold: '#f59e0b',
        }
      }
    },
  },
  plugins: [],
}