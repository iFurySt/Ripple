import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { RefreshCw, FileText, Send, CheckCircle, XCircle, Clock, AlertTriangle, ExternalLink } from 'lucide-react'
import { dashboardApi } from '@/services/api'
import { formatDate, formatNumber, getSuccessRate } from '@/lib/utils'
import { RecentPages } from '@/components/RecentPages'
import { RecentJobs } from '@/components/RecentJobs'
import { DetailedListDialog } from '@/components/DetailedListDialog'
import type { DashboardSummary } from '@/types/dashboard'

export function DashboardSummary() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogType, setDialogType] = useState<'pages' | 'jobs'>('pages')
  const [dialogTitle, setDialogTitle] = useState('')

  const fetchSummary = async () => {
    try {
      setError(null)
      const data = await dashboardApi.getSummary()
      setSummary(data)
    } catch (err) {
      setError('Failed to fetch dashboard summary')
      console.error('Error fetching summary:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await dashboardApi.updateStats()
      await fetchSummary()
    } catch (err) {
      setError('Failed to refresh statistics')
    } finally {
      setRefreshing(false)
    }
  }

  const openDialog = (type: 'pages' | 'jobs', title: string) => {
    setDialogType(type)
    setDialogTitle(title)
    setDialogOpen(true)
  }

  useEffect(() => {
    fetchSummary()
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[...Array(8)].map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardHeader className="pb-3">
              <div className="h-4 bg-muted rounded w-3/4"></div>
            </CardHeader>
            <CardContent>
              <div className="h-8 bg-muted rounded w-1/2 mb-2"></div>
              <div className="h-3 bg-muted rounded w-full"></div>
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (error || !summary) {
    return (
      <Card className="col-span-full">
        <CardContent className="flex items-center justify-center py-8">
          <div className="text-center">
            <AlertTriangle className="h-8 w-8 text-destructive mx-auto mb-2" />
            <p className="text-destructive">{error || 'No data available'}</p>
            <Button onClick={fetchSummary} variant="outline" className="mt-4">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  const successRate = getSuccessRate(summary.successful_jobs_today, summary.total_jobs_today)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Dashboard Overview</h2>
        <Button onClick={handleRefresh} disabled={refreshing} variant="outline">
          <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh Stats
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Pages */}
        <Card 
          className="cursor-pointer hover:shadow-md transition-all duration-200"
          onClick={() => openDialog('pages', 'All Pages')}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Pages</CardTitle>
            <div className="flex items-center space-x-1">
              <FileText className="h-4 w-4 text-muted-foreground" />
              <ExternalLink className="h-3 w-3 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(summary.total_pages)}</div>
            <p className="text-xs text-muted-foreground">Notion pages synced • Click to view</p>
          </CardContent>
        </Card>

        {/* Today's Jobs */}
        <Card 
          className="cursor-pointer hover:shadow-md transition-all duration-200"
          onClick={() => openDialog('jobs', 'All Jobs')}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Today's Jobs</CardTitle>
            <div className="flex items-center space-x-1">
              <Send className="h-4 w-4 text-muted-foreground" />
              <ExternalLink className="h-3 w-3 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(summary.total_jobs_today)}</div>
            <p className="text-xs text-muted-foreground">
              {successRate}% success rate • Click to view
            </p>
          </CardContent>
        </Card>

        {/* Successful Jobs */}
        <Card 
          className="cursor-pointer hover:shadow-md transition-all duration-200"
          onClick={() => openDialog('jobs', 'Successful Jobs')}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Successful</CardTitle>
            <div className="flex items-center space-x-1">
              <CheckCircle className="h-4 w-4 text-green-600" />
              <ExternalLink className="h-3 w-3 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {formatNumber(summary.successful_jobs_today)}
            </div>
            <p className="text-xs text-muted-foreground">Jobs completed today • Click to view</p>
          </CardContent>
        </Card>

        {/* Failed Jobs */}
        <Card 
          className="cursor-pointer hover:shadow-md transition-all duration-200"
          onClick={() => openDialog('jobs', 'Failed Jobs')}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Failed</CardTitle>
            <div className="flex items-center space-x-1">
              <XCircle className="h-4 w-4 text-red-600" />
              <ExternalLink className="h-3 w-3 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">
              {formatNumber(summary.failed_jobs_today)}
            </div>
            <p className="text-xs text-muted-foreground">Jobs failed today • Click to view</p>
          </CardContent>
        </Card>

        {/* Pending Jobs */}
        <Card 
          className="cursor-pointer hover:shadow-md transition-all duration-200"
          onClick={() => openDialog('jobs', 'Pending Jobs')}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pending</CardTitle>
            <div className="flex items-center space-x-1">
              <Clock className="h-4 w-4 text-yellow-600" />
              <ExternalLink className="h-3 w-3 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">
              {formatNumber(summary.pending_jobs_count)}
            </div>
            <p className="text-xs text-muted-foreground">Jobs in queue • Click to view</p>
          </CardContent>
        </Card>

        {/* Active Platforms */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Platforms</CardTitle>
            <Badge variant="success">{summary.active_platforms_count}</Badge>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {summary.active_platforms_count}/{summary.total_platforms_count}
            </div>
            <p className="text-xs text-muted-foreground">Platforms enabled</p>
          </CardContent>
        </Card>

        {/* Unresolved Errors */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Unresolved Errors</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">
              {formatNumber(summary.unresolved_errors_count)}
            </div>
            <p className="text-xs text-muted-foreground">Need attention</p>
          </CardContent>
        </Card>

        {/* Last Activity */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Last Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {summary.last_sync_time && (
                <div className="text-xs">
                  <span className="text-muted-foreground">Sync: </span>
                  {formatDate(summary.last_sync_time)}
                </div>
              )}
              {summary.last_publish_time && (
                <div className="text-xs">
                  <span className="text-muted-foreground">Publish: </span>
                  {formatDate(summary.last_publish_time)}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Activity Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RecentPages 
          limit={5} 
          onViewAll={() => openDialog('pages', 'All Pages')} 
        />
        <RecentJobs 
          limit={5} 
          onViewAll={() => openDialog('jobs', 'All Jobs')} 
        />
      </div>

      {/* Quick Job Status Views */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <RecentJobs 
          limit={3} 
          statusFilter="completed" 
          onViewAll={() => openDialog('jobs', 'Successful Jobs')} 
        />
        <RecentJobs 
          limit={3} 
          statusFilter="failed" 
          onViewAll={() => openDialog('jobs', 'Failed Jobs')} 
        />
        <RecentJobs 
          limit={3} 
          statusFilter="pending" 
          onViewAll={() => openDialog('jobs', 'Pending Jobs')} 
        />
      </div>

      {/* Detail Dialog */}
      <DetailedListDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        type={dialogType}
        title={dialogTitle}
      />
    </div>
  )
}