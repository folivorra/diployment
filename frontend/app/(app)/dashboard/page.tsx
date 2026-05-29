'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { FolderGit2, Plus } from 'lucide-react'
import { ApiError, apiFetch } from '@/lib/api'
import { PageHeader } from '@/components/layout/page-header'
import { ProjectCard } from '@/components/projects/project-card'
import { EmptyState } from '@/components/ui/empty-state'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { Job, Project } from '@/lib/types'

export default function DashboardPage() {
  const router = useRouter()
  const [projects, setProjects] = useState<Project[]>([])
  const [lastJobs, setLastJobs] = useState<Map<string, Job | undefined>>(new Map())
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    apiFetch<Project[]>('/api/projects')
      .then(async (p) => {
        const list = p ?? []
        if (cancelled) return
        setProjects(list)

        const jobs = await Promise.all(
          list.map(async (project) => {
            try {
              const items = (await apiFetch<Job[]>(`/api/projects/${project.id}/jobs`)) ?? []
              return [project.id, items[0]] as const
            } catch {
              return [project.id, undefined] as const
            }
          }),
        )
        if (!cancelled) setLastJobs(new Map(jobs))
      })
      .catch((err) => {
        if (err instanceof ApiError && (err.status === 401 || err.status === 400)) {
          router.replace('/')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [router])

  return (
    <>
      <PageHeader
        title="Проекты"
        description="Все репозитории, подключённые к diployment."
        actions={
          !loading && projects.length > 0 ? (
            <Link href="/projects/new">
              <Button>
                <Plus className="h-4 w-4" />
                Импортировать проект
              </Button>
            </Link>
          ) : null
        }
      />

      {loading ? (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <EmptyState
          icon={<FolderGit2 className="h-6 w-6" />}
          title="Проектов пока нет"
          description="Импортируй репозиторий — будем ловить коммиты и катить деплои."
          action={
            <Link href="/projects/new">
              <Button>
                <Plus className="h-4 w-4" />
                Импортировать первый проект
              </Button>
            </Link>
          }
        />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {projects.map((project, i) => (
            <ProjectCard
              key={project.id}
              project={project}
              lastJob={lastJobs.get(project.id)}
              index={i}
            />
          ))}
        </div>
      )}
    </>
  )
}
