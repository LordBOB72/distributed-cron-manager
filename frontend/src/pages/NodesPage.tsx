import { useNodes } from '../hooks/useJobs'
import { formatDistanceToNow } from 'date-fns'

export function NodesPage() {
  const { data: nodes = [], isLoading } = useNodes()

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-medium">Worker Nodes</h1>
      <div className="grid gap-3">
        {isLoading && <p className="text-sm text-gray-500 font-mono animate-pulse">Loading…</p>}
        {nodes.map((node) => (
          <div key={node.id} className="bg-surface-1 border border-surface-3 rounded-lg px-5 py-4 flex items-center justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="font-mono text-sm text-cyan-400">{node.hostname}</span>
                {node.is_leader && (
                  <span className="text-xs px-2 py-0.5 rounded bg-yellow-500/20 text-yellow-400 border border-yellow-500/40 font-mono">leader</span>
                )}
              </div>
              <p className="text-xs text-gray-500 font-mono">{node.id.slice(0, 16)}…</p>
            </div>
            <div className="text-right space-y-1">
              <p className="text-sm font-mono text-gray-300">{node.run_count} runs</p>
              <p className="text-xs text-gray-500">
                last seen {formatDistanceToNow(new Date(node.last_seen), { addSuffix: true })}
              </p>
            </div>
          </div>
        ))}
        {!isLoading && nodes.length === 0 && (
          <p className="text-sm text-gray-500 font-mono">No nodes registered.</p>
        )}
      </div>
    </div>
  )
}
