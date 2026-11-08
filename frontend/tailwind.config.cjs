/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'sans-serif',
        ],
        mono: ['Consolas', 'Monaco', 'Courier New', 'monospace'],
      },
      fontSize: {
        xs: ['0.75rem', { lineHeight: '1.5', letterSpacing: '0.01em' }],
        sm: ['0.9rem', { lineHeight: '1.55', letterSpacing: '0.01em' }],
        base: ['1.05rem', { lineHeight: '1.65', letterSpacing: '0' }],
        lg: ['1.18rem', { lineHeight: '1.55', letterSpacing: '-0.01em' }],
        xl: ['1.28rem', { lineHeight: '1.45', letterSpacing: '-0.02em' }],
        '2xl': ['1.55rem', { lineHeight: '1.32', letterSpacing: '-0.02em' }],
      },
      transitionTimingFunction: {
        smooth: 'cubic-bezier(0.4, 0, 0.2, 1)',
      },
      animation: {
        typing: 'typing 1.4s ease-in-out infinite',
      },
    },
  },
  plugins: [],
};
