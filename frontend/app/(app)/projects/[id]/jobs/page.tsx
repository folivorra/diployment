'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { GitBranch, Inbox } from 'lucide-react'
import { apiFetch, ApiError } from '@/lib/api'
import { Breadcrumbs } from '@/components/layout/breadcrumbs'
import { PageHeader } from '@/components/layout/page-header'
import { JobRow } from '@/components/jobs/job-row'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import type { Job, Project } from '@/lib/types'

const POLL_INTERVAL_MS = 5000

export default function JobsPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [jobs, setJobs] = useState<Job[]>([])
  const [project, setProject] = useState<Project | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      apiFetch<Project[]>('/api/projects'),
      apiFetch<Job[]>(`/api/projects/${id}/jobs`),
    ])
      .then(([projects, j]) => {
        setProject(projects?.find((p) => p.id === id) ?? null)
        setJobs(j ?? [])
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          if (err.status === 401 || err.status === 400) router.replace('/')
          else if (err.status === 403 || err.status === 404) router.replace('/dashboard')
        }
      })
      .finally(() => setLoading(false))
  }, [id, router])

  // Poll the jobs list to pick up new jobs and status transitions of in-flight
  // ones. SSE is reserved for the single-job detail page; the list view stays
  // cheap.
  useEffect(() => {
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    async function tick() {
      if (cancelled) return
      if (document.visibilityState === 'visible') {
        try {
          const fresh = (await apiFetch<Job[]>(`/api/projects/${id}/jobs`)) ?? []
          if (!cancelled) {
            setJobs((prev) => (sameJobs(prev, fresh) ? prev : fresh))
          }
        } catch {
          // Swallow transient errors — next tick will retry.
        }
      }
      if (!cancelled) timer = setTimeout(tick, POLL_INTERVAL_MS)
    }

    timer = setTimeout(tick, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [id])

  if (loading) {
    return (
      <>
        <Skeleton className="mb-6 h-6 w-64" />
        <Skeleton className="mb-3 h-9 w-48" />
        <div className="mt-8 space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      </>
    )
  }

  return (
    <>
      <div className="mb-2">
        <Breadcrumbs
          items={[
            { label: 'Дашборд', href: '/dashboard' },
            { label: project?.repo_full_name ?? '…', href: project ? `/projects/${project.id}` : undefined, mono: true },
            { label: 'Джобы' },
          ]}
        />
      </div>

      <PageHeader
        title="Джобы"
        description="Каждый коммит, попавший в эту ветку."
        meta={
          project && (
            <>
              <Badge tone="neutral" className="font-mono">
                <GitBranch className="h-3 w-3" />
                {project.branch}
              </Badge>
              <Link
                href={`/projects/${id}`}
                className="text-xs text-zinc-500 transition-colors hover:text-zinc-300"
              >
                Настройки →
              </Link>
            </>
          )
        }
      />

      {jobs.length === 0 ? (
        <EmptyState
          icon={<Inbox className="h-6 w-6" />}
          title="Джоб пока нет"
          description="Сделай push в указанную ветку, чтобы запустить первую джобу."
        />
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-800/80 bg-zinc-900/30">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-zinc-800 bg-zinc-900/60 text-[10px] font-semibold uppercase tracking-wider text-zinc-500">
                <th className="py-3 pl-5 pr-4">Статус</th>
                <th className="px-4 py-3">Коммит</th>
                <th className="px-4 py-3">Сообщение</th>
                <th className="px-4 py-3">Длительность</th>
                <th className="px-4 py-3 pr-5 text-right">Когда</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job, i) => (
                <JobRow
                  key={job.id}
                  job={job}
                  projectId={id}
                  index={i}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}

// Returns true if the list is unchanged for rendering purposes: same length,
// same ordered ids, and same status for each job.
function sameJobs(a: Job[], b: Job[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i].id !== b[i].id || a[i].status !== b[i].status) return false
  }
  return true
}
