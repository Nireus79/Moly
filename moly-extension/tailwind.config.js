/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#6366f1',
        secondary: '#8b5cf6',
      },
      fontFamily: {
        sans: ['system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
  safelist: [
    'bg-blue-50',
    'text-blue-900',
    'border-blue-200',
    'bg-purple-50',
    'text-purple-900',
    'border-purple-200',
    'bg-pink-50',
    'text-pink-900',
    'border-pink-200',
  ],
}
