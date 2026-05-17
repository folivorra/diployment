import type { JobStatus } from '@/lib/types'

const styles: Record<JobStatus, string> = {
  pending: 'bg-zinc-700 text-zinc-300',
  running: 'bg-blue-900 text-blue-300',
  success: 'bg-green-900 text-green-300',
  failed:  'bg-red-900 text-red-300',
}

export default function StatusBadge({ status }: { status: JobStatus }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${styles[status]}`}>
      {status}
    </span>
  )
}
