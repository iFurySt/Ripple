import axios, { type AxiosResponse } from 'axios'
import type {
  DashboardSummary,
  PlatformStats,
  ErrorLog,
  SystemStats,
  NotionPage,
  DistributionJob,
  ApiResponse
} from '@/types/dashboard'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export const dashboardApi = {
  // Get dashboard summary
  getSummary: async (): Promise<DashboardSummary> => {
    const response = await api.get<ApiResponse<DashboardSummary>>('/dashboard/summary')
    return response.data.summary
  },

  // Get platform statistics
  getPlatformStats: async (days: number = 7): Promise<PlatformStats[]> => {
    const response = await api.get<ApiResponse<PlatformStats[]>>(`/dashboard/platform-stats?days=${days}`)
    return response.data.stats
  },

  // Get recent errors
  getRecentErrors: async (limit: number = 20): Promise<ErrorLog[]> => {
    const response = await api.get<ApiResponse<ErrorLog[]>>(`/dashboard/recent-errors?limit=${limit}`)
    return response.data.errors
  },

  // Get system statistics
  getSystemStats: async (days: number = 7): Promise<SystemStats[]> => {
    const response = await api.get<ApiResponse<SystemStats[]>>(`/dashboard/system-stats?days=${days}`)
    return response.data.stats
  },

  // Get recent pages
  getRecentPages: async (limit: number = 5): Promise<NotionPage[]> => {
    const response = await api.get<ApiResponse<NotionPage[]>>(`/dashboard/recent-pages?limit=${limit}`)
    return response.data.pages
  },

  // Get recent jobs
  getRecentJobs: async (limit: number = 5): Promise<DistributionJob[]> => {
    const response = await api.get<ApiResponse<DistributionJob[]>>(`/dashboard/recent-jobs?limit=${limit}`)
    return response.data.jobs
  },

  // Get all pages
  getAllPages: async (): Promise<NotionPage[]> => {
    const response = await api.get<ApiResponse<NotionPage[]>>('/notion/pages')
    return response.data.pages
  },

  // Get jobs with pagination and filtering
  getJobs: async (params: {
    limit?: number
    offset?: number
    status?: string
  } = {}): Promise<{
    jobs: DistributionJob[]
    total: number
    limit: number
    offset: number
  }> => {
    const queryParams = new URLSearchParams()
    if (params.limit) queryParams.append('limit', params.limit.toString())
    if (params.offset) queryParams.append('offset', params.offset.toString())
    if (params.status) queryParams.append('status', params.status)
    
    const response = await api.get<{
      jobs: DistributionJob[]
      total: number
      limit: number
      offset: number
    }>(`/dashboard/jobs?${queryParams}`)
    return response.data
  },

  // Update statistics
  updateStats: async (): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>('/dashboard/update-stats')
    return response.data
  },

  // Resolve error
  resolveError: async (errorId: number): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(`/dashboard/resolve-error/${errorId}`)
    return response.data
  },

  // Republish job
  republishJob: async (jobId: number): Promise<{ message: string; result?: any }> => {
    const response = await api.post<{ message: string; result?: any }>(`/dashboard/republish-job/${jobId}`)
    return response.data
  },
}

export default api