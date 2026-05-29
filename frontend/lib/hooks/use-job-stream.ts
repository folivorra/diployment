'use client'

import { useEffect, useRef, useState } from 'react'
import { getApiUrl } from '@/lib/api'
import type { JobStatus, Phase } from '@/lib/types'

export interface StatusEvent {
  status: JobStatus
  phase: Phase
  error?: string
}

export interface LogEvent {
  line: string
  phase: Phase
  done?: boolean
}

export type StreamState = 'connecting' | 'streaming' | 'done' | 'error'

export interface JobStream {
  status: JobStatus | null
  phase: Phase | null
  buildLines: string[]
  deployLines: string[]
  streamState: StreamState
  streamError: string | null
}

// Multiplexes named SSE events (`status` + `log`) from /api/jobs/:id/events
// across two phases (build / deploy). Closing logic mirrors the backend protocol:
//   - deploy `done` sentinel  → always close
//   - build  `done` sentinel  → only close if terminal status already arrived
export function useJobStream(jobId: string): JobStream {
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
          es.close()
          esRef.current = null
          setStreamState('done')
        } else {
          buildDone.current = true
          maybeClose()
        }
        return
      }
      if (data.phase === 'deploy') {
        setDeployLines((prev) => [...prev, data.line])
      } else {
        setBuildLines((prev) => [...prev, data.line])
      }
    })

    es.onerror = () => {
      es.close()
      esRef.current = null
      setStatus((current) => {
        const finished = current === 'success' || current === 'failed'
        setStreamState(finished ? 'done' : 'error')
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
