'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { apiFetch } from '@/lib/api'
import type { Project, User } from '@/lib/types'

export default function DashboardPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      apiFetch<User>('/api/me'),
      apiFetch<Project[]>('/api/projects'),
    ])
      .then(([u, p]) => {
        setUser(u)
        setProjects(p ?? [])
      })
      .catch((err) => {
        if (err.status === 401 || err.status === 400) router.replace('/')
      })
      .finally(() => setLoading(false))
  }, [router])

  if (loading) {
    return <div className="flex min-h-screen items-center justify-center text-zinc-400">Loading...</div>
  }

  return (
    <main className="mx-auto max-w-3xl p-8">
      <div className="mb-8 flex items-center justify-between">
        <div className="flex items-center gap-3">
          {user?.avatar_url && (
            <img src={user.avatar_url} alt="avatar" className="h-8 w-8 rounded-full" />
          )}
          <h1 className="text-2xl font-bold">Dashboard</h1>
        </div>
        <Link
          href="/projects/new"
          className="rounded-lg bg-zinc-100 px-4 py-2 text-sm font-semibold text-zinc-900 transition-colors hover:bg-zinc-300"
        >
          Import new project
        </Link>
      </div>

      {projects.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 p-12 text-center text-zinc-500">
          No projects yet.{' '}
          <Link href="/projects/new" className="text-zinc-300 underline hover:text-white">
            Import one
          </Link>
          .
        </div>
      ) : (
        <ul className="flex flex-col gap-3">
          {projects.map((p) => (
            <li key={p.id} className="flex items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900 px-5 py-4">
              <div>
                <p className="font-medium">{p.repo_full_name}</p>
                <p className="mt-0.5 text-sm text-zinc-500">
                  branch: <span className="text-zinc-400">{p.branch}</span>
                  {' · '}
                  {new Date(p.created_at).toLocaleDateString()}
                </p>
              </div>
              <Link
                href={`/projects/${p.id}/jobs`}
                className="rounded-lg border border-zinc-700 px-3 py-1.5 text-sm text-zinc-300 transition-colors hover:border-zinc-500 hover:text-white"
              >
                View jobs
              </Link>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
