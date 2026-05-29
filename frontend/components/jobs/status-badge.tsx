import { CheckCircle2, XCircle, Clock, Hammer, Rocket } from 'lucide-react'
import type { JobStatus } from '@/lib/types'
import { cn } from '@/lib/cn'

interface StatusBadgeProps {
  status: JobStatus
  size?: 'sm' | 'md'
}

const config: Record<JobStatus, {
  label: string
  tone: string
  Icon: React.ElementType
  pulse?: boolean
}> = {
  pending:   { label: 'В очереди', tone: 'border-zinc-700 bg-zinc-800/60 text-zinc-300',                       Icon: Clock },
  building:  { label: 'Сборка',    tone: 'border-indigo-500/40 bg-indigo-500/10 text-indigo-300',              Icon: Hammer, pulse: true },
  deploying: { label: 'Деплой',    tone: 'border-violet-500/40 bg-violet-500/10 text-violet-300',              Icon: Rocket, pulse: true },
  success:   { label: 'Готово',    tone: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300',           Icon: CheckCircle2 },
  failed:    { label: 'Упало',     tone: 'border-red-500/40 bg-red-500/10 text-red-300',                       Icon: XCircle },
}

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const { label, tone, Icon, pulse } = config[status]
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border font-medium leading-none',
        size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-1 text-xs',
        tone,
      )}
    >
      <span className="relative inline-flex h-1.5 w-1.5 items-center justify-center">
        <span className={cn('h-1.5 w-1.5 rounded-full bg-current', pulse && 'animate-pulse-dot')} />
        {pulse && (
          <span className="absolute inset-0 -m-0.5 rounded-full bg-current opacity-30 animate-pulse-glow" />
        )}
      </span>
      <Icon className={size === 'sm' ? 'h-3 w-3' : 'h-3.5 w-3.5'} strokeWidth={2.5} />
      {label}
    </span>
  )
}

export function StatusDot({ status, className }: { status: JobStatus; className?: string }) {
  const { tone, pulse } = config[status]
  const baseTone = tone.match(/text-[a-z]+-\d+/)?.[0] ?? 'text-zinc-400'
  return (
    <span className={cn('relative inline-flex h-2 w-2', className)}>
      <span className={cn('h-2 w-2 rounded-full bg-current', baseTone, pulse && 'animate-pulse-dot')} />
      {pulse && <span className={cn('absolute inset-0 -m-0.5 rounded-full bg-current opacity-30 animate-pulse-glow', baseTone)} />}
    </span>
  )
}
