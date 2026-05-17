export type JobStatus = 'pending' | 'running' | 'success' | 'failed'

export interface User {
  id: string
  github_id: number
  avatar_url: string
  created_at: string
}

export interface Repo {
  id: number
  full_name: string
  private: boolean
  clone_url: string
}

export interface Project {
  id: string
  user_id: string
  repo_full_name: string
  branch: string
  clone_url: string
  webhook_id: number
  created_at: string
}

export interface Job {
  id: string
  status: JobStatus
  commit_sha: string
  commit_msg: string
  log_url?: string
  created_at: string
  finished_at?: string
}
