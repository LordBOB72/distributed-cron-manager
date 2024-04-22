import clsx from 'clsx'
import type { JobStatus } from '../types'

const styles: Record<JobStatus, string> = {
  pending:     'bg-gray-500/20 text-gray-400 border-gray-500/40',
  running:     'bg-blue-500/20 text-blue-400 border-blue-500/40 animate-pulse',
  success:     'bg-green-500/20 text-green-400 border-green-500/40',
  failed:      'bg-red-500/20 text-red-400 border-red-500/40',
  dead_letter: 'bg-red-900/40 text-red-300 border-red-700/60',
  timed_out:   'bg-orange-500/20 text-orange-400 border-orange-500/40',
}

export function RunStatusBadge({ status }: { status: JobStatus }) {
  return (
    <span className={clsx('text-xs font-mono px-2 py-0.5 rounded border whitespace-nowrap', styles[status])}>
      {status}
    </span>
  )
}
