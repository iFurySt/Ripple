import { useEffect, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { FileText, Send, ChevronLeft, ChevronRight, Filter } from 'lucide-react'
import { dashboardApi } from '@/services/api'
import { formatDate, formatNumber } from '@/lib/utils'
import { ErrorDisplay } from '@/components/ErrorDisplay'
import type { NotionPage, DistributionJob } from '@/types/dashboard'

interface DetailedListDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  type: 'pages' | 'jobs'
  title: string
}

export function DetailedListDialog({ open, onOpenChange, type, title }: DetailedListDialogProps) {
  const [pages, setPages] = useState<NotionPage[]>([])
  const [jobs, setJobs] = useState<DistributionJob[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  // For jobs pagination
  const [currentPage, setCurrentPage] = useState(0)
  const [totalJobs, setTotalJobs] = useState(0)
  const [jobStatus, setJobStatus] = useState<string>('')
  const limit = 20

  const fetchData = async () => {
    if (!open) return
    
    setLoading(true)
    setError(null)
    
    try {
      if (type === 'pages') {
        const data = await dashboardApi.getAllPages()
        setPages(data)
      } else {
        const data = await dashboardApi.getJobs({
          limit,
          offset: currentPage * limit,
          status: jobStatus || undefined
        })
        setJobs(data.jobs)
        setTotalJobs(data.total)
      }
    } catch (err) {
      setError(`Failed to fetch ${type}`)
      console.error(`Error fetching ${type}:`, err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [open, type, currentPage, jobStatus])

  const handlePrevPage = () => {
    if (currentPage > 0) {
      setCurrentPage(currentPage - 1)
    }
  }

  const handleNextPage = () => {
    const maxPage = Math.ceil(totalJobs / limit) - 1
    if (currentPage < maxPage) {
      setCurrentPage(currentPage + 1)
    }
  }

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'done':
      case 'completed': return 'success'
      case 'failed': return 'destructive'
      case 'pending': return 'warning'
      case 'in progress': return 'warning'
      case 'draft': return 'secondary'
      default: return 'outline'
    }
  }

  const renderPages = () => {
    if (loading) {
      return (
        <div className="space-y-4">
          {[...Array(10)].map((_, i) => (
            <div key={i} className="animate-pulse border rounded-lg p-4">
              <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
              <div className="h-3 bg-muted rounded w-1/2 mb-2"></div>
              <div className="h-3 bg-muted rounded w-1/4"></div>
            </div>
          ))}
        </div>
      )
    }

    return (
      <div className="space-y-4 max-h-96 overflow-y-auto">
        {pages.map((page) => (
          <div key={page.id} className="border rounded-lg p-4 hover:bg-muted/50">
            <div className="flex items-start justify-between mb-2">
              <h4 className="font-medium">{page.title}</h4>
              <Badge variant={getStatusColor(page.status)}>
                {page.status}
              </Badge>
            </div>
            <div className="text-sm text-muted-foreground space-y-1">
              {page.owner && <p>Owner: {page.owner}</p>}
              <p>Updated: {formatDate(page.updated_at)}</p>
              {page.post_date && <p>Post Date: {formatDate(page.post_date)}</p>}
            </div>
            {page.platforms && page.platforms.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-2">
                {page.platforms.map((platform, index) => (
                  <Badge key={index} variant="outline" className="text-xs">
                    {platform}
                  </Badge>
                ))}
              </div>
            )}
            {page.tags && page.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-2">
                {page.tags.map((tag, index) => (
                  <Badge key={index} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    )
  }

  const renderJobs = () => {
    if (loading) {
      return (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="animate-pulse border rounded-lg p-4">
              <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
              <div className="h-3 bg-muted rounded w-1/2 mb-2"></div>
              <div className="h-3 bg-muted rounded w-1/4"></div>
            </div>
          ))}
        </div>
      )
    }

    return (
      <div className="space-y-4">
        {/* Status Filter */}
        <div className="flex items-center space-x-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select 
            value={jobStatus} 
            onChange={(e) => {
              setJobStatus(e.target.value)
              setCurrentPage(0)
            }}
            className="px-3 py-1 border rounded-md bg-background text-sm"
          >
            <option value="">All Status</option>
            <option value="pending">Pending</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        {/* Jobs List */}
        <div className="space-y-4 max-h-80 overflow-y-auto">
          {jobs.map((job) => (
            <div key={job.id} className="border rounded-lg p-4 hover:bg-muted/50">
              <div className="flex items-start justify-between mb-2">
                <h4 className="font-medium">
                  {job.page?.title || `Job #${job.id}`}
                </h4>
                <Badge variant={getStatusColor(job.status)}>
                  {job.status}
                </Badge>
              </div>
              <div className="text-sm text-muted-foreground space-y-1">
                <p>Platform: {job.platform?.display_name || job.platform?.name || 'Unknown'}</p>
                <p>Updated: {formatDate(job.updated_at)}</p>
                {job.published_at && (
                  <p className="text-green-600">Published: {formatDate(job.published_at)}</p>
                )}
                <p>Job ID: #{job.id}</p>
              </div>
              {job.error && (
                <ErrorDisplay 
                  error={job.error} 
                  compact={false}
                  className="mt-2"
                />
              )}
            </div>
          ))}
        </div>

        {/* Pagination */}
        {type === 'jobs' && totalJobs > limit && (
          <div className="flex items-center justify-between border-t pt-4">
            <p className="text-sm text-muted-foreground">
              Showing {currentPage * limit + 1} - {Math.min((currentPage + 1) * limit, totalJobs)} of {formatNumber(totalJobs)} jobs
            </p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handlePrevPage}
                disabled={currentPage === 0}
              >
                <ChevronLeft className="h-4 w-4" />
                Previous
              </Button>
              <span className="text-sm">
                Page {currentPage + 1} of {Math.ceil(totalJobs / limit)}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleNextPage}
                disabled={currentPage >= Math.ceil(totalJobs / limit) - 1}
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            {type === 'pages' ? (
              <FileText className="h-5 w-5" />
            ) : (
              <Send className="h-5 w-5" />
            )}
            <span>{title}</span>
          </DialogTitle>
          <DialogDescription>
            {type === 'pages' 
              ? `Browse all Notion pages in the system (${formatNumber(pages.length)} total)`
              : `Browse all distribution jobs${jobStatus ? ` with status: ${jobStatus}` : ''} (${formatNumber(totalJobs)} total)`
            }
          </DialogDescription>
        </DialogHeader>

        {error ? (
          <div className="text-center py-8">
            <p className="text-destructive">{error}</p>
            <Button onClick={fetchData} variant="outline" className="mt-4">
              Retry
            </Button>
          </div>
        ) : (
          <div className="overflow-hidden">
            {type === 'pages' ? renderPages() : renderJobs()}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}