import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { TrendingUp, TrendingDown, AlertCircle } from 'lucide-react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import { dashboardApi } from '@/services/api'
import { formatDate, formatNumber, getSuccessRate } from '@/lib/utils'
import type { PlatformStats } from '@/types/dashboard'

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8']

export function PlatformStats() {
  const [stats, setStats] = useState<PlatformStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [days, setDays] = useState(7)

  const fetchStats = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await dashboardApi.getPlatformStats(days)
      setStats(data)
    } catch (err) {
      setError('Failed to fetch platform statistics')
      console.error('Error fetching platform stats:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStats()
  }, [days])

  // 按平台聚合数据
  const platformSummary = stats.reduce((acc, stat) => {
    const existing = acc.find(p => p.platform_name === stat.platform_name)
    if (existing) {
      existing.total_jobs += stat.total_jobs
      existing.successful_jobs += stat.successful_jobs
      existing.failed_jobs += stat.failed_jobs
      existing.pending_jobs += stat.pending_jobs
      existing.error_count += stat.error_count
    } else {
      acc.push({
        platform_name: stat.platform_name,
        total_jobs: stat.total_jobs,
        successful_jobs: stat.successful_jobs,
        failed_jobs: stat.failed_jobs,
        pending_jobs: stat.pending_jobs,
        error_count: stat.error_count,
        last_success_at: stat.last_success_at,
        last_failure_at: stat.last_failure_at,
        platform: stat.platform
      })
    }
    return acc
  }, [] as any[])

  // 饼图数据
  const pieData = platformSummary.map(platform => ({
    name: platform.platform_name,
    value: platform.total_jobs
  }))

  // 柱状图数据
  const barData = platformSummary.map(platform => ({
    name: platform.platform_name,
    successful: platform.successful_jobs,
    failed: platform.failed_jobs,
    pending: platform.pending_jobs
  }))

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-muted rounded w-1/4"></div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="h-96 bg-muted rounded"></div>
            <div className="h-96 bg-muted rounded"></div>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <div className="text-center">
            <AlertCircle className="h-8 w-8 text-destructive mx-auto mb-2" />
            <p className="text-destructive">{error}</p>
            <Button onClick={fetchStats} variant="outline" className="mt-4">
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
        <h2 className="text-2xl font-bold">Platform Statistics</h2>
        <div className="flex items-center space-x-2">
          <span className="text-sm text-muted-foreground">Last</span>
          <select 
            value={days} 
            onChange={(e) => setDays(Number(e.target.value))}
            className="px-3 py-1 border rounded-md bg-background"
          >
            <option value={7}>7 days</option>
            <option value={14}>14 days</option>
            <option value={30}>30 days</option>
          </select>
        </div>
      </div>

      {/* 平台概览卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {platformSummary.map((platform) => {
          const successRate = getSuccessRate(platform.successful_jobs, platform.total_jobs)
          const isHealthy = successRate >= 80
          
          return (
            <Card key={platform.platform_name}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{platform.platform_name}</CardTitle>
                <Badge variant={platform.platform.enabled ? "success" : "secondary"}>
                  {platform.platform.enabled ? "Active" : "Inactive"}
                </Badge>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Total Jobs</span>
                    <span className="font-medium">{formatNumber(platform.total_jobs)}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Success Rate</span>
                    <div className="flex items-center">
                      {isHealthy ? (
                        <TrendingUp className="h-3 w-3 text-green-600 mr-1" />
                      ) : (
                        <TrendingDown className="h-3 w-3 text-red-600 mr-1" />
                      )}
                      <span className={`font-medium ${isHealthy ? 'text-green-600' : 'text-red-600'}`}>
                        {successRate}%
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Pending</span>
                    <span className="font-medium text-yellow-600">{platform.pending_jobs}</span>
                  </div>
                  {platform.error_count > 0 && (
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">Errors</span>
                      <span className="font-medium text-red-600">{platform.error_count}</span>
                    </div>
                  )}
                  {platform.last_success_at && (
                    <div className="text-xs text-muted-foreground">
                      Last success: {formatDate(platform.last_success_at)}
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* 图表区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 工作分布饼图 */}
        <Card>
          <CardHeader>
            <CardTitle>Jobs Distribution by Platform</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {pieData.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value) => [formatNumber(value as number), 'Jobs']} />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* 工作状态柱状图 */}
        <Card>
          <CardHeader>
            <CardTitle>Job Status by Platform</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={barData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip formatter={(value) => [formatNumber(value as number), 'Jobs']} />
                <Bar dataKey="successful" stackId="a" fill="#22c55e" name="Successful" />
                <Bar dataKey="failed" stackId="a" fill="#ef4444" name="Failed" />
                <Bar dataKey="pending" stackId="a" fill="#eab308" name="Pending" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* 详细数据表格 */}
      {stats.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Detailed Statistics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">Date</th>
                    <th className="text-left p-2">Platform</th>
                    <th className="text-right p-2">Total</th>
                    <th className="text-right p-2">Success</th>
                    <th className="text-right p-2">Failed</th>
                    <th className="text-right p-2">Pending</th>
                    <th className="text-right p-2">Success Rate</th>
                    <th className="text-right p-2">Errors</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.map((stat) => {
                    const successRate = getSuccessRate(stat.successful_jobs, stat.total_jobs)
                    return (
                      <tr key={`${stat.date}-${stat.platform_name}`} className="border-b hover:bg-muted/50">
                        <td className="p-2">{new Date(stat.date).toLocaleDateString()}</td>
                        <td className="p-2">
                          <div className="flex items-center">
                            <span>{stat.platform_name}</span>
                            <Badge 
                              variant={stat.platform.enabled ? "success" : "secondary"} 
                              className="ml-2"
                            >
                              {stat.platform.enabled ? "Active" : "Inactive"}
                            </Badge>
                          </div>
                        </td>
                        <td className="text-right p-2">{formatNumber(stat.total_jobs)}</td>
                        <td className="text-right p-2 text-green-600">{formatNumber(stat.successful_jobs)}</td>
                        <td className="text-right p-2 text-red-600">{formatNumber(stat.failed_jobs)}</td>
                        <td className="text-right p-2 text-yellow-600">{formatNumber(stat.pending_jobs)}</td>
                        <td className={`text-right p-2 ${successRate >= 80 ? 'text-green-600' : 'text-red-600'}`}>
                          {successRate}%
                        </td>
                        <td className="text-right p-2">{stat.error_count}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}