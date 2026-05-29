'use client'

import { motion } from 'motion/react'
import { Check, Hammer, Rocket, X, Clock } from 'lucide-react'
import type { JobStatus, Phase } from '@/lib/types'
import { cn } from '@/lib/cn'

type PhaseState = 'pending' | 'active' | 'success' | 'failed'

interface PhaseTimelineProps {
  status: JobStatus | null
  phase: Phase | null
  hasDeploy: boolean
}

function resolveState(target: Phase, status: JobStatus | null, phase: Phase | null, hasDeploy: boolean): PhaseState {
  if (status === null) return 'pending'

  // For terminal jobs the per-event `phase` is not available; infer the failure
  // location from `hasDeploy` (true ⇒ deploy phase was reached).
  const failedPhase: Phase | null =
    status === 'failed' ? (phase ?? (hasDeploy ? 'deploy' : 'build')) : phase

  if (target === 'build') {
    if (status === 'building') return 'active'
    if (status === 'failed' && failedPhase === 'build') return 'failed'
    // success/deploying/failed-during-deploy → build finished OK
    if (status === 'pending') return 'pending'
    return 'success'
  }

  // target === 'deploy'
  if (!hasDeploy && status === 'failed') return 'pending'
  if (status === 'deploying') return 'active'
  if (status === 'success') return 'success'
  if (status === 'failed' && failedPhase === 'deploy') return 'failed'
  return 'pending'
}

function Node({ state, Icon, label }: { state: PhaseState; Icon: React.ElementType; label: string }) {
  const ring = {
    pending: 'border-zinc-800 bg-zinc-900 text-zinc-600',
    active:  'border-indigo-500/60 bg-indigo-500/15 text-indigo-300',
    success: 'border-emerald-500/50 bg-emerald-500/10 text-emerald-300',
    failed:  'border-red-500/50 bg-red-500/10 text-red-300',
  }[state]

  const StatusIcon = state === 'success' ? Check : state === 'failed' ? X : state === 'active' ? Icon : Clock

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative">
        <motion.div
          className={cn('flex h-9 w-9 items-center justify-center rounded-full border transition-colors', ring)}
          initial={false}
          animate={{ scale: state === 'active' ? 1.05 : 1 }}
          transition={{ duration: 0.24, ease: [0.22, 1, 0.36, 1] }}
        >
          <StatusIcon className="h-4 w-4" strokeWidth={2.5} />
        </motion.div>
        {state === 'active' && (
          <span className="absolute inset-0 -m-1 rounded-full bg-indigo-500/20 animate-pulse-glow" />
        )}
      </div>
      <span className={cn('text-[11px] font-medium uppercase tracking-wider', {
        'text-zinc-600': state === 'pending',
        'text-indigo-300': state === 'active',
        'text-emerald-400': state === 'success',
        'text-red-400': state === 'failed',
      })}>{label}</span>
    </div>
  )
}

export function PhaseTimeline({ status, phase, hasDeploy }: PhaseTimelineProps) {
  const buildState = resolveState('build', status, phase, hasDeploy)
  const deployState = resolveState('deploy', status, phase, hasDeploy)

  const connectorActive = buildState === 'success' || buildState === 'active'
  const connectorComplete = deployState === 'active' || deployState === 'success' || deployState === 'failed'

  return (
    <div className="flex items-center gap-3">
      <Node state={buildState} Icon={Hammer} label="Сборка" />
      <div className="relative -mt-5 h-px w-12 overflow-hidden bg-zinc-800">
        <motion.div
          className={cn(
            'absolute inset-y-0 left-0',
            connectorComplete ? 'bg-emerald-500/60' : connectorActive ? 'bg-indigo-500/60' : 'bg-zinc-800',
          )}
          initial={false}
          animate={{ width: connectorComplete ? '100%' : connectorActive ? '60%' : '0%' }}
          transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
        />
      </div>
      <Node state={deployState} Icon={Rocket} label="Деплой" />
    </div>
  )
}
