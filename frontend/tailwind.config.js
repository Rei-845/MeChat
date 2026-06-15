/** @type {import('tailwindcss').Config} */
// 语义化双主题：颜色以 CSS 变量（RGB 通道）形式注入，配合 <alpha-value> 让 /透明度 修饰符照常可用。
// 默认浅色（Telegram 风），html.dark 切换为 Telegram 夜间。具体变量见 style.css。
const ch = (v) => `rgb(var(${v}) / <alpha-value>)`

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        primary: { DEFAULT: ch('--primary'), light: ch('--primary-light') }, // 品牌蓝
        ink:     { DEFAULT: ch('--ink'), 2: ch('--ink-2'), 3: ch('--ink-3') }, // 主/次/提示文字
        surface: { DEFAULT: ch('--surface'), 2: ch('--surface-2') },          // 面板 + 悬浮
        bg:      ch('--app-bg'),                                              // 应用底色（bg-bg）
        accent:  ch('--accent'),
        danger:  ch('--danger'),
        warning: ch('--warning'),
        online:  ch('--online'),
        border:  ch('--border'),
      },
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'system-ui', '-apple-system', 'Roboto', 'sans-serif'],
      },
      borderRadius: {
        sm: '8px', md: '12px', lg: '16px', xl: '20px', '2xl': '24px',
      },
      backdropBlur: { xs: '4px', sm: '8px', md: '16px', lg: '24px' },
      animation: {
        'blob': 'blob 12s ease-in-out infinite',
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-up': 'slideUp 0.25s cubic-bezier(0.16,1,0.3,1)',
        'scale-in': 'scaleIn 0.2s cubic-bezier(0.16,1,0.3,1)',
      },
      keyframes: {
        blob: {
          '0%,100%': { transform: 'translate(0,0) scale(1)' },
          '33%':     { transform: 'translate(30px,-20px) scale(1.05)' },
          '66%':     { transform: 'translate(-20px,10px) scale(0.95)' },
        },
        fadeIn:   { from: { opacity: '0' }, to: { opacity: '1' } },
        slideUp:  { from: { opacity: '0', transform: 'translateY(12px)' }, to: { opacity: '1', transform: 'translateY(0)' } },
        scaleIn:  { from: { opacity: '0', transform: 'scale(0.95)' }, to: { opacity: '1', transform: 'scale(1)' } },
      },
      boxShadow: {
        'glass': '0 1px 2px rgba(16,28,40,0.06), 0 4px 16px rgba(16,28,40,0.06)',
        'glow':  '0 0 0 3px rgb(var(--primary) / 0.15)',
        'glow-sm': '0 0 0 2px rgb(var(--primary) / 0.18)',
      },
    },
  },
  plugins: [],
}
