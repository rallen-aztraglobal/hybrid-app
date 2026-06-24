/** @type {import('tailwindcss').Config} */
// 设计令牌取自 docs/admin/ui/index.html（视觉权威）。颜色用 CSS 变量驱动，便于暗色侧栏与浅色主区共存。
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: 'var(--bg)',
        panel: 'var(--panel)',
        'panel-2': 'var(--panel-2)',
        ink: 'var(--ink)',
        'ink-2': 'var(--ink-2)',
        muted: 'var(--muted)',
        line: 'var(--line)',
        'line-2': 'var(--line-2)',
        brand: 'var(--brand)',
        'brand-ink': 'var(--brand-ink)',
        ok: 'var(--ok)',
        warn: 'var(--warn)',
        down: 'var(--down)',
        // 大渠道品牌色
        ap: 'var(--ap)',
        bp: 'var(--bp)',
        gp: 'var(--gp)',
      },
      fontFamily: {
        mono: 'var(--mono)',
        sans: 'var(--sans)',
      },
      borderRadius: {
        card: 'var(--radius)',
        sm2: 'var(--radius-sm)',
      },
      boxShadow: {
        'sm2': 'var(--shadow-sm)',
        'md2': 'var(--shadow-md)',
        'lg2': 'var(--shadow-lg)',
      },
      keyframes: {
        fade: {
          from: { opacity: '0', transform: 'translateY(6px)' },
          to: { opacity: '1', transform: 'none' },
        },
      },
      animation: {
        fade: 'fade .25s ease',
      },
    },
  },
  plugins: [],
};
