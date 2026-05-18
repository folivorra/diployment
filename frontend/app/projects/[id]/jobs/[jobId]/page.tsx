'use client'

import React, { useEffect, useRef, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { CheckCircle2, XCircle, Loader2, Clock, AlertCircle } from 'lucide-react'
import { getApiUrl } from '@/lib/api'
import StatusBadge from '@/components/StatusBadge'
import type { JobStatus } from '@/lib/types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface StatusEvent {
  status: JobStatus
  error: string
}

interface LogEvent {
  line: string
  done?: boolean
}

type StreamState = 'connecting' | 'streaming' | 'done' | 'error'

// ---------------------------------------------------------------------------
// Hook: useJobStream
// Connects to GET /api/jobs/:id/events, handles both `status` and `log`
// named SSE events, and manages cleanup on unmount.
// ---------------------------------------------------------------------------

interface JobStream {
  status: JobStatus | null
  logLines: string[]
  streamState: StreamState
  streamError: string | null
}

function useJobStream(jobId: string): JobStream {
  const [status, setStatus] = useState<JobStatus | null>(null)
  const [logLines, setLogLines] = useState<string[]>([])
  const [streamState, setStreamState] = useState<StreamState>('connecting')
  const [streamError, setStreamError] = useState<string | null>(null)

  // Track terminal conditions independently so we can close when both are met.
  const gotTerminalStatus = useRef(false)
  const gotLogsDone = useRef(false)
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    gotTerminalStatus.current = false
    gotLogsDone.current = false
    setStatus(null)
    setLogLines([])
    setStreamState('connecting')
    setStreamError(null)

    const es = new EventSource(`${getApiUrl()}/api/jobs/${jobId}/events`, {
      withCredentials: true,
    })
    esRef.current = es

    function maybeClose(): void {
      if (gotTerminalStatus.current && gotLogsDone.current) {
        es.close()
        setStreamState('done')
      }
    }

    es.addEventListener('status', (e: MessageEvent) => {
      const data = JSON.parse(e.data as string) as StatusEvent
      setStatus(data.status)
      setStreamState('streaming')

      const isTerminal = data.status === 'success' || data.status === 'failed'
      if (isTerminal) {
        gotTerminalStatus.current = true
        setStreamError(data.error || null)
        maybeClose()
      }
    })

    es.addEventListener('log', (e: MessageEvent) => {
      const data = JSON.parse(e.data as string) as LogEvent
      if (data.done) {
        gotLogsDone.current = true
        maybeClose()
      } else {
        setLogLines((prev) => [...prev, data.line])
      }
    })

    // onerror fires on network drop or when the server closes the connection
    // after a clean finish. Only treat it as a real error if we never received
    // a terminal status (i.e. we didn't finish cleanly).
    es.onerror = () => {
      es.close()
      esRef.current = null
      setStatus((current) => {
        const isFinished = current === 'success' || current === 'failed'
        if (!isFinished) {
          setStreamState('error')
        } else {
          // Server closed connection after a clean finish — mark done.
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

  return { status, logLines, streamState, streamError }
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
        {streamState === 'streaming' && status === 'running' && (
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

interface LogPanelProps {
  logLines: string[]
  streamState: StreamState
  status: JobStatus | null
}

function LogPanel({ logLines, streamState, status }: LogPanelProps): React.JSX.Element {
  const logEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logLines])

  const isEmpty = logLines.length === 0

  return (
    <div className="rounded-xl border border-zinc-800 overflow-hidden">
      <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900 px-5 py-3">
        <span className="text-sm font-medium text-zinc-400">Build logs</span>
        {streamState === 'streaming' && (
          <span className="flex items-center gap-1.5 text-xs text-zinc-600">
            <Loader2 className="h-3 w-3 animate-spin" />
            Streaming
          </span>
        )}
        {streamState === 'done' && (
          <span className="flex items-center gap-1.5 text-xs text-zinc-600">
            <CheckCircle2 className="h-3 w-3" />
            Complete
          </span>
        )}
        {streamState === 'error' && (
          <span className="flex items-center gap-1.5 text-xs text-red-500">
            <AlertCircle className="h-3 w-3" />
            Connection lost
          </span>
        )}
      </div>

      <div className="h-[520px] overflow-y-auto bg-zinc-950 p-4 font-mono text-xs leading-5">
        {isEmpty ? (
          <span className="text-zinc-600">
            {streamState === 'connecting' ? 'Waiting for logs…' : 'No logs.'}
          </span>
        ) : (
          logLines.map((line, i) => (
            <div
              key={i}
              className="whitespace-pre-wrap break-all text-zinc-300"
            >
              {line.replace(/\n$/, '')}
            </div>
          ))
        )}

        {/* Terminal finish marker */}
        {streamState === 'done' && !isEmpty && (
          <div className="mt-3 flex items-center gap-2 border-t border-zinc-800 pt-3">
            {status === 'success' ? (
              <>
                <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-400" />
                <span className="text-emerald-400">Build succeeded</span>
              </>
            ) : status === 'failed' ? (
              <>
                <XCircle className="h-3.5 w-3.5 shrink-0 text-red-400" />
                <span className="text-red-400">Build failed</span>
              </>
            ) : (
              <>
                <Clock className="h-3.5 w-3.5 shrink-0 text-zinc-500" />
                <span className="text-zinc-500">Stream ended</span>
              </>
            )}
          </div>
        )}

        <div ref={logEndRef} />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function JobDetailPage(): React.JSX.Element {
  const { id, jobId } = useParams<{ id: string; jobId: string }>()
  const router = useRouter()

  const { status, logLines, streamState, streamError } = useJobStream(jobId)

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
      <LogPanel logLines={logLines} streamState={streamState} status={status} />
    </main>
  )
}
