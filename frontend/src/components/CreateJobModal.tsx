import { useState } from 'react'
import { useJobMutations } from '../hooks/useJobs'

export function CreateJobModal({ onClose }: { onClose: () => void }) {
  const { create } = useJobMutations()
  const [form, setForm] = useState({
    name: '', cron_expr: '0 * * * *', timezone: 'UTC',
    command: '', max_retries: 3, timeout_secs: 300,
    tags: [] as string[], depends_on: [] as string[],
  })
  const [err, setErr] = useState<string | null>(null)

  const set = (k: string, v: unknown) => setForm((f) => ({ ...f, [k]: v }))

  async function submit() {
    setErr(null)
    try {
      await create.mutateAsync(form)
      onClose()
    } catch (e) { setErr(String(e)) }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-surface-1 border border-surface-3 rounded-lg p-6 w-full max-w-lg space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="font-medium">Create job</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-white">✕</button>
        </div>

        <div className="grid grid-cols-2 gap-3">
          {[
            { label: 'Name', key: 'name', placeholder: 'cleanup-old-logs', span: 2 },
            { label: 'Cron expression', key: 'cron_expr', placeholder: '0 2 * * *', span: 1 },
            { label: 'Timezone', key: 'timezone', placeholder: 'UTC', span: 1 },
            { label: 'Command', key: 'command', placeholder: '/usr/bin/cleanup.sh', span: 2 },
          ].map(({ label, key, placeholder, span }) => (
            <div key={key} className={span === 2 ? 'col-span-2' : ''}>
              <label className="block text-xs text-gray-500 mb-1">{label}</label>
              <input
                value={String(form[key as keyof typeof form])}
                onChange={(e) => set(key, e.target.value)}
                placeholder={placeholder}
                className="w-full bg-surface-0 border border-surface-3 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
              />
            </div>
          ))}

          <div>
            <label className="block text-xs text-gray-500 mb-1">Max retries</label>
            <input type="number" value={form.max_retries} onChange={(e) => set('max_retries', parseInt(e.target.value))}
              className="w-full bg-surface-0 border border-surface-3 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500" />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Timeout (s)</label>
            <input type="number" value={form.timeout_secs} onChange={(e) => set('timeout_secs', parseInt(e.target.value))}
              className="w-full bg-surface-0 border border-surface-3 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500" />
          </div>
        </div>

        {err && <p className="text-xs text-red-400 font-mono">{err}</p>}

        <div className="flex justify-end gap-3 pt-1">
          <button onClick={onClose} className="text-sm text-gray-500 hover:text-white px-3 py-1.5">Cancel</button>
          <button
            onClick={submit}
            disabled={create.isPending || !form.name || !form.command}
            className="text-sm px-4 py-1.5 bg-blue-500/20 text-blue-400 border border-blue-500/40 rounded hover:bg-blue-500/30 disabled:opacity-50"
          >
            {create.isPending ? 'Creating…' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  )
}
