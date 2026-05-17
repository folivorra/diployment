'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { apiFetch } from '@/lib/api'
import type { Repo } from '@/lib/types'

export default function NewProjectPage() {
  const router = useRouter()
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [branches, setBranches] = useState<Record<number, string>>({})
  const [states, setStates] = useState<Record<number, 'idle' | 'loading' | 'imported'>>({})

  useEffect(() => {
    apiFetch<Repo[]>('/api/repos')
      .then((r) => setRepos(r ?? []))
      .catch((err) => {
        if (err.status === 401 || err.status === 400) router.replace('/')
      })
      .finally(() => setLoading(false))
  }, [router])

  function branchFor(repo: Repo) {
    return branches[repo.id] ?? 'main'
  }

  async function handleImport(repo: Repo) {
    setStates((s) => ({ ...s, [repo.id]: 'loading' }))
    try {
      await apiFetch('/api/projects/import', {
        method: 'POST',
        body: JSON.stringify({
          repo_full_name: repo.full_name,
          clone_url: repo.clone_url,
          branch: branchFor(repo),
        }),
      })
      router.push('/dashboard')
    } catch (err: unknown) {
      const status = (err as { status?: number }).status
      if (status === 409) {
        setStates((s) => ({ ...s, [repo.id]: 'imported' }))
      } else {
        setStates((s) => ({ ...s, [repo.id]: 'idle' }))
        alert('Import failed. Please try again.')
      }
    }
  }

  if (loading) {
    return <div className="flex min-h-screen items-center justify-center text-zinc-400">Loading repositories...</div>
  }

  return (
    <main className="mx-auto max-w-3xl p-8">
      <div className="mb-8 flex items-center gap-4">
        <Link href="/dashboard" className="text-zinc-500 hover:text-zinc-300">
          ← Back
        </Link>
        <h1 className="text-2xl font-bold">Import project</h1>
      </div>

      {repos.length === 0 ? (
        <p className="text-zinc-500">No repositories found.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {repos.map((repo) => (
            <li key={repo.id} className="flex items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-900 px-5 py-3">
              <div className="min-w-0 flex-1">
                <span className="font-medium">{repo.full_name}</span>
                {repo.private && (
                  <span className="ml-2 rounded bg-zinc-700 px-1.5 py-0.5 text-xs text-zinc-400">Private</span>
                )}
              </div>
              <input
                type="text"
                value={branchFor(repo)}
                onChange={(e) => setBranches((b) => ({ ...b, [repo.id]: e.target.value }))}
                placeholder="branch"
                className="w-28 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 outline-none focus:border-zinc-500"
              />
              {states[repo.id] === 'imported' ? (
                <span className="text-sm text-zinc-500">Already imported</span>
              ) : (
                <button
                  onClick={() => handleImport(repo)}
                  disabled={states[repo.id] === 'loading'}
                  className="rounded-lg bg-zinc-100 px-4 py-1.5 text-sm font-semibold text-zinc-900 transition-colors hover:bg-zinc-300 disabled:opacity-50"
                >
                  {states[repo.id] === 'loading' ? 'Importing…' : 'Import'}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}
