import * as React from 'react'
import { cn } from '@/lib/cn'

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: React.ReactNode
  action?: React.ReactNode
  className?: string
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 px-8 py-16 text-center',
        className,
      )}
    >
      {icon && (
        <div className="flex h-14 w-14 items-center justify-center rounded-full border border-zinc-800 bg-gradient-to-br from-indigo-500/10 to-violet-500/5 text-zinc-300">
          {icon}
        </div>
      )}
      <div className="flex flex-col gap-1.5">
        <h3 className="text-base font-medium text-zinc-100">{title}</h3>
        {description && <div className="max-w-sm text-sm text-zinc-500">{description}</div>}
      </div>
      {action}
    </div>
  )
}
