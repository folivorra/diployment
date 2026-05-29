'use client'

import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Search, Lock, Globe } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'
import type { Repo } from '@/lib/types'

interface RepoPickerProps {
  repos: Repo[]
  selectedId: number | null
  onSelect: (repo: Repo) => void
}

export function RepoPicker({ repos, selectedId, onSelect }: RepoPickerProps) {
  const [query, setQuery] = useState('')

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return repos
    return repos.filter((r) => r.full_name.toLowerCase().includes(q))
  }, [repos, query])

  return (
    <div className="flex h-full flex-col">
      <div className="relative mb-3">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-500" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Поиск репозиториев…"
          className="pl-9"
        />
      </div>

      <div className="flex-1 overflow-y-auto rounded-xl border border-zinc-800/80 bg-zinc-900/20">
        {filtered.length === 0 ? (
          <p className="px-4 py-8 text-center text-xs text-zinc-600">
            {query ? 'Нет совпадений.' : 'Репозиториев не найдено.'}
          </p>
        ) : (
          <ul className="flex flex-col">
            <AnimatePresence initial={false}>
              {filtered.map((repo, i) => {
                const isSelected = repo.id === selectedId
                return (
                  <motion.li
                    key={repo.id}
                    initial={{ opacity: 0, x: -4 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.16, delay: Math.min(i * 0.01, 0.15) }}
                  >
                    <button
                      type="button"
                      onClick={() => onSelect(repo)}
                      className={cn(
                        'flex w-full items-center gap-2.5 border-l-2 px-4 py-2.5 text-left text-sm transition-all duration-150',
                        isSelected
                          ? 'border-l-indigo-500 bg-indigo-500/8 text-zinc-50'
                          : 'border-l-transparent text-zinc-300 hover:bg-zinc-900/60 hover:text-zinc-100',
                      )}
                    >
                      {repo.private ? (
                        <Lock className="h-3 w-3 shrink-0 text-amber-400/80" />
                      ) : (
                        <Globe className="h-3 w-3 shrink-0 text-zinc-500" />
                      )}
                      <span className="truncate font-mono text-[13px]">{repo.full_name}</span>
                    </button>
                  </motion.li>
                )
              })}
            </AnimatePresence>
          </ul>
        )}
      </div>
    </div>
  )
}
