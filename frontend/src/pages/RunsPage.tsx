import { useAllRuns } from '../hooks/useJobs'
import { RunStatusBadge } from '../components/RunStatusBadge'
import { formatDistanceToNow } from 'date-fns'

export function RunsPage() {
  const { data: runs = [], isLoading } = useAllRuns()

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-medium">All Runs</h1>
      <div className="bg-surface-1 border border-surface-3 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-surface-2 border-b border-surface-3 text-left">
              <th className="px-4 py-3 text-gray-400 font-medium">Job</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Node</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Status</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Attempt</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Duration</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Scheduled</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500 font-mono text-sm animate-pulse">Loading…</td></tr>}
            {runs.map((run) => {
              const duration = run.started_at && run.completed_at
                ? `${((new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000).toFixed(1)}s`
                : '—'
              return (
                <tr key={run.id} className="border-b border-surface-3 last:border-0 hover:bg-surface-2/30">
                  <td className="px-4 py-3 font-mono text-cyan-400 text-xs">{run.job_name ?? run.job_id.slice(0, 8)}</td>
                  <td className="px-4 py-3 font-mono text-gray-500 text-xs">{run.node_id.slice(0, 8)}</td>
                  <td className="px-4 py-3"><RunStatusBadge status={run.status} /></td>
                  <td className="px-4 py-3 text-gray-400 font-mono text-xs">{run.attempt}</td>
                  <td className="px-4 py-3 text-gray-400 font-mono text-xs">{duration}</td>
                  <td className="px-4 py-3 text-gray-500 text-xs">
                    {formatDistanceToNow(new Date(run.scheduled_at), { addSuffix: true })}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
