'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import StatusBadge from '@/components/StatusBadge'
import type { JobStatus } from '@/lib/types'

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

const TERMINAL = new Set<JobStatus>(['success', 'failed'])

export default function JobDetailPage() {
  const { id, jobId } = useParams<{ id: string; jobId: string }>()
  const router = useRouter()
  const [status, setStatus] = useState<JobStatus | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    const es = new EventSource(`${API_URL}/api/jobs/${jobId}/events`, {
      withCredentials: true,
    })

    es.onmessage = (e: MessageEvent) => {
      const data = JSON.parse(e.data) as { status: JobStatus }
      setStatus(data.status)
      if (TERMINAL.has(data.status)) {
        es.close()
      }
    }

    es.onerror = () => {
      es.close()
      if (status === null) {
        setError(true)
      }
    }

    return () => es.close()
  }, [jobId]) // eslint-disable-line react-hooks/exhaustive-deps

  if (error) {
    return (
      <main className="mx-auto max-w-3xl p-8">
        <div className="mb-8 flex items-center gap-4">
          <Link href={`/projects/${id}/jobs`} className="text-zinc-500 hover:text-zinc-300">
            ← Build history
          </Link>
        </div>
        <div className="rounded-xl border border-zinc-800 p-12 text-center text-zinc-500">
          Failed to load job.{' '}
          <button onClick={() => router.refresh()} className="underline hover:text-zinc-300">
            Retry
          </button>
        </div>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl p-8">
      <div className="mb-8 flex items-center gap-4">
        <Link href={`/projects/${id}/jobs`} className="text-zinc-500 hover:text-zinc-300">
          ← Build history
        </Link>
        <h1 className="text-2xl font-bold">Job</h1>
        <span className="font-mono text-sm text-zinc-500">{jobId.slice(0, 8)}</span>
      </div>

      <div className="rounded-xl border border-zinc-800 p-8">
        <div className="flex items-center gap-3">
          <span className="text-sm text-zinc-400">Status</span>
          {status === null ? (
            <span className="text-sm text-zinc-500">Connecting...</span>
          ) : (
            <StatusBadge status={status} />
          )}
        </div>
      </div>
    </main>
  )
}