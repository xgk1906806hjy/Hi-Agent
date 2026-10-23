/** 黑色 Orbit 图标：圆角方块 + 斜切环。 */
export function AppIcon({ size = 16, className = "" }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={`text-ink ${className}`}
      aria-hidden
    >
      <rect x="1.5" y="1.5" width="29" height="29" rx="8" fill="currentColor" />
      <path
        d="M22.8 8.6a9.2 9.2 0 1 0 1.1 12.4"
        stroke="var(--color-bg)"
        strokeWidth="2.6"
        strokeLinecap="round"
      />
      <path
        d="M9.2 23.4a9.2 9.2 0 0 1-1.1-12.4"
        stroke="var(--color-bg)"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.55"
      />
    </svg>
  );
}
