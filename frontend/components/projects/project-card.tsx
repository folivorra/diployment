'use client'

import Link from 'next/link'
import { motion } from 'motion/react'
import { GitBranch, ArrowUpRight } from 'lucide-react'
import { StatusBadge } from '@/components/jobs/status-badge'
import { Tooltip } from '@/components/ui/tooltip'
import { shortSha, timeAgo, formatDateTime } from '@/lib/format'
import type { Job, Project } from '@/lib/types'

interface ProjectCardProps {
  project: Project
  lastJob?: Job
  index: number
}

export function ProjectCard({ project, lastJob, index }: ProjectCardProps) {
  const owner = project.repo_full_name.split('/')[0]
  const name = project.repo_full_name.split('/')[1]

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.04, duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
    >
      <Link
        href={`/projects/${project.id}/jobs`}
        className="group relative block overflow-hidden rounded-xl border border-zinc-800/80 bg-zinc-900/40 p-5 backdrop-blur-sm transition-all duration-200 hover:border-zinc-700 hover:bg-zinc-900/70 hover:shadow-[0_0_0_1px_rgba(99,102,241,0.15),0_12px_40px_-12px_rgba(99,102,241,0.35)]"
      >
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 -z-10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
          style={{
            background:
              'radial-gradient(80% 60% at 100% 0%, rgba(99, 102, 241, 0.08) 0%, transparent 60%)',
          }}
        />

        <div className="mb-3 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-[11px] text-zinc-500">{owner}</p>
            <h3 className="truncate font-mono text-sm font-semibold text-zinc-50">{name}</h3>
          </div>
          <ArrowUpRight className="h-4 w-4 shrink-0 text-zinc-600 transition-all duration-200 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-indigo-300" />
        </div>

        <div className="flex items-center gap-2 text-xs text-zinc-500">
          <GitBranch className="h-3 w-3" />
          <span className="font-mono text-zinc-400">{project.branch}</span>
        </div>

        <div className="mt-4 flex items-center justify-between border-t border-zinc-800/60 pt-3">
          {lastJob ? (
            <>
              <div className="flex min-w-0 items-center gap-2">
                <StatusBadge status={lastJob.status} size="sm" />
                <span className="truncate font-mono text-[11px] text-zinc-500">
                  {shortSha(lastJob.commit_sha)}
                </span>
              </div>
              <Tooltip content={formatDateTime(lastJob.created_at)}>
                <span className="shrink-0 text-[11px] text-zinc-500">
                  {timeAgo(lastJob.created_at)}
                </span>
              </Tooltip>
            </>
          ) : (
            <span className="text-[11px] text-zinc-600">Джоб ещё нет</span>
          )}
        </div>
      </Link>
    </motion.div>
  )
}
