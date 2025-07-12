import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DashboardSummary } from '@/components/DashboardSummary'
import { PlatformStats } from '@/components/PlatformStats'
import { ErrorLogs } from '@/components/ErrorLogs'
import { SystemTrends } from '@/components/SystemTrends'
import { 
  BarChart3, 
  TrendingUp, 
  AlertTriangle, 
  Activity 
} from 'lucide-react'

export function NavigationTabs() {
  const location = useLocation()
  const navigate = useNavigate()
  
  // 从URL路径中获取当前活动的标签
  const getCurrentTab = () => {
    const path = location.pathname.replace('/', '') || 'overview'
    return ['overview', 'platforms', 'trends', 'errors'].includes(path) ? path : 'overview'
  }
  
  const activeTab = getCurrentTab()
  
  const handleTabChange = (value: string) => {
    navigate(`/${value}`)
  }

  // 更新页面标题
  useEffect(() => {
    const titles = {
      overview: 'Overview - Ripple Dashboard',
      platforms: 'Platforms - Ripple Dashboard', 
      trends: 'Trends - Ripple Dashboard',
      errors: 'Errors - Ripple Dashboard'
    }
    document.title = titles[activeTab as keyof typeof titles] || 'Ripple Dashboard'
  }, [activeTab])

  return (
    <Tabs value={activeTab} onValueChange={handleTabChange}>
      <TabsList className="grid w-full grid-cols-4">
        <TabsTrigger value="overview" className="flex items-center space-x-2">
          <Activity className="h-4 w-4" />
          <span>Overview</span>
        </TabsTrigger>
        <TabsTrigger value="platforms" className="flex items-center space-x-2">
          <BarChart3 className="h-4 w-4" />
          <span>Platforms</span>
        </TabsTrigger>
        <TabsTrigger value="trends" className="flex items-center space-x-2">
          <TrendingUp className="h-4 w-4" />
          <span>Trends</span>
        </TabsTrigger>
        <TabsTrigger value="errors" className="flex items-center space-x-2">
          <AlertTriangle className="h-4 w-4" />
          <span>Errors</span>
        </TabsTrigger>
      </TabsList>

      <div className="mt-6">
        <TabsContent value="overview" className="space-y-6">
          <DashboardSummary />
        </TabsContent>

        <TabsContent value="platforms" className="space-y-6">
          <PlatformStats />
        </TabsContent>

        <TabsContent value="trends" className="space-y-6">
          <SystemTrends />
        </TabsContent>

        <TabsContent value="errors" className="space-y-6">
          <ErrorLogs />
        </TabsContent>
      </div>
    </Tabs>
  )
}