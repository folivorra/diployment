'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { motion } from 'motion/react'
import { Loader2, GitBranch, Server, Settings2, KeyRound } from 'lucide-react'
import { toast } from 'sonner'
import { Field, Input, Select, Textarea } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { apiFetch, ApiError } from '@/lib/api'
import { useRefreshProjects } from '@/lib/hooks/projects-context'
import type { Repo } from '@/lib/types'
import { cn } from '@/lib/cn'

interface ImportFormProps {
  repo: Repo
}

export function ImportForm({ repo }: ImportFormProps) {
  const router = useRouter()
  const refreshProjects = useRefreshProjects()
  const [branches, setBranches] = useState<string[]>([])
  const [branchLoading, setBranchLoading] = useState(true)
  const [branch, setBranch] = useState('')

  const [sshHost, setSshHost] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [deployRestartCmd, setDeployRestartCmd] = useState('')
  const [deployWorkdir, setDeployWorkdir] = useState('')

  const [importing, setImporting] = useState(false)

  useEffect(() => {
    setBranches([])
    setBranch('')
    setBranchLoading(true)
    apiFetch<string[]>(`/api/repos/branches?full_name=${encodeURIComponent(repo.full_name)}`)
      .then((list) => {
        const b = list ?? []
        setBranches(b)
        // Prefer "main" or "master" if present
        const preferred = b.find((x) => x === 'main') ?? b.find((x) => x === 'master') ?? b[0]
        setBranch(preferred ?? '')
      })
      .catch(() => setBranches([]))
      .finally(() => setBranchLoading(false))
  }, [repo.full_name])

  const portNum = parseInt(sshPort, 10)
  const portValid = Number.isFinite(portNum) && portNum > 0 && portNum < 65536
  const keyValid = sshKey.trim().startsWith('-----BEGIN')

  const canSubmit =
    !importing &&
    !branchLoading &&
    branch !== '' &&
    sshHost.trim() !== '' &&
    portValid &&
    sshUser.trim() !== '' &&
    keyValid &&
    deployWorkdir.trim() !== '' &&
    deployRestartCmd.trim() !== ''

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return

    setImporting(true)
    try {
      await apiFetch('/api/projects/import', {
        method: 'POST',
        body: JSON.stringify({
          repo_full_name: repo.full_name,
          clone_url: repo.clone_url,
          branch,
          ssh_host: sshHost.trim(),
          ssh_port: portNum,
          ssh_user: sshUser.trim(),
          ssh_key: btoa(sshKey),
          deploy_restart_cmd: deployRestartCmd.trim(),
          deploy_workdir: deployWorkdir.trim(),
        }),
      })
      toast.success('Проект импортирован', { description: repo.full_name })
      await refreshProjects()
      router.push('/dashboard')
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        toast.error('Репозиторий уже импортирован')
      } else {
        toast.error('Импорт не удался', { description: 'Проверь SSH-данные и попробуй ещё раз.' })
      }
    } finally {
      setImporting(false)
    }
  }

  return (
    <motion.form
      key={repo.id}
      onSubmit={handleSubmit}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
      className="flex flex-col gap-5"
    >
      <header className="flex items-start justify-between gap-3 border-b border-zinc-800/60 pb-4">
        <div className="min-w-0">
          <p className="text-[10px] font-medium uppercase tracking-wider text-zinc-500">
            Импорт
          </p>
          <p className="truncate font-mono text-base text-zinc-50">{repo.full_name}</p>
        </div>
      </header>

      <Section icon={<GitBranch className="h-4 w-4" />} title="Источник">
        <Field label="Ветка" htmlFor="branch">
          {branchLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : branches.length === 0 ? (
            <p className="text-xs text-red-400">Веток не найдено.</p>
          ) : (
            <Select id="branch" value={branch} onChange={(e) => setBranch(e.target.value)}>
              {branches.map((b) => (
                <option key={b} value={b}>{b}</option>
              ))}
            </Select>
          )}
        </Field>
      </Section>

      <Section icon={<Server className="h-4 w-4" />} title="SSH-цель">
        <div className="grid grid-cols-[1fr_120px] gap-3">
          <Field label="Хост" htmlFor="ssh-host">
            <Input
              id="ssh-host"
              value={sshHost}
              onChange={(e) => setSshHost(e.target.value)}
              placeholder="deploy.example.com"
              autoComplete="off"
            />
          </Field>
          <Field label="Порт" htmlFor="ssh-port">
            <Input
              id="ssh-port"
              type="number"
              value={sshPort}
              onChange={(e) => setSshPort(e.target.value)}
              min={1}
              max={65535}
              className={cn(!portValid && sshPort !== '' && 'border-red-500/50')}
            />
          </Field>
        </div>

        <Field label="Пользователь" htmlFor="ssh-user">
          <Input
            id="ssh-user"
            value={sshUser}
            onChange={(e) => setSshUser(e.target.value)}
            placeholder="deploy"
            autoComplete="off"
          />
        </Field>

        <Field label="Приватный ключ" htmlFor="ssh-key" hint="Вставляется как есть; шифруется перед сохранением.">
          <div className="relative">
            <KeyRound className="pointer-events-none absolute left-3 top-3 h-3.5 w-3.5 text-zinc-600" />
            <Textarea
              id="ssh-key"
              value={sshKey}
              onChange={(e) => setSshKey(e.target.value)}
              rows={6}
              placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
              className={cn('pl-9', sshKey !== '' && !keyValid && 'border-red-500/50')}
              spellCheck={false}
            />
          </div>
        </Field>
      </Section>

      <Section icon={<Settings2 className="h-4 w-4" />} title="Деплой">
        <Field label="Рабочая директория" htmlFor="deploy-workdir">
          <Input
            id="deploy-workdir"
            value={deployWorkdir}
            onChange={(e) => setDeployWorkdir(e.target.value)}
            placeholder="/srv/myapp"
            className="font-mono"
            autoComplete="off"
          />
        </Field>
        <Field
          label="Команда перезапуска"
          htmlFor="deploy-restart"
          hint="Выполняется после загрузки артефакта."
        >
          <Input
            id="deploy-restart"
            value={deployRestartCmd}
            onChange={(e) => setDeployRestartCmd(e.target.value)}
            placeholder="systemctl restart myapp"
            className="font-mono"
            autoComplete="off"
          />
        </Field>
      </Section>

      <div className="flex justify-end gap-2 border-t border-zinc-800/60 pt-4">
        <Button type="submit" disabled={!canSubmit}>
          {importing ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {importing ? 'Импорт…' : 'Импортировать проект'}
        </Button>
      </div>
    </motion.form>
  )
}

function Section({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-2 text-zinc-300">
        <span className="text-indigo-400">{icon}</span>
        <span className="text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
          {title}
        </span>
      </div>
      <div className="flex flex-col gap-3">{children}</div>
    </section>
  )
}
