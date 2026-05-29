'use client'

import { useRouter } from 'next/navigation'
import { motion } from 'motion/react'
import { GitCommit } from 'lucide-react'
import { StatusBadge } from '@/components/jobs/status-badge'
import { Tooltip } from '@/components/ui/tooltip'
import { formatDateTime, formatDuration, shortSha, timeAgo } from '@/lib/format'
import type { Job } from '@/lib/types'

interface JobRowProps {
  job: Job
  projectId: string
  index: number
}

export function JobRow({ job, projectId, index }: JobRowProps) {
  const router = useRouter()
  const status = job.status
  const start = job.build_started_at ?? job.created_at
  const end = job.deploy_finished_at ?? job.build_finished_at
  const duration = formatDuration(start, end)

  return (
    <motion.tr
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.025, duration: 0.24, ease: [0.22, 1, 0.36, 1] }}
      onClick={() => router.push(`/projects/${projectId}/jobs/${job.id}`)}
      className="group cursor-pointer border-b border-zinc-800/60 transition-colors last:border-b-0 hover:bg-zinc-900/40"
    >
      <td className="py-3.5 pl-5 pr-4">
        <StatusBadge status={status} />
      </td>
      <td className="px-4 py-3.5">
        <div className="flex items-center gap-2 text-sm">
          <GitCommit className="h-3.5 w-3.5 shrink-0 text-zinc-600" />
          <span className="font-mono text-zinc-300">{shortSha(job.commit_sha)}</span>
        </div>
      </td>
      <td className="max-w-xs px-4 py-3.5">
        <p className="truncate text-sm text-zinc-200">{job.commit_msg || <span className="text-zinc-600">(без сообщения)</span>}</p>
      </td>
      <td className="px-4 py-3.5 text-sm text-zinc-400">
        {duration ?? <span className="text-zinc-600">—</span>}
      </td>
      <td className="px-4 py-3.5 pr-5 text-right text-sm text-zinc-500">
        <Tooltip content={formatDateTime(job.created_at)}>
          <span>{timeAgo(job.created_at)}</span>
        </Tooltip>
      </td>
    </motion.tr>
  )
}
