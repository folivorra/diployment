'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { motion, AnimatePresence } from 'motion/react'
import { ExternalLink, GitCommit, Loader2 } from 'lucide-react'
import { apiFetch, ApiError, getApiUrl } from '@/lib/api'
import { useJobStream } from '@/lib/hooks/use-job-stream'
import { Breadcrumbs } from '@/components/layout/breadcrumbs'
import { StatusBadge } from '@/components/jobs/status-badge'
import { PhaseTimeline } from '@/components/jobs/phase-timeline'
import { LogPane } from '@/components/jobs/log-pane'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipProvider } from '@/components/ui/tooltip'
import { formatDateTime, formatDuration, githubCommitUrl, shortSha, timeAgo } from '@/lib/format'
import type { Job, JobStatus, Phase, Project } from '@/lib/types'

const TERMINAL = new Set<JobStatus>(['success', 'failed'])

export default function JobDetailPage() {
  const { id, jobId } = useParams<{ id: string; jobId: string }>()
  const router = useRouter()

  const [project, setProject] = useState<Project | null>(null)
  const [job, setJob] = useState<Job | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      apiFetch<Project[]>('/api/projects'),
      apiFetch<Job[]>(`/api/projects/${id}/jobs`),
    ])
      .then(([projects, jobs]) => {
        const proj = projects?.find((p) => p.id === id) ?? null
        const jb = jobs?.find((j) => j.id === jobId) ?? null
        setProject(proj)
        setJob(jb)
        if (!jb) router.replace(`/projects/${id}/jobs`)
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          if (err.status === 401 || err.status === 400) router.replace('/')
          else router.replace('/dashboard')
        }
      })
      .finally(() => setLoading(false))
  }, [id, jobId, router])

  if (loading || !job || !project) {
    return (
      <>
        <Skeleton className="mb-6 h-5 w-64" />
        <Skeleton className="mb-4 h-32 w-full" />
        <Skeleton className="h-80 w-full" />
      </>
    )
  }

  const wasTerminal = TERMINAL.has(job.status)

  return (
    <>
      <div className="mb-2">
        <Breadcrumbs
          items={[
            { label: 'Дашборд', href: '/dashboard' },
            { label: project.repo_full_name, href: `/projects/${project.id}`, mono: true },
            { label: 'Джобы', href: `/projects/${id}/jobs` },
            { label: `#${shortSha(job.commit_sha)}`, mono: true },
          ]}
        />
      </div>

      {wasTerminal ? (
        <TerminalJobView project={project} job={job} />
      ) : (
        <LiveJobView project={project} job={job} />
      )}
    </>
  )
}

// ---------------------------------------------------------------------------
// Live job view — SSE stream
// ---------------------------------------------------------------------------

function LiveJobView({ project, job }: { project: Project; job: Job }) {
  const { status, phase, buildLines, deployLines, streamState } = useJobStream(job.id)

  const currentStatus = status ?? job.status
  const hasDeploy = deployLines.length > 0 || phase === 'deploy' || currentStatus === 'deploying' || currentStatus === 'success'
  const isStreaming = streamState === 'streaming'
  const isDone = streamState === 'done'

  return (
    <TooltipProvider delayDuration={300}>
      <JobHeader project={project} job={job} status={currentStatus} phase={phase} hasDeploy={hasDeploy} />

      {streamState === 'connecting' && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/40 px-4 py-2.5 text-xs text-zinc-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Подключаемся к стриму…
        </div>
      )}

      <div className="flex flex-col gap-3">
        <LogPane
          title="Сборка"
          lines={buildLines}
          status={resolveBuildStatus(currentStatus, phase, isStreaming, isDone, hasDeploy)}
        />

        <AnimatePresence>
          {hasDeploy && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
              className="overflow-hidden"
            >
              <LogPane
                title="Деплой"
                lines={deployLines}
                status={resolveDeployStatus(currentStatus, phase, isStreaming)}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </TooltipProvider>
  )
}

// ---------------------------------------------------------------------------
// Terminal job view — persistent logs from MinIO
// ---------------------------------------------------------------------------

function TerminalJobView({ project, job }: { project: Project; job: Job }) {
  const [buildLog, setBuildLog] = useState<string[] | null>(null)
  const [deployLog, setDeployLog] = useState<string[] | null>(null)
  const [buildErr, setBuildErr] = useState(false)
  const [deployErr, setDeployErr] = useState(false)

  // The deploy phase has been entered if the deployer recorded a start. This
  // covers failures that happen before the first log line (e.g. SSH dial),
  // which would otherwise be misattributed to the build phase.
  const hasDeploy = Boolean(job.deploy_started_at)

  useEffect(() => {
    fetchPhase(job.id, 'build').then(setBuildLog).catch(() => setBuildErr(true))
    if (hasDeploy) {
      fetchPhase(job.id, 'deploy').then(setDeployLog).catch(() => setDeployErr(true))
    }
  }, [job.id, hasDeploy])

  return (
    <TooltipProvider delayDuration={300}>
      <JobHeader project={project} job={job} status={job.status} phase={null} hasDeploy={hasDeploy} />

      <div className="flex flex-col gap-3">
        <LogPane
          title="Сборка"
          lines={buildLog ?? []}
          status={job.status === 'failed' && !hasDeploy ? 'failed' : 'success'}
          emptyHint={buildErr ? 'Не удалось загрузить сохранённый лог.' : 'Загрузка лога…'}
        />

        {hasDeploy && (
          <LogPane
            title="Деплой"
            lines={deployLog ?? []}
            status={job.status === 'success' ? 'success' : 'failed'}
            emptyHint={deployErr ? 'Не удалось загрузить сохранённый лог.' : 'Загрузка лога…'}
          />
        )}
      </div>
    </TooltipProvider>
  )
}

type LogStatus = 'pending' | 'streaming' | 'success' | 'failed'

function resolveBuildStatus(
  status: JobStatus,
  phase: Phase | null,
  isStreaming: boolean,
  isDone: boolean,
  hasDeploy: boolean,
): LogStatus {
  if (phase === 'build' && isStreaming) return 'streaming'
  if (status === 'failed' && !hasDeploy) return 'failed'
  if (status === 'success' || status === 'deploying' || hasDeploy || isDone) return 'success'
  return 'pending'
}

function resolveDeployStatus(status: JobStatus, phase: Phase | null, isStreaming: boolean): LogStatus {
  if (phase === 'deploy' && isStreaming) return 'streaming'
  if (status === 'success') return 'success'
  if (status === 'failed') return 'failed'
  return 'pending'
}

async function fetchPhase(jobId: string, phase: Phase): Promise<string[]> {
  const res = await fetch(`${getApiUrl()}/api/jobs/${jobId}/logs/${phase}`, {
    credentials: 'include',
  })
  if (!res.ok) throw new ApiError(res.status)
  const text = await res.text()
  return text.split('\n').filter((l) => l.length > 0)
}

// ---------------------------------------------------------------------------
// Shared header card
// ---------------------------------------------------------------------------

interface JobHeaderProps {
  project: Project
  job: Job
  status: JobStatus
  phase: Phase | null
  hasDeploy: boolean
}

function JobHeader({ project, job, status, phase, hasDeploy }: JobHeaderProps) {
  const commitUrl = githubCommitUrl(project.clone_url, job.commit_sha)
  const duration = formatDuration(
    job.build_started_at ?? job.created_at,
    job.deploy_finished_at ?? job.build_finished_at,
  )
  const isLive = status === 'building' || status === 'deploying' || status === 'pending'

  return (
    <motion.div
      initial={{ opacity: 0, y: -4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="mb-6 overflow-hidden rounded-xl border border-zinc-800/80 bg-zinc-900/40 backdrop-blur-sm"
    >
      <div className="flex flex-wrap items-start justify-between gap-6 px-6 py-5">
        <div className="flex min-w-0 flex-col gap-3">
          <div className="flex items-center gap-3">
            <StatusBadge status={status} />
            {isLive && (
              <span className="flex items-center gap-1.5 text-[11px] text-zinc-500">
                <span className="relative inline-flex h-1.5 w-1.5">
                  <span className="h-1.5 w-1.5 rounded-full bg-indigo-400 animate-pulse-dot" />
                </span>
                В реальном времени
              </span>
            )}
          </div>

          <div className="flex items-center gap-2.5 text-sm">
            <GitCommit className="h-4 w-4 text-zinc-500" />
            {commitUrl ? (
              <a
                href={commitUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-1.5 font-mono text-zinc-200 transition-colors hover:text-indigo-300"
              >
                {shortSha(job.commit_sha)}
                <ExternalLink className="h-3 w-3 opacity-0 transition-opacity group-hover:opacity-100" />
              </a>
            ) : (
              <span className="font-mono text-zinc-200">{shortSha(job.commit_sha)}</span>
            )}
            <span className="text-zinc-200">·</span>
            <span className="truncate text-zinc-300">{job.commit_msg || <span className="text-zinc-600">(без сообщения)</span>}</span>
          </div>

          <div className="flex items-center gap-3 text-xs text-zinc-500">
            <Tooltip content={formatDateTime(job.created_at)}>
              <span>Старт {timeAgo(job.created_at)}</span>
            </Tooltip>
            {duration && (
              <>
                <span className="text-zinc-700">•</span>
                <span>{duration}</span>
              </>
            )}
            <span className="text-zinc-700">•</span>
            <Link
              href={`/projects/${project.id}/jobs`}
              className="transition-colors hover:text-zinc-300"
            >
              {project.repo_full_name}
            </Link>
          </div>

        </div>

        <div className="shrink-0">
          <PhaseTimeline status={status} phase={phase} hasDeploy={hasDeploy} />
        </div>
      </div>
    </motion.div>
  )
}
