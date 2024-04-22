/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: { extend: { colors: { surface: { 0: '#0a0e17', 1: '#0f1623', 2: '#151e2e', 3: '#1c2a40' } }, fontFamily: { mono: ['"JetBrains Mono"', 'monospace'], sans: ['"DM Sans"', 'sans-serif'] } } },
  plugins: [],
}
