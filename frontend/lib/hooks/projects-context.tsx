'use client'

import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { apiFetch } from '@/lib/api'
import type { Project } from '@/lib/types'

interface ProjectsContextValue {
  projects: Project[]
  refresh: () => Promise<void>
}

const ProjectsContext = createContext<ProjectsContextValue | null>(null)

export function ProjectsProvider({
  initial,
  children,
}: {
  initial: Project[]
  children: React.ReactNode
}) {
  const [projects, setProjects] = useState<Project[]>(initial)

  const refresh = useCallback(async () => {
    const list = await apiFetch<Project[]>('/api/projects')
    setProjects(list ?? [])
  }, [])

  // Keep state in sync if the layout refetches and passes a new initial list.
  useEffect(() => {
    setProjects(initial)
  }, [initial])

  return (
    <ProjectsContext.Provider value={{ projects, refresh }}>
      {children}
    </ProjectsContext.Provider>
  )
}

export function useProjects(): Project[] {
  const ctx = useContext(ProjectsContext)
  if (!ctx) throw new Error('useProjects must be used inside <ProjectsProvider>')
  return ctx.projects
}

export function useRefreshProjects(): () => Promise<void> {
  const ctx = useContext(ProjectsContext)
  if (!ctx) throw new Error('useRefreshProjects must be used inside <ProjectsProvider>')
  return ctx.refresh
}
