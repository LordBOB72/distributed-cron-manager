import type { Job, JobRun, Node } from '../types'

const BASE = '/api/v1'

async function req<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    ...opts,
  })
  if (!res.ok) throw new Error(`${res.status}: ${await res.text().catch(() => res.statusText)}`)
  if (res.status === 204) return undefined as unknown as T
  return res.json()
}

export const api = {
  jobs: {
    list: () => req<Job[]>('/jobs'),
    get: (id: string) => req<Job>(`/jobs/${id}`),
    create: (body: Omit<Job, 'id' | 'enabled' | 'paused' | 'created_at' | 'updated_at'>) =>
      req<{ id: string }>('/jobs', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: string, body: Partial<Job>) =>
      req<void>(`/jobs/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id: string) => req<void>(`/jobs/${id}`, { method: 'DELETE' }),
    trigger: (id: string) => req<{ status: string }>(`/jobs/${id}/trigger`, { method: 'POST' }),
    pause: (id: string) => req<void>(`/jobs/${id}/pause`, { method: 'POST' }),
    resume: (id: string) => req<void>(`/jobs/${id}/resume`, { method: 'POST' }),
    runs: (id: string) => req<JobRun[]>(`/jobs/${id}/runs`),
  },
  runs: {
    list: (limit = 100) => req<JobRun[]>(`/runs?limit=${limit}`),
  },
  nodes: {
    list: () => req<Node[]>('/nodes'),
  },
}
