'use client'

import React, { useEffect, useRef, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { CheckCircle2, XCircle, Loader2, AlertCircle } from 'lucide-react'
import { getApiUrl } from '@/lib/api'
import StatusBadge from '@/components/StatusBadge'
import type { JobStatus, Phase } from '@/lib/types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface StatusEvent {
  status: JobStatus
  phase: Phase
  error?: string
}

interface LogEvent {
  line: string
  phase: Phase
  done?: boolean
}

type StreamState = 'connecting' | 'streaming' | 'done' | 'error'

// ---------------------------------------------------------------------------
// Hook: useJobStream
// Connects to GET /api/jobs/:id/events, multiplexes status + log named SSE
// events across two phases (build / deploy), and manages cleanup on unmount.
//
// Closing logic mirrors the backend protocol:
//   - deploy `done` sentinel → always close (deploy is the final phase)
//   - build `done` sentinel → record buildDone; close only if terminal status
//     already arrived (i.e. job ended during build phase without deploying)
// ---------------------------------------------------------------------------

interface JobStream {
  status: JobStatus | null
  phase: Phase | null
  buildLines: string[]
  deployLines: string[]
  streamState: StreamState
  streamError: string | null
}

function useJobStream(jobId: string): JobStream {
  const [status, setStatus] = useState<JobStatus | null>(null)
  const [phase, setPhase] = useState<Phase | null>(null)
  const [buildLines, setBuildLines] = useState<string[]>([])
  const [deployLines, setDeployLines] = useState<string[]>([])
  const [streamState, setStreamState] = useState<StreamState>('connecting')
  const [streamError, setStreamError] = useState<string | null>(null)

  const gotTerminalStatus = useRef(false)
  const buildDone = useRef(false)
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    gotTerminalStatus.current = false
    buildDone.current = false
    setStatus(null)
    setPhase(null)
    setBuildLines([])
    setDeployLines([])
    setStreamState('connecting')
    setStreamError(null)

    const es = new EventSource(`${getApiUrl()}/api/jobs/${jobId}/events`, {
      withCredentials: true,
    })
    esRef.current = es

    function maybeClose(): void {
      if (gotTerminalStatus.current && buildDone.current) {
        es.close()
        esRef.current = null
        setStreamState('done')
      }
    }

    es.addEventListener('status', (e: MessageEvent) => {
      const data = JSON.parse(e.data as string) as StatusEvent
      setStatus(data.status)
      setPhase(data.phase)
      setStreamState('streaming')

      const isTerminal = data.status === 'success' || data.status === 'failed'
      if (isTerminal) {
        gotTerminalStatus.current = true
        setStreamError(data.error ?? null)
        maybeClose()
      }
    })

    es.addEventListener('log', (e: MessageEvent) => {
      const data = JSON.parse(e.data as string) as LogEvent
      if (data.done) {
        if (data.phase === 'deploy') {
          // Deploy done is always the final sentinel — close unconditionally.
          es.close()
          esRef.current = null
          setStreamState('done')
        } else {
          // Build done: only close if we already have a terminal status
          // (means the job failed during build — no deploy phase will follow).
          buildDone.current = true
          maybeClose()
        }
      } else {
        if (data.phase === 'deploy') {
          setDeployLines((prev) => [...prev, data.line])
        } else {
          setBuildLines((prev) => [...prev, data.line])
        }
      }
    })

    es.onerror = () => {
      es.close()
      esRef.current = null
      setStatus((current) => {
        const isFinished = current === 'success' || current === 'failed'
        if (!isFinished) {
          setStreamState('error')
        } else {
          setStreamState('done')
        }
        return current
      })
    }

    return () => {
      es.close()
      esRef.current = null
    }
  }, [jobId])

  return { status, phase, buildLines, deployLines, streamState, streamError }
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StatusCard({
  status,
  streamState,
  streamError,
}: {
  status: JobStatus | null
  streamState: StreamState
  streamError: string | null
}): React.JSX.Element {
  const isLive = status === 'building' || status === 'deploying'

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-5 py-4 mb-4">
      <div className="flex items-center gap-3">
        <span className="text-sm text-zinc-400">Status</span>
        {status === null ? (
          <span className="flex items-center gap-1.5 text-sm text-zinc-500">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Connecting…
          </span>
        ) : (
          <StatusBadge status={status} />
        )}
        {streamState === 'streaming' && isLive && (
          <span className="ml-auto text-xs text-zinc-600 flex items-center gap-1">
            <span className="inline-block h-1.5 w-1.5 rounded-full bg-blue-400 animate-pulse" />
            Live
          </span>
        )}
      </div>
      {streamError && (
        <p className="mt-2 flex items-center gap-1.5 text-xs text-red-400">
          <AlertCircle className="h-3.5 w-3.5 shrink-0" />
          {streamError}
        </p>
      )}
    </div>
  )
}

interface PhaseLogSectionProps {
  /** Display title for the section header */
  title: string
  lines: string[]
  /** Whether this phase is currently the active streaming phase */
  isActive: boolean
  /** Whether the overall stream has finished */
  streamDone: boolean
  /** Whether this specific phase succeeded */
  isPhaseSuccess: boolean
  /** Whether this specific phase failed */
  isPhaseFailed: boolean
  /** Label shown in the success marker, e.g. "Build succeeded" */
  successLabel: string
  /** Ref to scroll into view — only the last section owns the scroll anchor */
  scrollRef?: React.RefObject<HTMLDivElement | null>
}

function PhaseLogSection({
  title,
  lines,
  isActive,
  streamDone,
  isPhaseSuccess,
  isPhaseFailed,
  successLabel,
  scrollRef,
}: PhaseLogSectionProps): React.JSX.Element {
  return (
    <div className="rounded-xl border border-zinc-800 overflow-hidden">
      {/* Section header */}
      <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900 px-5 py-3">
        <span className="text-sm font-medium text-zinc-400">{title}</span>
        {isActive && !streamDone && (
          <span className="flex items-center gap-1.5 text-xs text-zinc-600">
            <Loader2 className="h-3 w-3 animate-spin" />
            Streaming
          </span>
        )}
        {(!isActive || streamDone) && lines.length > 0 && (
          <span className="flex items-center gap-1.5 text-xs text-zinc-600">
            <CheckCircle2 className="h-3 w-3" />
            Complete
          </span>
        )}
      </div>

      {/* Log body */}
      <div className="h-72 overflow-y-auto bg-zinc-950 p-4 font-mono text-xs leading-5">
        {lines.length === 0 ? (
          <span className="text-zinc-600">
            {isActive ? 'Waiting for logs…' : 'No logs.'}
          </span>
        ) : (
          lines.map((line, i) => (
            <div key={i} className="whitespace-pre-wrap break-all text-zinc-300">
              {line.replace(/\x1b\[[0-9;]*[mGKHF]/g, '').replace(/\r/g, '').replace(/\n$/, '')}
            </div>
          ))
        )}

        {streamDone && lines.length > 0 && (isPhaseSuccess || isPhaseFailed) && (
          <div className="mt-3 flex items-center gap-2 border-t border-zinc-800 pt-3">
            {isPhaseSuccess ? (
              <>
                <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-400" />
                <span className="text-emerald-400">{successLabel}</span>
              </>
            ) : (
              <>
                <XCircle className="h-3.5 w-3.5 shrink-0 text-red-400" />
                <span className="text-red-400">Failed</span>
              </>
            )}
          </div>
        )}

        <div ref={scrollRef} />
      </div>
    </div>
  )
}

interface LogPanelProps {
  buildLines: string[]
  deployLines: string[]
  streamState: StreamState
  status: JobStatus | null
  phase: Phase | null
}

function LogPanel({ buildLines, deployLines, streamState, status, phase }: LogPanelProps): React.JSX.Element {
  const buildScrollRef = useRef<HTMLDivElement>(null)
  const deployScrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll the active section on new lines.
  useEffect(() => {
    buildScrollRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [buildLines])

  useEffect(() => {
    deployScrollRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [deployLines])

  const isDone = streamState === 'done'
  const isSuccess = status === 'success'
  const isFailed = status === 'failed'

  // Show deploy section if the backend has sent any deploy logs OR we're in the deploy phase.
  const showDeploy = deployLines.length > 0 || phase === 'deploy'

  // Build section is "active" when we're in the building phase.
  const buildActive = phase === 'build' && streamState === 'streaming'
  // Deploy section is "active" when we're in the deploy phase.
  const deployActive = phase === 'deploy' && streamState === 'streaming'

  // Build succeeded if deploy ran (we got past build) or overall success without deploy.
  // Build failed if overall failed and no deploy was attempted.
  const buildPhaseSuccess = showDeploy || isSuccess
  const buildPhaseFailed = isFailed && !showDeploy

  // Deploy succeeded/failed maps directly to the overall job result.
  const deployPhaseSuccess = isSuccess
  const deployPhaseFailed = isFailed

  return (
    <div className="flex flex-col gap-3">
      {streamState === 'error' && (
        <div className="flex items-center gap-1.5 rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-3 text-xs text-red-400">
          <AlertCircle className="h-3.5 w-3.5 shrink-0" />
          Connection lost. The job may still be running — refresh to check.
        </div>
      )}

      <PhaseLogSection
        title="Build logs"
        lines={buildLines}
        isActive={buildActive}
        streamDone={isDone}
        isPhaseSuccess={buildPhaseSuccess}
        isPhaseFailed={buildPhaseFailed}
        successLabel="Build succeeded"
        scrollRef={buildScrollRef}
      />

      {showDeploy && (
        <PhaseLogSection
          title="Deploy logs"
          lines={deployLines}
          isActive={deployActive}
          streamDone={isDone}
          isPhaseSuccess={deployPhaseSuccess}
          isPhaseFailed={deployPhaseFailed}
          successLabel="Deploy succeeded"
          scrollRef={deployScrollRef}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function JobDetailPage(): React.JSX.Element {
  const { id, jobId } = useParams<{ id: string; jobId: string }>()
  const router = useRouter()

  const { status, phase, buildLines, deployLines, streamState, streamError } = useJobStream(jobId)

  if (streamState === 'error' && status === null) {
    return (
      <main className="mx-auto max-w-4xl p-8">
        <div className="mb-8 flex items-center gap-4">
          <Link href={`/projects/${id}/jobs`} className="text-zinc-500 hover:text-zinc-300 transition-colors">
            ← Build history
          </Link>
        </div>
        <div className="rounded-xl border border-zinc-800 p-12 text-center">
          <AlertCircle className="mx-auto mb-3 h-8 w-8 text-zinc-600" />
          <p className="text-zinc-500">
            Failed to load job.{' '}
            <button
              onClick={() => router.refresh()}
              className="text-zinc-300 underline hover:text-white transition-colors"
            >
              Retry
            </button>
          </p>
        </div>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-4xl p-8">
      <div className="mb-8 flex items-center gap-4">
        <Link
          href={`/projects/${id}/jobs`}
          className="text-zinc-500 hover:text-zinc-300 transition-colors"
        >
          ← Build history
        </Link>
        <h1 className="text-2xl font-bold">Job</h1>
        <span className="font-mono text-sm text-zinc-500">{jobId.slice(0, 8)}</span>
      </div>

      <StatusCard status={status} streamState={streamState} streamError={streamError} />
      <LogPanel
        buildLines={buildLines}
        deployLines={deployLines}
        streamState={streamState}
        status={status}
        phase={phase}
      />
    </main>
  )
}
