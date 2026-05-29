'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'
import { ApiError, apiFetch } from '@/lib/api'
import { Sidebar } from '@/components/layout/sidebar'
import { ProjectsProvider } from '@/lib/hooks/projects-context'
import type { Project, User } from '@/lib/types'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [ready, setReady] = useState(false)

  useEffect(() => {
    Promise.all([
      apiFetch<User>('/api/me'),
      apiFetch<Project[]>('/api/projects'),
    ])
      .then(([u, p]) => {
        setUser(u)
        setProjects(p ?? [])
        setReady(true)
      })
      .catch((err) => {
        if (err instanceof ApiError && (err.status === 401 || err.status === 400)) {
          router.replace('/')
          return
        }
        // Unexpected error — surface to default error boundary.
        throw err
      })
  }, [router])

  if (!ready || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950 text-sm text-zinc-500">
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        Загрузка…
      </div>
    )
  }

  return (
    <ProjectsProvider initial={projects}>
      <div className="relative flex min-h-screen">
        <div className="pointer-events-none fixed inset-0 bg-radial-violet" />
        <div className="pointer-events-none fixed inset-0 bg-grid opacity-50 [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_30%,transparent_75%)]" />

        <Sidebar user={user} />

        <main className="relative z-10 flex-1 overflow-x-hidden">
          <div className="mx-auto max-w-6xl px-8 py-10">{children}</div>
        </main>
      </div>
    </ProjectsProvider>
  )
}
