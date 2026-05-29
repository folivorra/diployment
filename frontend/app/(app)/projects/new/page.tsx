'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { motion } from 'motion/react'
import { ArrowLeft, Inbox } from 'lucide-react'
import { apiFetch, ApiError } from '@/lib/api'
import { Breadcrumbs } from '@/components/layout/breadcrumbs'
import { RepoPicker } from '@/components/projects/repo-picker'
import { ImportForm } from '@/components/projects/import-form'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/ui/empty-state'
import type { Repo } from '@/lib/types'

export default function NewProjectPage() {
  const router = useRouter()
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Repo | null>(null)

  useEffect(() => {
    apiFetch<Repo[]>('/api/repos')
      .then((r) => setRepos(r ?? []))
      .catch((err) => {
        if (err instanceof ApiError && (err.status === 401 || err.status === 400)) {
          router.replace('/')
        }
      })
      .finally(() => setLoading(false))
  }, [router])

  return (
    <>
      <div className="mb-2">
        <Breadcrumbs
          items={[
            { label: 'Дашборд', href: '/dashboard' },
            { label: 'Импорт проекта' },
          ]}
        />
      </div>

      <motion.div
        initial={{ opacity: 0, y: -4 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28 }}
        className="mb-8 flex items-end justify-between"
      >
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-50">Импорт репозитория</h1>
          <p className="mt-1 text-sm text-zinc-500">
            Выбери GitHub-репозиторий, укажи SSH-цель — будем катить на каждый push.
          </p>
        </div>
        <Link href="/dashboard" className="flex items-center gap-1.5 text-xs text-zinc-500 transition-colors hover:text-zinc-300">
          <ArrowLeft className="h-3 w-3" />
          Отмена
        </Link>
      </motion.div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
        <div className="flex flex-col" style={{ height: 'calc(100vh - 240px)', minHeight: 480 }}>
          {loading ? (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-9 w-full" />
              ))}
            </div>
          ) : (
            <RepoPicker repos={repos} selectedId={selected?.id ?? null} onSelect={setSelected} />
          )}
        </div>

        <div className="rounded-xl border border-zinc-800/80 bg-zinc-900/30 p-6 backdrop-blur-sm">
          {selected ? (
            <ImportForm repo={selected} />
          ) : (
            <EmptyState
              icon={<Inbox className="h-6 w-6" />}
              title="Выбери репозиторий"
              description="Выбери репозиторий из списка слева, чтобы настроить деплой."
              className="border-0 bg-transparent py-14"
            />
          )}
        </div>
      </div>
    </>
  )
}
