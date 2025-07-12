import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { AlertTriangle, Check, RefreshCw, Filter, X } from 'lucide-react'
import { dashboardApi } from '@/services/api'
import { formatDate } from '@/lib/utils'
import { ErrorDisplay } from '@/components/ErrorDisplay'
import type { ErrorLog } from '@/types/dashboard'

export function ErrorLogs() {
  const [errors, setErrors] = useState<ErrorLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [limit, setLimit] = useState(20)
  const [filter, setFilter] = useState<'all' | 'unresolved' | 'resolved'>('unresolved')
  const [resolving, setResolving] = useState<number | null>(null)

  const fetchErrors = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await dashboardApi.getRecentErrors(limit)
      setErrors(data)
    } catch (err) {
      setError('Failed to fetch error logs')
      console.error('Error fetching errors:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleResolveError = async (errorId: number) => {
    try {
      setResolving(errorId)
      await dashboardApi.resolveError(errorId)
      // 更新本地状态
      setErrors(errors.map(err => 
        err.id === errorId 
          ? { ...err, resolved: true, resolved_at: new Date().toISOString() }
          : err
      ))
    } catch (err) {
      console.error('Error resolving error:', err)
    } finally {
      setResolving(null)
    }
  }

  useEffect(() => {
    fetchErrors()
  }, [limit])

  const filteredErrors = errors.filter(err => {
    if (filter === 'resolved') return err.resolved
    if (filter === 'unresolved') return !err.resolved
    return true
  })

  const getErrorLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error': return 'destructive'
      case 'warn': return 'warning'
      case 'info': return 'secondary'
      default: return 'outline'
    }
  }

  const getSourceColor = (source: string) => {
    const colors = ['default', 'secondary', 'outline']
    const hash = source.split('').reduce((a, b) => a + b.charCodeAt(0), 0)
    return colors[hash % colors.length] as any
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <div className="animate-pulse">
            <div className="h-6 bg-muted rounded w-1/4 mb-2"></div>
            <div className="h-4 bg-muted rounded w-1/3"></div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="animate-pulse border rounded-lg p-4">
                <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
                <div className="h-3 bg-muted rounded w-1/2 mb-2"></div>
                <div className="h-3 bg-muted rounded w-full"></div>
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
        <CardContent className="flex items-center justify-center py-8">
          <div className="text-center">
            <AlertTriangle className="h-8 w-8 text-destructive mx-auto mb-2" />
            <p className="text-destructive">{error}</p>
            <Button onClick={fetchErrors} variant="outline" className="mt-4">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Error Logs</h2>
        <div className="flex items-center space-x-4">
          {/* 过滤器 */}
          <div className="flex items-center space-x-2">
            <Filter className="h-4 w-4 text-muted-foreground" />
            <select 
              value={filter} 
              onChange={(e) => setFilter(e.target.value as any)}
              className="px-3 py-1 border rounded-md bg-background text-sm"
            >
              <option value="all">All</option>
              <option value="unresolved">Unresolved</option>
              <option value="resolved">Resolved</option>
            </select>
          </div>
          
          {/* 数量限制 */}
          <div className="flex items-center space-x-2">
            <span className="text-sm text-muted-foreground">Show</span>
            <select 
              value={limit} 
              onChange={(e) => setLimit(Number(e.target.value))}
              className="px-3 py-1 border rounded-md bg-background text-sm"
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
          </div>

          <Button onClick={fetchErrors} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* 统计信息 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Total Errors</span>
              <span className="text-lg font-bold">{errors.length}</span>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Unresolved</span>
              <span className="text-lg font-bold text-red-600">
                {errors.filter(e => !e.resolved).length}
              </span>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Resolved</span>
              <span className="text-lg font-bold text-green-600">
                {errors.filter(e => e.resolved).length}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 错误列表 */}
      <Card>
        <CardContent className="p-0">
          {filteredErrors.length === 0 ? (
            <div className="text-center py-8">
              <AlertTriangle className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
              <p className="text-muted-foreground">
                {filter === 'unresolved' ? 'No unresolved errors' : 
                 filter === 'resolved' ? 'No resolved errors' : 'No errors found'}
              </p>
            </div>
          ) : (
            <div className="divide-y">
              {filteredErrors.map((errorLog) => (
                <div key={errorLog.id} className="p-4 hover:bg-muted/50">
                  <div className="flex items-start justify-between">
                    <div className="flex-1 space-y-2">
                      {/* 错误标题和状态 */}
                      <div className="flex items-center space-x-2">
                        <Badge variant={getErrorLevelColor(errorLog.level)}>
                          {errorLog.level}
                        </Badge>
                        <Badge variant={getSourceColor(errorLog.source)}>
                          {errorLog.source}
                        </Badge>
                        {errorLog.platform_name && (
                          <Badge variant="outline">
                            {errorLog.platform_name}
                          </Badge>
                        )}
                        {errorLog.resolved ? (
                          <Badge variant="success">
                            <Check className="h-3 w-3 mr-1" />
                            Resolved
                          </Badge>
                        ) : (
                          <Badge variant="destructive">
                            <X className="h-3 w-3 mr-1" />
                            Unresolved
                          </Badge>
                        )}
                      </div>

                      {/* 错误标题 */}
                      <h4 className="font-medium">{errorLog.title}</h4>
                      
                      {/* 错误信息 */}
                      <ErrorDisplay 
                        error={errorLog.message} 
                        compact={true}
                        className="mt-2"
                      />

                      {/* 元数据 */}
                      <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                        <span>{formatDate(errorLog.created_at)}</span>
                        {errorLog.page && (
                          <span>Page: {errorLog.page.title}</span>
                        )}
                        {errorLog.job && (
                          <span>Job ID: {errorLog.job.id}</span>
                        )}
                      </div>

                      {/* 解决时间 */}
                      {errorLog.resolved && errorLog.resolved_at && (
                        <div className="text-xs text-green-600">
                          Resolved at {formatDate(errorLog.resolved_at)}
                        </div>
                      )}
                    </div>

                    {/* 操作按钮 */}
                    <div className="ml-4">
                      {!errorLog.resolved && (
                        <Button
                          onClick={() => handleResolveError(errorLog.id)}
                          disabled={resolving === errorLog.id}
                          size="sm"
                          variant="outline"
                        >
                          {resolving === errorLog.id ? (
                            <RefreshCw className="h-3 w-3 animate-spin" />
                          ) : (
                            <Check className="h-3 w-3" />
                          )}
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* 堆栈信息（可展开） */}
                  {errorLog.stack_trace && (
                    <details className="mt-2">
                      <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
                        Stack Trace
                      </summary>
                      <pre className="mt-2 text-xs bg-muted p-2 rounded overflow-x-auto">
                        {errorLog.stack_trace}
                      </pre>
                    </details>
                  )}

                  {/* 上下文信息 */}
                  {errorLog.context && (
                    <details className="mt-2">
                      <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
                        Context
                      </summary>
                      <pre className="mt-2 text-xs bg-muted p-2 rounded overflow-x-auto">
                        {JSON.stringify(JSON.parse(errorLog.context), null, 2)}
                      </pre>
                    </details>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}