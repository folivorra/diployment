'use client'

import { useEffect, useState } from 'react'
import { getApiUrl } from '@/lib/api'
import type { Job, JobStatus, Phase } from '@/lib/types'

const TERMINAL = new Set<JobStatus>(['success', 'failed'])

export function useJobStatuses(jobs: Job[]): Map<string, JobStatus> {
  const [statuses, setStatuses] = useState<Map<string, JobStatus>>(
    () => new Map(jobs.map((j) => [j.id, j.status])),
  )

  useEffect(() => {
    const active = jobs.filter((j) => !TERMINAL.has(j.status))
    if (active.length === 0) return

    const sources = active.map((job) => {
      const es = new EventSource(`${getApiUrl()}/api/jobs/${job.id}/events`, {
        withCredentials: true,
      })

      es.addEventListener('status', (e: MessageEvent) => {
        const { status } = JSON.parse(e.data) as { status: JobStatus; phase: Phase; error?: string }
        setStatuses((prev) => new Map(prev).set(job.id, status))
        if (TERMINAL.has(status)) es.close()
      })

      es.onerror = () => es.close()

      return es
    })

    return () => sources.forEach((es) => es.close())
  }, [jobs.map((j) => j.id).join(',')]) // eslint-disable-line react-hooks/exhaustive-deps

  return statuses
}
