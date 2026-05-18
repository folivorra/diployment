'use client'

import { useEffect, useState } from 'react'
import type { Job, JobStatus } from '@/lib/types'

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'
const TERMINAL = new Set<JobStatus>(['success', 'failed'])

export function useJobStatuses(jobs: Job[]): Map<string, JobStatus> {
  const [statuses, setStatuses] = useState<Map<string, JobStatus>>(
    () => new Map(jobs.map((j) => [j.id, j.status])),
  )

  useEffect(() => {
    const active = jobs.filter((j) => !TERMINAL.has(j.status))
    if (active.length === 0) return

    const sources = active.map((job) => {
      const es = new EventSource(`${API_URL}/api/jobs/${job.id}/events`, {
        withCredentials: true,
      })

      es.onmessage = (e: MessageEvent) => {
        const { status } = JSON.parse(e.data) as { status: JobStatus }
        setStatuses((prev) => new Map(prev).set(job.id, status))
        if (TERMINAL.has(status)) es.close()
      }

      es.onerror = () => es.close()

      return es
    })

    return () => sources.forEach((es) => es.close())
  }, [jobs.map((j) => j.id).join(',')])  // eslint-disable-line react-hooks/exhaustive-deps

  return statuses
}