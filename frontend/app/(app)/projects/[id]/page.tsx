'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { motion } from 'motion/react'
import { Activity, FileText, GitBranch, Lock, Server, Settings2, ExternalLink, ArrowRight } from 'lucide-react'
import { apiFetch, ApiError } from '@/lib/api'
import { Breadcrumbs } from '@/components/layout/breadcrumbs'
import { PageHeader } from '@/components/layout/page-header'
import { Card, CardBody, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/jobs/status-badge'
import { Tooltip, TooltipProvider } from '@/components/ui/tooltip'
import { formatDateTime, formatDuration, githubCommitUrl, shortSha, timeAgo } from '@/lib/format'
import type { Job, JobStatus, Project } from '@/lib/types'

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [project, setProject] = useState<Project | null>(null)
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      apiFetch<Project[]>('/api/projects'),
      apiFetch<Job[]>(`/api/projects/${id}/jobs`),
    ])
      .then(([projects, j]) => {
        const proj = projects?.find((p) => p.id === id) ?? null
        if (!proj) {
          router.replace('/dashboard')
          return
        }
        setProject(proj)
        setJobs(j ?? [])
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          if (err.status === 401 || err.status === 400) router.replace('/')
          else router.replace('/dashboard')
        }
      })
      .finally(() => setLoading(false))
  }, [id, router])

  if (loading || !project) {
    return (
      <>
        <Skeleton className="mb-4 h-5 w-64" />
        <Skeleton className="mb-3 h-9 w-72" />
        <div className="mt-8 grid grid-cols-1 gap-4 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      </>
    )
  }

  const latest = jobs[0]
  const total = jobs.length
  const successes = jobs.filter((j) => j.status === 'success').length
  const successRate = total > 0 ? Math.round((successes / total) * 100) : null

  const completedDurations = jobs
    .map((j) => {
      const ms = j.deploy_finished_at && j.build_started_at
        ? new Date(j.deploy_finished_at).getTime() - new Date(j.build_started_at).getTime()
        : null
      return ms !== null && ms > 0 ? ms : null
    })
    .filter((v): v is number => v !== null)
  const avgDuration =
    completedDurations.length > 0
      ? completedDurations.reduce((a, b) => a + b, 0) / completedDurations.length
      : null

  return (
    <TooltipProvider delayDuration={300}>
      <div className="mb-2">
        <Breadcrumbs
          items={[
            { label: 'Дашборд', href: '/dashboard' },
            { label: project.repo_full_name, mono: true },
          ]}
        />
      </div>

      <PageHeader
        title={<span className="font-mono">{project.repo_full_name}</span>}
        description="Конфигурация и последняя активность по проекту."
        meta={
          <>
            <Badge tone="neutral" className="font-mono">
              <GitBranch className="h-3 w-3" />
              {project.branch}
            </Badge>
            <span className="text-xs text-zinc-500">
              Импортирован {timeAgo(project.created_at)}
            </span>
          </>
        }
        actions={
          <Link href={`/projects/${project.id}/jobs`}>
            <Button variant="secondary">
              Открыть джобы
              <ArrowRight className="h-3.5 w-3.5" />
            </Button>
          </Link>
        }
      />

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">
            <FileText className="mr-1.5 h-3.5 w-3.5" />
            Обзор
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings2 className="mr-1.5 h-3.5 w-3.5" />
            Настройки
          </TabsTrigger>
          <TabsTrigger value="activity">
            <Activity className="mr-1.5 h-3.5 w-3.5" />
            Активность
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <motion.div
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.28 }}
            className="grid grid-cols-1 gap-4 md:grid-cols-2"
          >
            <Card>
              <CardHeader>
                <CardTitle>Репозиторий</CardTitle>
              </CardHeader>
              <CardBody className="flex flex-col gap-2.5">
                <KV label="Имя" value={<span className="font-mono">{project.repo_full_name}</span>} />
                <KV label="Ветка" value={<span className="font-mono">{project.branch}</span>} />
                <KV
                  label="Clone URL"
                  value={
                    <a
                      href={project.clone_url.replace(/\.git$/, '')}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 truncate font-mono text-zinc-300 transition-colors hover:text-indigo-300"
                    >
                      {project.clone_url}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  }
                />
                <KV
                  label="Webhook"
                  value={
                    <span className="font-mono text-zinc-400">#{project.webhook_id}</span>
                  }
                />
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Последняя джоба</CardTitle>
              </CardHeader>
              <CardBody>
                {latest ? (
                  <LatestJobCard projectId={project.id} job={latest} cloneUrl={project.clone_url} />
                ) : (
                  <p className="text-sm text-zinc-500">Джоб пока нет.</p>
                )}
              </CardBody>
            </Card>

            <Card className="md:col-span-2">
              <CardHeader>
                <CardTitle>Статистика</CardTitle>
              </CardHeader>
              <CardBody>
                <div className="grid grid-cols-2 gap-6 md:grid-cols-4">
                  <Stat label="Всего джоб" value={String(total)} />
                  <Stat label="Успешных" value={String(successes)} tone="success" />
                  <Stat label="Доля успеха" value={successRate !== null ? `${successRate}%` : '—'} />
                  <Stat label="Средняя длительность" value={avgDuration !== null ? formatDurationMs(avgDuration) : '—'} />
                </div>
              </CardBody>
            </Card>
          </motion.div>
        </TabsContent>

        <TabsContent value="settings">
          <motion.div
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.28 }}
            className="grid grid-cols-1 gap-4 md:grid-cols-2"
          >
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Server className="h-3.5 w-3.5 text-indigo-300" />
                  SSH-подключение
                </CardTitle>
              </CardHeader>
              <CardBody className="flex flex-col gap-2.5">
                <KV label="Хост" value={<span className="font-mono">{project.ssh_host}</span>} />
                <KV label="Порт" value={<span className="font-mono">{project.ssh_port}</span>} />
                <KV label="Пользователь" value={<span className="font-mono">{project.ssh_user}</span>} />
                <KV
                  label="Приватный ключ"
                  value={
                    <span className="inline-flex items-center gap-1.5 text-zinc-400">
                      <Lock className="h-3.5 w-3.5" />
                      Зашифрован при импорте
                    </span>
                  }
                />
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Settings2 className="h-3.5 w-3.5 text-indigo-300" />
                  Деплой
                </CardTitle>
              </CardHeader>
              <CardBody className="flex flex-col gap-2.5">
                <KV label="Рабочая директория" value={<span className="font-mono">{project.deploy_workdir}</span>} />
                <KV
                  label="Команда перезапуска"
                  value={<code className="block truncate rounded bg-zinc-900 px-2 py-1 font-mono text-[11px] text-zinc-300">{project.deploy_restart_cmd}</code>}
                />
              </CardBody>
            </Card>

            <Card className="md:col-span-2">
              <CardBody>
                <p className="text-xs text-zinc-500">
                  Настройки фиксируются при импорте. Чтобы поменять — переимпортируй репозиторий; редактирования на месте пока нет.
                </p>
              </CardBody>
            </Card>
          </motion.div>
        </TabsContent>

        <TabsContent value="activity">
          <motion.div
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.28 }}
          >
            <Card>
              <CardHeader>
                <CardTitle>Последние {Math.min(5, jobs.length)} из {jobs.length}</CardTitle>
                <Link
                  href={`/projects/${project.id}/jobs`}
                  className="text-xs text-zinc-400 transition-colors hover:text-zinc-200"
                >
                  Все →
                </Link>
              </CardHeader>
              <CardBody className="!p-0">
                {jobs.length === 0 ? (
                  <p className="px-5 py-8 text-center text-sm text-zinc-500">Джоб пока нет.</p>
                ) : (
                  <ul>
                    {jobs.slice(0, 5).map((job) => (
                      <ActivityRow key={job.id} projectId={project.id} job={job} />
                    ))}
                  </ul>
                )}
              </CardBody>
            </Card>
          </motion.div>
        </TabsContent>
      </Tabs>
    </TooltipProvider>
  )
}

function KV({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 text-sm">
      <span className="text-[11px] uppercase tracking-wider text-zinc-500">{label}</span>
      <span className="min-w-0 truncate text-right text-zinc-200">{value}</span>
    </div>
  )
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: 'success' }) {
  return (
    <div>
      <p className="text-[10px] font-medium uppercase tracking-wider text-zinc-500">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${tone === 'success' ? 'text-emerald-300' : 'text-zinc-50'}`}>
        {value}
      </p>
    </div>
  )
}

function LatestJobCard({ projectId, job, cloneUrl }: { projectId: string; job: Job; cloneUrl: string }) {
  const url = githubCommitUrl(cloneUrl, job.commit_sha)
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2.5">
        <StatusBadge status={job.status} />
        {url ? (
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 font-mono text-xs text-zinc-300 transition-colors hover:text-indigo-300"
          >
            {shortSha(job.commit_sha)}
            <ExternalLink className="h-3 w-3" />
          </a>
        ) : (
          <span className="font-mono text-xs text-zinc-300">{shortSha(job.commit_sha)}</span>
        )}
      </div>
      <p className="line-clamp-2 text-sm text-zinc-200">
        {job.commit_msg || <span className="text-zinc-600">(без сообщения)</span>}
      </p>
      <div className="flex items-center justify-between text-xs text-zinc-500">
        <Tooltip content={formatDateTime(job.created_at)}>
          <span>{timeAgo(job.created_at)}</span>
        </Tooltip>
        <Link
          href={`/projects/${projectId}/jobs/${job.id}`}
          className="text-zinc-400 transition-colors hover:text-indigo-300"
        >
          Открыть джобу →
        </Link>
      </div>
    </div>
  )
}

function ActivityRow({ projectId, job }: { projectId: string; job: Job }) {
  const start = job.build_started_at ?? job.created_at
  const end = job.deploy_finished_at ?? job.build_finished_at
  const duration = formatDuration(start, end)

  return (
    <li>
      <Link
        href={`/projects/${projectId}/jobs/${job.id}`}
        className="flex items-center gap-3 border-b border-zinc-800/60 px-5 py-3 transition-colors last:border-b-0 hover:bg-zinc-900/40"
      >
        <StatusBadge status={job.status as JobStatus} size="sm" />
        <span className="font-mono text-xs text-zinc-400">{shortSha(job.commit_sha)}</span>
        <span className="min-w-0 flex-1 truncate text-sm text-zinc-300">{job.commit_msg}</span>
        <span className="text-xs text-zinc-500">{duration ?? '—'}</span>
        <Tooltip content={formatDateTime(job.created_at)}>
          <span className="text-xs text-zinc-500">{timeAgo(job.created_at)}</span>
        </Tooltip>
      </Link>
    </li>
  )
}

function formatDurationMs(ms: number): string {
  const totalSec = Math.round(ms / 1000)
  if (totalSec < 60) return `${totalSec}с`
  const mins = Math.floor(totalSec / 60)
  const secs = totalSec % 60
  if (mins < 60) return secs ? `${mins}м ${secs}с` : `${mins}м`
  const hrs = Math.floor(mins / 60)
  const rmins = mins % 60
  return rmins ? `${hrs}ч ${rmins}м` : `${hrs}ч`
}
