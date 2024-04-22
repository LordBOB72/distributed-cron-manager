import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { api } from '../lib/api'

export function useJobs() {
  return useQuery({ queryKey: ['jobs'], queryFn: api.jobs.list, refetchInterval: 15_000 })
}

export function useJobRuns(jobId: string | null) {
  return useQuery({
    queryKey: ['runs', jobId],
    queryFn: () => api.jobs.runs(jobId!),
    enabled: !!jobId,
    refetchInterval: 5_000,
  })
}

export function useAllRuns() {
  return useQuery({ queryKey: ['runs'], queryFn: () => api.runs.list(), refetchInterval: 5_000 })
}

export function useNodes() {
  return useQuery({ queryKey: ['nodes'], queryFn: api.nodes.list, refetchInterval: 15_000 })
}

export function useJobMutations() {
  const qc = useQueryClient()
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['jobs'] })
    qc.invalidateQueries({ queryKey: ['runs'] })
  }

  return {
    trigger:  useMutation({ mutationFn: api.jobs.trigger,  onSuccess: invalidate }),
    pause:    useMutation({ mutationFn: api.jobs.pause,    onSuccess: invalidate }),
    resume:   useMutation({ mutationFn: api.jobs.resume,   onSuccess: invalidate }),
    delete:   useMutation({ mutationFn: api.jobs.delete,   onSuccess: invalidate }),
    create:   useMutation({ mutationFn: api.jobs.create,   onSuccess: invalidate }),
  }
}

// subscribes to the WebSocket and invalidates run queries on any message
export function useRunStream() {
  const qc = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const ws = new WebSocket(`ws://${window.location.host}/ws`)
    wsRef.current = ws
    ws.onmessage = () => {
      qc.invalidateQueries({ queryKey: ['runs'] })
    }
    return () => ws.close()
  }, [qc])
}
