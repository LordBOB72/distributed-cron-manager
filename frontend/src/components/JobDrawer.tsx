import { useJobRuns } from '../hooks/useJobs'
import { RunStatusBadge } from './RunStatusBadge'
import type { Job } from '../types'
import { formatDistanceToNow } from 'date-fns'

interface Props {
  job: Job
  onClose: () => void
}

export function JobDrawer({ job, onClose }: Props) {
  const { data: runs = [] } = useJobRuns(job.id)

  return (
    <div className="fixed inset-0 z-40 flex">
      <div className="flex-1 bg-black/40" onClick={onClose} />
      <div className="w-[520px] bg-surface-1 border-l border-surface-3 flex flex-col overflow-hidden">
        <div className="px-5 py-4 border-b border-surface-3 flex items-center justify-between">
          <div>
            <h2 className="font-mono text-cyan-400 font-medium">{job.name}</h2>
            <p className="text-xs text-gray-500 font-mono mt-0.5">{job.id}</p>
          </div>
          <button onClick={onClose} className="text-gray-500 hover:text-white text-lg">✕</button>
        </div>

        <div className="px-5 py-4 border-b border-surface-3 space-y-2 text-sm">
          <Row label="Schedule" value={job.cron_expr} mono />
          <Row label="Timezone" value={job.timezone} />
          <Row label="Command" value={job.command} mono />
          <Row label="Max retries" value={String(job.max_retries)} />
          <Row label="Timeout" value={`${job.timeout_secs}s`} />
          {job.tags.length > 0 && <Row label="Tags" value={job.tags.join(', ')} />}
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-4">
          <h3 className="text-xs text-gray-500 uppercase tracking-wider mb-3">Run history</h3>
          <div className="space-y-2">
            {runs.map((run) => {
              const duration =
                run.started_at && run.completed_at
                  ? `${((new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000).toFixed(1)}s`
                  : null
              return (
                <div key={run.id} className="bg-surface-2 rounded p-3 space-y-1.5">
                  <div className="flex items-center justify-between">
                    <RunStatusBadge status={run.status} />
                    <div className="flex items-center gap-3 text-xs text-gray-500">
                      {duration && <span>{duration}</span>}
                      <span className="font-mono">{run.node_id.slice(0, 8)}</span>
                      <span>{formatDistanceToNow(new Date(run.scheduled_at), { addSuffix: true })}</span>
                    </div>
                  </div>
                  {run.log_output && (
                    <pre className="text-xs font-mono text-gray-400 bg-surface-0 rounded p-2 overflow-x-auto max-h-24">
                      {run.log_output.slice(0, 500)}
                    </pre>
                  )}
                </div>
              )
            })}
            {runs.length === 0 && (
              <p className="text-sm text-gray-500 font-mono">No runs yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex gap-3">
      <span className="text-gray-500 w-24 shrink-0">{label}</span>
      <span className={mono ? 'font-mono text-gray-300 text-xs' : 'text-gray-300'}>{value}</span>
    </div>
  )
}
