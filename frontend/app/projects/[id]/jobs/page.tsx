'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { apiFetch } from '@/lib/api'
import StatusBadge from '@/components/StatusBadge'
import type { Job } from '@/lib/types'

export default function JobsPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchJobs = () =>
      apiFetch<Job[]>(`/api/projects/${id}/jobs`)
        .then((j) => setJobs(j ?? []))
        .catch((err) => {
          if (err.status === 401 || err.status === 400) router.replace('/')
          if (err.status === 403 || err.status === 404) router.replace('/dashboard')
        })
        .finally(() => setLoading(false))

    fetchJobs()
    const interval = setInterval(fetchJobs, 5000)
    return () => clearInterval(interval)
  }, [id, router])

  if (loading) {
    return <div className="flex min-h-screen items-center justify-center text-zinc-400">Loading jobs...</div>
  }

  return (
    <main className="mx-auto max-w-3xl p-8">
      <div className="mb-8 flex items-center gap-4">
        <Link href="/dashboard" className="text-zinc-500 hover:text-zinc-300">
          ← Dashboard
        </Link>
        <h1 className="text-2xl font-bold">Build history</h1>
      </div>

      {jobs.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 p-12 text-center text-zinc-500">
          No builds yet. Push a commit to trigger one.
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-800">
          <table className="w-full text-sm">
            <thead className="border-b border-zinc-800 bg-zinc-900 text-zinc-400">
              <tr>
                <th className="px-5 py-3 text-left font-medium">Status</th>
                <th className="px-5 py-3 text-left font-medium">Commit</th>
                <th className="px-5 py-3 text-left font-medium">Message</th>
                <th className="px-5 py-3 text-left font-medium">Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800">
              {jobs.map((job) => (
                <tr
                  key={job.id}
                  className="cursor-pointer bg-zinc-950 transition-colors hover:bg-zinc-900"
                  onClick={() => router.push(`/projects/${id}/jobs/${job.id}`)}
                >
                  <td className="px-5 py-3">
                    <StatusBadge status={job.status} />
                  </td>
                  <td className="px-5 py-3 font-mono text-zinc-400">
                    {job.commit_sha.slice(0, 7)}
                  </td>
                  <td className="max-w-xs truncate px-5 py-3 text-zinc-300">
                    {job.commit_msg}
                  </td>
                  <td className="px-5 py-3 text-zinc-500">
                    {new Date(job.created_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  )
}
