import { cn } from '@/lib/cn'

export function Logo({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-indigo-500 to-violet-600 shadow-[0_0_18px_-4px_rgba(99,102,241,0.7)]">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2.4}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-3.5 w-3.5 text-white"
          aria-hidden="true"
        >
          <path d="M4 17l8 4 8-4" />
          <path d="M4 12l8 4 8-4" />
          <path d="M12 3L4 7l8 4 8-4-8-4z" />
        </svg>
      </span>
      <span className="text-sm font-semibold tracking-tight text-zinc-100">diployment</span>
    </div>
  )
}
