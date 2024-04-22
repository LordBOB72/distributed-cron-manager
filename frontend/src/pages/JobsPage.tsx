import { useState } from 'react'
import { useJobs, useJobMutations } from '../hooks/useJobs'
import { RunStatusBadge } from '../components/RunStatusBadge'
import { JobDrawer } from '../components/JobDrawer'
import { CreateJobModal } from '../components/CreateJobModal'
import type { Job } from '../types'

export function JobsPage() {
  const { data: jobs = [], isLoading } = useJobs()
  const { trigger, pause, resume, delete: del } = useJobMutations()
  const [selected, setSelected] = useState<Job | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium">Jobs</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="text-sm px-3 py-1.5 bg-blue-500/20 text-blue-400 border border-blue-500/40 rounded hover:bg-blue-500/30 transition-colors"
        >
          + New job
        </button>
      </div>

      <div className="rounded-lg border border-surface-3 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-surface-2 border-b border-surface-3 text-left">
              <th className="px-4 py-3 text-gray-400 font-medium">Name</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Schedule</th>
              <th className="px-4 py-3 text-gray-400 font-medium">TZ</th>
              <th className="px-4 py-3 text-gray-400 font-medium">State</th>
              <th className="px-4 py-3 text-gray-400 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500 font-mono text-sm animate-pulse">Loading…</td></tr>
            )}
            {jobs.map((job) => (
              <tr
                key={job.id}
                onClick={() => setSelected(job)}
                className="border-b border-surface-3 last:border-0 hover:bg-surface-2/50 cursor-pointer"
              >
                <td className="px-4 py-3 font-mono text-cyan-400">{job.name}</td>
                <td className="px-4 py-3 font-mono text-gray-300 text-xs">{job.cron_expr}</td>
                <td className="px-4 py-3 text-gray-500 text-xs">{job.timezone}</td>
                <td className="px-4 py-3">
                  {job.paused ? (
                    <span className="text-xs font-mono px-2 py-0.5 rounded border bg-yellow-500/10 text-yellow-400 border-yellow-500/30">paused</span>
                  ) : (
                    <span className="text-xs font-mono px-2 py-0.5 rounded border bg-green-500/10 text-green-400 border-green-500/30">active</span>
                  )}
                </td>
                <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => trigger.mutate(job.id)}
                      className="text-xs px-2 py-1 bg-blue-500/10 text-blue-400 border border-blue-500/30 rounded hover:bg-blue-500/20 transition-colors"
                    >
                      Run now
                    </button>
                    {job.paused ? (
                      <button onClick={() => resume.mutate(job.id)} className="text-xs text-green-400 hover:text-green-300">Resume</button>
                    ) : (
                      <button onClick={() => pause.mutate(job.id)} className="text-xs text-yellow-400 hover:text-yellow-300">Pause</button>
                    )}
                    <button onClick={() => del.mutate(job.id)} className="text-xs text-gray-600 hover:text-red-400">Delete</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!isLoading && jobs.length === 0 && (
          <p className="px-4 py-8 text-center text-sm text-gray-500 font-mono">No jobs defined.</p>
        )}
      </div>

      {selected && <JobDrawer job={selected} onClose={() => setSelected(null)} />}
      {showCreate && <CreateJobModal onClose={() => setShowCreate(false)} />}
    </div>
  )
}
