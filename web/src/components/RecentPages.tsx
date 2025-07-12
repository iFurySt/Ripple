import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { FileText, ExternalLink, Clock, Calendar } from 'lucide-react'
import { dashboardApi } from '@/services/api'
import { formatDate } from '@/lib/utils'
import type { NotionPage } from '@/types/dashboard'

interface RecentPagesProps {
  limit?: number
  showHeader?: boolean
  onViewAll?: () => void
}

export function RecentPages({ limit = 5, showHeader = true, onViewAll }: RecentPagesProps) {
  const [pages, setPages] = useState<NotionPage[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchPages = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await dashboardApi.getRecentPages(limit)
      setPages(data)
    } catch (err) {
      setError('Failed to fetch recent pages')
      console.error('Error fetching recent pages:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchPages()
  }, [limit])

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'done': return 'success'
      case 'in progress': return 'warning'
      case 'draft': return 'secondary'
      default: return 'outline'
    }
  }

  if (loading) {
    return (
      <Card>
        {showHeader && (
          <CardHeader>
            <CardTitle className="text-lg">Recent Pages</CardTitle>
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
                <div className="h-6 w-16 bg-muted rounded"></div>
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
            <CardTitle className="text-lg">Recent Pages</CardTitle>
          </CardHeader>
        )}
        <CardContent>
          <div className="text-center py-4">
            <p className="text-destructive text-sm">{error}</p>
            <Button onClick={fetchPages} variant="outline" size="sm" className="mt-2">
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      {showHeader && (
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-lg">Recent Pages</CardTitle>
          {onViewAll && (
            <Button variant="ghost" size="sm" onClick={onViewAll}>
              <ExternalLink className="h-4 w-4 mr-1" />
              View All
            </Button>
          )}
        </CardHeader>
      )}
      <CardContent>
        {pages.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <FileText className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No pages found</p>
          </div>
        ) : (
          <div className="space-y-3">
            {pages.map((page) => (
              <div
                key={page.id}
                className="flex items-center space-x-3 p-3 border rounded-lg hover:bg-muted/50 transition-colors"
              >
                <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2 mb-1">
                    <h4 className="font-medium truncate">{page.title}</h4>
                    <Badge variant={getStatusColor(page.status)}>
                      {page.status}
                    </Badge>
                  </div>
                  <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                    {page.owner && (
                      <span className="flex items-center">
                        Owner: {page.owner}
                      </span>
                    )}
                    <span className="flex items-center">
                      <Clock className="h-3 w-3 mr-1" />
                      {formatDate(page.updated_at)}
                    </span>
                    {page.post_date && (
                      <span className="flex items-center">
                        <Calendar className="h-3 w-3 mr-1" />
                        {formatDate(page.post_date)}
                      </span>
                    )}
                  </div>
                  {page.tags && page.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-2">
                      {page.tags.slice(0, 3).map((tag, index) => (
                        <Badge key={index} variant="outline" className="text-xs">
                          {tag}
                        </Badge>
                      ))}
                      {page.tags.length > 3 && (
                        <Badge variant="outline" className="text-xs">
                          +{page.tags.length - 3}
                        </Badge>
                      )}
                    </div>
                  )}
                </div>
                {page.platforms && page.platforms.length > 0 && (
                  <div className="flex flex-col items-end space-y-1">
                    <span className="text-xs text-muted-foreground">
                      {page.platforms.length} platform{page.platforms.length > 1 ? 's' : ''}
                    </span>
                    <div className="flex flex-wrap gap-1 justify-end">
                      {page.platforms.slice(0, 2).map((platform, index) => (
                        <Badge key={index} variant="secondary" className="text-xs">
                          {platform}
                        </Badge>
                      ))}
                      {page.platforms.length > 2 && (
                        <Badge variant="secondary" className="text-xs">
                          +{page.platforms.length - 2}
                        </Badge>
                      )}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}