import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { TrendingUp, TrendingDown, AlertCircle } from 'lucide-react'
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { dashboardApi } from '@/services/api'
import { formatNumber } from '@/lib/utils'
import type { SystemStats } from '@/types/dashboard'

export function SystemTrends() {
  const [stats, setStats] = useState<SystemStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [days, setDays] = useState(7)

  const fetchStats = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await dashboardApi.getSystemStats(days)
      setStats(data.reverse()) // 按时间正序排列
    } catch (err) {
      setError('Failed to fetch system statistics')
      console.error('Error fetching system stats:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStats()
  }, [days])

  // 计算趋势
  const calculateTrend = (data: number[]) => {
    if (data.length < 2) return 0
    const recent = data.slice(-3).reduce((a, b) => a + b, 0) / Math.min(3, data.length)
    const earlier = data.slice(0, -3).reduce((a, b) => a + b, 0) / Math.max(1, data.length - 3)
    return ((recent - earlier) / Math.max(1, earlier)) * 100
  }

  // 准备图表数据
  const chartData = stats.map(stat => ({
    date: new Date(stat.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    totalJobs: stat.total_distribution_jobs,
    successful: stat.successful_jobs,
    failed: stat.failed_jobs,
    pending: stat.pending_jobs,
    successRate: stat.total_distribution_jobs > 0 
      ? Math.round((stat.successful_jobs / stat.total_distribution_jobs) * 100) 
      : 0,
    totalPages: stat.total_notion_pages,
    activePlatforms: stat.active_platforms
  }))

  // 计算关键指标趋势
  const totalJobsTrend = calculateTrend(stats.map(s => s.total_distribution_jobs))
  const successRateTrend = calculateTrend(stats.map(s => 
    s.total_distribution_jobs > 0 ? (s.successful_jobs / s.total_distribution_jobs) * 100 : 0
  ))
  const pagesTrend = calculateTrend(stats.map(s => s.total_notion_pages))

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-muted rounded w-1/4"></div>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-24 bg-muted rounded"></div>
            ))}
          </div>
          <div className="h-96 bg-muted rounded"></div>
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

  const latest = stats[stats.length - 1]
  
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">System Trends</h2>
        <div className="flex items-center space-x-2">
          <span className="text-sm text-muted-foreground">Period</span>
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

      {/* 趋势指标卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* 总任务数趋势 */}
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Jobs</p>
                <p className="text-2xl font-bold">{latest ? formatNumber(latest.total_distribution_jobs) : '0'}</p>
              </div>
              <div className="flex items-center">
                {totalJobsTrend > 0 ? (
                  <TrendingUp className="h-4 w-4 text-green-600 mr-1" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-600 mr-1" />
                )}
                <span className={`text-sm font-medium ${totalJobsTrend > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {Math.abs(totalJobsTrend).toFixed(1)}%
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 成功率趋势 */}
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Success Rate</p>
                <p className="text-2xl font-bold">
                  {latest && latest.total_distribution_jobs > 0 
                    ? Math.round((latest.successful_jobs / latest.total_distribution_jobs) * 100)
                    : 0}%
                </p>
              </div>
              <div className="flex items-center">
                {successRateTrend > 0 ? (
                  <TrendingUp className="h-4 w-4 text-green-600 mr-1" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-600 mr-1" />
                )}
                <span className={`text-sm font-medium ${successRateTrend > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {Math.abs(successRateTrend).toFixed(1)}%
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 页面数趋势 */}
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Pages</p>
                <p className="text-2xl font-bold">{latest ? formatNumber(latest.total_notion_pages) : '0'}</p>
              </div>
              <div className="flex items-center">
                {pagesTrend > 0 ? (
                  <TrendingUp className="h-4 w-4 text-green-600 mr-1" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-600 mr-1" />
                )}
                <span className={`text-sm font-medium ${pagesTrend > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {Math.abs(pagesTrend).toFixed(1)}%
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 工作状态趋势图 */}
      <Card>
        <CardHeader>
          <CardTitle>Job Status Trends</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip formatter={(value) => [formatNumber(value as number), '']} />
              <Legend />
              <Area 
                type="monotone" 
                dataKey="successful" 
                stackId="1" 
                stroke="#22c55e" 
                fill="#22c55e" 
                name="Successful"
              />
              <Area 
                type="monotone" 
                dataKey="failed" 
                stackId="1" 
                stroke="#ef4444" 
                fill="#ef4444" 
                name="Failed"
              />
              <Area 
                type="monotone" 
                dataKey="pending" 
                stackId="1" 
                stroke="#eab308" 
                fill="#eab308" 
                name="Pending"
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* 成功率和页面数趋势 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 成功率趋势 */}
        <Card>
          <CardHeader>
            <CardTitle>Success Rate Trend</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={250}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis domain={[0, 100]} />
                <Tooltip formatter={(value) => [`${value}%`, 'Success Rate']} />
                <Line 
                  type="monotone" 
                  dataKey="successRate" 
                  stroke="#22c55e" 
                  strokeWidth={2}
                  dot={{ fill: '#22c55e', strokeWidth: 2 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* 页面和平台趋势 */}
        <Card>
          <CardHeader>
            <CardTitle>Pages & Platforms</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={250}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip formatter={(value) => [formatNumber(value as number), '']} />
                <Legend />
                <Line 
                  type="monotone" 
                  dataKey="totalPages" 
                  stroke="#3b82f6" 
                  strokeWidth={2}
                  name="Total Pages"
                />
                <Line 
                  type="monotone" 
                  dataKey="activePlatforms" 
                  stroke="#8b5cf6" 
                  strokeWidth={2}
                  name="Active Platforms"
                />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* 详细数据表格 */}
      {stats.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Historical Data</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">Date</th>
                    <th className="text-right p-2">Pages</th>
                    <th className="text-right p-2">Total Jobs</th>
                    <th className="text-right p-2">Successful</th>
                    <th className="text-right p-2">Failed</th>
                    <th className="text-right p-2">Pending</th>
                    <th className="text-right p-2">Success Rate</th>
                    <th className="text-right p-2">Active Platforms</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.map((stat) => {
                    const successRate = stat.total_distribution_jobs > 0 
                      ? Math.round((stat.successful_jobs / stat.total_distribution_jobs) * 100)
                      : 0
                    
                    return (
                      <tr key={stat.date} className="border-b hover:bg-muted/50">
                        <td className="p-2">{new Date(stat.date).toLocaleDateString()}</td>
                        <td className="text-right p-2">{formatNumber(stat.total_notion_pages)}</td>
                        <td className="text-right p-2">{formatNumber(stat.total_distribution_jobs)}</td>
                        <td className="text-right p-2 text-green-600">{formatNumber(stat.successful_jobs)}</td>
                        <td className="text-right p-2 text-red-600">{formatNumber(stat.failed_jobs)}</td>
                        <td className="text-right p-2 text-yellow-600">{formatNumber(stat.pending_jobs)}</td>
                        <td className={`text-right p-2 ${successRate >= 80 ? 'text-green-600' : 'text-red-600'}`}>
                          {successRate}%
                        </td>
                        <td className="text-right p-2">{stat.active_platforms}/{stat.total_platforms}</td>
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