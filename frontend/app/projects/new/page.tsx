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
  const [selected, setSelected] = useState<Repo | null>(null)
  const [branches, setBranches] = useState<string[]>([])
  const [branchLoading, setBranchLoading] = useState(false)
  const [branch, setBranch] = useState('')
  const [importing, setImporting] = useState(false)

  // SSH / deploy fields
  const [sshHost, setSshHost] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [deployRestartCmd, setDeployRestartCmd] = useState('')
  const [deployWorkdir, setDeployWorkdir] = useState('')

  useEffect(() => {
    apiFetch<Repo[]>('/api/repos')
      .then((r) => setRepos(r ?? []))
      .catch((err) => {
        if (err.status === 401 || err.status === 400) router.replace('/')
      })
      .finally(() => setLoading(false))
  }, [router])

  async function selectRepo(repo: Repo) {
    setSelected(repo)
    setBranches([])
    setBranch('')
    setBranchLoading(true)
    try {
      const list = await apiFetch<string[]>(
        `/api/repos/branches?full_name=${encodeURIComponent(repo.full_name)}`,
      )
      const b = list ?? []
      setBranches(b)
      setBranch(b[0] ?? '')
    } catch {
      setBranches([])
    } finally {
      setBranchLoading(false)
    }
  }

  async function handleImport() {
    if (!selected || !branch) return
    setImporting(true)
    try {
      await apiFetch('/api/projects/import', {
        method: 'POST',
        body: JSON.stringify({
          repo_full_name: selected.full_name,
          clone_url: selected.clone_url,
          branch,
          ssh_host: sshHost,
          ssh_port: parseInt(sshPort, 10),
          ssh_user: sshUser,
          ssh_key: btoa(sshKey),
          deploy_restart_cmd: deployRestartCmd,
          deploy_workdir: deployWorkdir,
        }),
      })
      router.push('/dashboard')
    } catch (err: unknown) {
      const status = (err as { status?: number }).status
      if (status === 409) {
        alert('This repository is already imported.')
      } else {
        alert('Import failed. Please try again.')
      }
    } finally {
      setImporting(false)
    }
  }

  const importDisabled =
    importing ||
    branchLoading ||
    !branch ||
    !sshHost ||
    !sshPort ||
    !sshUser ||
    !sshKey ||
    !deployRestartCmd ||
    !deployWorkdir

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-zinc-400">
        Loading repositories...
      </div>
    )
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
        <div className="flex flex-col gap-2">
          {repos.map((repo) => {
            const isSelected = selected?.id === repo.id
            return (
              <div key={repo.id} className="overflow-hidden rounded-xl border border-zinc-800">
                <button
                  onClick={() => isSelected ? setSelected(null) : selectRepo(repo)}
                  className={`flex w-full items-center gap-3 px-5 py-3 text-left transition-colors ${
                    isSelected ? 'bg-zinc-800' : 'bg-zinc-900 hover:bg-zinc-800'
                  }`}
                >
                  <span className="flex-1 font-medium">{repo.full_name}</span>
                  {repo.private && (
                    <span className="rounded bg-zinc-700 px-1.5 py-0.5 text-xs text-zinc-400">
                      Private
                    </span>
                  )}
                  <span className="text-xs text-zinc-500">{isSelected ? '▲' : '▼'}</span>
                </button>

                {isSelected && (
                  <div className="border-t border-zinc-800 bg-zinc-950 px-5 py-4">
                    <div className="flex flex-col gap-4">

                      {/* Branch selection */}
                      <div className="flex items-center gap-3">
                        <label className="w-28 shrink-0 text-sm text-zinc-400">Branch</label>
                        {branchLoading ? (
                          <span className="text-sm text-zinc-500">Loading branches…</span>
                        ) : branches.length === 0 ? (
                          <span className="text-sm text-zinc-500">No branches found</span>
                        ) : (
                          <select
                            value={branch}
                            onChange={(e) => setBranch(e.target.value)}
                            className="rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 outline-none focus:border-zinc-500"
                          >
                            {branches.map((b) => (
                              <option key={b} value={b}>
                                {b}
                              </option>
                            ))}
                          </select>
                        )}
                      </div>

                      {/* SSH / Deploy section */}
                      <div className="flex flex-col gap-3 rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-4">
                        <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">
                          SSH &amp; Deploy
                        </p>

                        {/* SSH host + port on one row */}
                        <div className="flex gap-3">
                          <div className="flex flex-1 items-center gap-3">
                            <label className="w-28 shrink-0 text-sm text-zinc-400">Host</label>
                            <input
                              type="text"
                              value={sshHost}
                              onChange={(e) => setSshHost(e.target.value)}
                              placeholder="example.com"
                              className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500"
                            />
                          </div>
                          <div className="flex items-center gap-3">
                            <label className="w-10 shrink-0 text-sm text-zinc-400">Port</label>
                            <input
                              type="number"
                              value={sshPort}
                              onChange={(e) => setSshPort(e.target.value)}
                              placeholder="22"
                              className="w-20 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500"
                            />
                          </div>
                        </div>

                        {/* SSH user */}
                        <div className="flex items-center gap-3">
                          <label className="w-28 shrink-0 text-sm text-zinc-400">User</label>
                          <input
                            type="text"
                            value={sshUser}
                            onChange={(e) => setSshUser(e.target.value)}
                            placeholder="deploy"
                            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500"
                          />
                        </div>

                        {/* Deploy workdir */}
                        <div className="flex items-center gap-3">
                          <label className="w-28 shrink-0 text-sm text-zinc-400">Workdir</label>
                          <input
                            type="text"
                            value={deployWorkdir}
                            onChange={(e) => setDeployWorkdir(e.target.value)}
                            placeholder="/srv/myapp"
                            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500"
                          />
                        </div>

                        {/* Restart command */}
                        <div className="flex items-center gap-3">
                          <label className="w-28 shrink-0 text-sm text-zinc-400">Restart cmd</label>
                          <input
                            type="text"
                            value={deployRestartCmd}
                            onChange={(e) => setDeployRestartCmd(e.target.value)}
                            placeholder="systemctl restart myapp"
                            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500"
                          />
                        </div>

                        {/* SSH private key */}
                        <div className="flex gap-3">
                          <label className="w-28 shrink-0 pt-1.5 text-sm text-zinc-400">
                            Private key
                          </label>
                          <textarea
                            value={sshKey}
                            onChange={(e) => setSshKey(e.target.value)}
                            rows={6}
                            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..."
                            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 font-mono text-xs text-zinc-200 placeholder-zinc-600 outline-none focus:border-zinc-500 resize-none"
                          />
                        </div>
                      </div>

                      {/* Import button */}
                      <div className="flex justify-end">
                        <button
                          onClick={handleImport}
                          disabled={importDisabled}
                          className="rounded-lg bg-zinc-100 px-5 py-2 text-sm font-semibold text-zinc-900 transition-colors hover:bg-zinc-300 disabled:opacity-50"
                        >
                          {importing ? 'Importing…' : 'Import'}
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </main>
  )
}
