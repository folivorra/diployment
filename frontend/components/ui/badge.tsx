import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/cn'

const badgeVariants = cva(
  'inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium leading-none',
  {
    variants: {
      tone: {
        neutral: 'border-zinc-800 bg-zinc-900/80 text-zinc-300',
        accent:  'border-indigo-500/30 bg-indigo-500/10 text-indigo-300',
        success: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
        danger:  'border-red-500/30 bg-red-500/10 text-red-300',
        warn:    'border-amber-500/30 bg-amber-500/10 text-amber-300',
        violet:  'border-violet-500/30 bg-violet-500/10 text-violet-300',
      },
    },
    defaultVariants: { tone: 'neutral' },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />
}
