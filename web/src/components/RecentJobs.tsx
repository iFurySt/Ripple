import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Send, ExternalLink, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import { dashboardApi } from '@/services/api'
import { formatDate } from '@/lib/utils'
import { ErrorDisplay } from '@/components/ErrorDisplay'
import type { DistributionJob } from '@/types/dashboard'

interface RecentJobsProps {
  limit?: number
  showHeader?: boolean
  onViewAll?: () => void
  statusFilter?: string
}

export function RecentJobs({ limit = 5, showHeader = true, onViewAll, statusFilter }: RecentJobsProps) {
  const [jobs, setJobs] = useState<DistributionJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchJobs = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await dashboardApi.getRecentJobs(limit)
      let filteredJobs = data
      if (statusFilter) {
        filteredJobs = data.filter(job => job.status === statusFilter)
      }
      setJobs(filteredJobs)
    } catch (err) {
      setError('Failed to fetch recent jobs')
      console.error('Error fetching recent jobs:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchJobs()
  }, [limit, statusFilter])

  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
        return <CheckCircle className="h-4 w-4 text-green-600" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'pending':
        return <Clock className="h-4 w-4 text-yellow-600" />
      default:
        return <AlertCircle className="h-4 w-4 text-gray-600" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed': return 'success'
      case 'failed': return 'destructive'
      case 'pending': return 'warning'
      default: return 'secondary'
    }
  }

  if (loading) {
    return (
      <Card>
        {showHeader && (
          <CardHeader>
            <CardTitle className="text-lg">
              Recent Jobs {statusFilter && `(${statusFilter})`}
            </CardTitle>
          </CardHeader>
        )}
        <CardContent>
          <div className="space-y-3">
            {[...Array(limit)].map((_, i) => (
              <div key={i} className="animate-pulse flex items-center space-x-3 p-3 border rounded-lg">
                <div className="h-4 w-4 bg-muted rounded"></div>
                <div className="flex-1">
                  <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
                  <div className="h-3 bg-muted rounded w-1/2"></div>
                </div>
                <div className="h-6 w-20 bg-muted rounded"></div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        {showHeader && (
          <CardHeader>
            <CardTitle className="text-lg">
              Recent Jobs {statusFilter && `(${statusFilter})`}
            </CardTitle>
          </CardHeader>
        )}
        <CardContent>
          <div className="text-center py-4">
            <p className="text-destructive text-sm">{error}</p>
            <Button onClick={fetchJobs} variant="outline" size="sm" className="mt-2">
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  const title = statusFilter 
    ? `Recent ${statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)} Jobs`
    : 'Recent Jobs'

  return (
    <Card>
      {showHeader && (
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-lg">{title}</CardTitle>
          {onViewAll && (
            <Button variant="ghost" size="sm" onClick={onViewAll}>
              <ExternalLink className="h-4 w-4 mr-1" />
              View All
            </Button>
          )}
        </CardHeader>
      )}
      <CardContent>
        {jobs.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Send className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No jobs found</p>
          </div>
        ) : (
          <div className="space-y-3">
            {jobs.map((job) => (
              <div
                key={job.id}
                className="flex items-center space-x-3 p-3 border rounded-lg hover:bg-muted/50 transition-colors"
              >
                {getStatusIcon(job.status)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2 mb-1">
                    <h4 className="font-medium truncate">
                      {job.page?.title || `Job #${job.id}`}
                    </h4>
                    <Badge variant={getStatusColor(job.status)}>
                      {job.status}
                    </Badge>
                  </div>
                  <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                    <span className="flex items-center">
                      Platform: {job.platform?.display_name || job.platform?.name || 'Unknown'}
                    </span>
                    <span className="flex items-center">
                      <Clock className="h-3 w-3 mr-1" />
                      {formatDate(job.updated_at)}
                    </span>
                    {job.published_at && (
                      <span className="flex items-center text-green-600">
                        Published: {formatDate(job.published_at)}
                      </span>
                    )}
                  </div>
                  {job.error && (
                    <ErrorDisplay 
                      error={job.error} 
                      compact={true}
                      className="mt-2"
                    />
                  )}
                </div>
                <div className="flex flex-col items-end space-y-1">
                  <span className="text-xs text-muted-foreground">
                    Job #{job.id}
                  </span>
                  {job.page?.notion_id && (
                    <span className="text-xs text-muted-foreground font-mono">
                      {job.page.notion_id.slice(0, 8)}...
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}