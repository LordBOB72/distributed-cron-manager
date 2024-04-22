export type JobStatus = 'pending' | 'running' | 'success' | 'failed' | 'dead_letter' | 'timed_out'

export interface Job {
  id: string
  name: string
  cron_expr: string
  timezone: string
  command: string
  max_retries: number
  timeout_secs: number
  tags: string[]
  depends_on: string[]
  enabled: boolean
  paused: boolean
  created_at: string
  updated_at: string
}

export interface JobRun {
  id: string
  job_id: string
  job_name?: string
  node_id: string
  status: JobStatus
  attempt: number
  exit_code: number
  log_output: string
  scheduled_at: string
  started_at: string | null
  completed_at: string | null
}

export interface Node {
  id: string
  hostname: string
  is_leader: boolean
  last_seen: string
  run_count: number
}
