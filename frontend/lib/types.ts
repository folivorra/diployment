export type JobStatus = 'pending' | 'building' | 'deploying' | 'success' | 'failed'
export type Phase = 'build' | 'deploy'

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
  ssh_host: string
  ssh_port: number
  ssh_user: string
  deploy_restart_cmd: string
  deploy_workdir: string
}

export interface Job {
  id: string
  status: JobStatus
  commit_sha: string
  commit_msg: string
  created_at: string
  build_log_url?: string
  build_started_at?: string
  build_finished_at?: string
  deploy_log_url?: string
  deploy_started_at?: string
  deploy_finished_at?: string
}
