export interface DashboardSummary {
  id: number
  total_pages: number
  total_jobs_today: number
  successful_jobs_today: number
  failed_jobs_today: number
  pending_jobs_count: number
  active_platforms_count: number
  total_platforms_count: number
  last_sync_time?: string
  last_publish_time?: string
  unresolved_errors_count: number
  avg_process_time_today: number
  updated_at: string
}

export interface Platform {
  id: number
  name: string
  display_name: string
  config: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface PlatformStats {
  id: number
  date: string
  platform_id: number
  platform_name: string
  total_jobs: number
  successful_jobs: number
  failed_jobs: number
  pending_jobs: number
  avg_process_time: number
  last_success_at?: string
  last_failure_at?: string
  error_count: number
  created_at: string
  updated_at: string
  platform: Platform
}

export interface NotionPage {
  id: number
  notion_id: string
  title: string
  en_title: string
  content: string
  summary: string
  tags: string[]
  status: string
  post_date?: string
  owner: string
  platforms: string[]
  content_type: string[]
  properties: string
  last_modified: string
  created_at: string
  updated_at: string
}

export interface DistributionJob {
  id: number
  page_id: number
  platform_id: number
  status: string
  content: string
  error: string
  published_at?: string
  created_at: string
  updated_at: string
  page: NotionPage
  platform: Platform
}

export interface ErrorLog {
  id: number
  level: string
  source: string
  platform_name: string
  page_id?: number
  job_id?: number
  title: string
  message: string
  stack_trace: string
  context: string
  resolved: boolean
  resolved_at?: string
  created_at: string
  updated_at: string
  page?: NotionPage
  job?: DistributionJob
}

export interface SystemStats {
  id: number
  date: string
  total_notion_pages: number
  total_distribution_jobs: number
  successful_jobs: number
  failed_jobs: number
  pending_jobs: number
  total_platforms: number
  active_platforms: number
  created_at: string
  updated_at: string
}

export interface MetricsSample {
  id: number
  metric_name: string
  metric_type: string
  value: number
  tags: string
  timestamp: string
  created_at: string
}

export interface ApiResponse<T> {
  [key: string]: T
}

export interface ErrorResponse {
  error: string
}