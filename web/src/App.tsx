import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { NavigationTabs } from '@/components/NavigationTabs'
import { Waves } from 'lucide-react'

function DashboardContent() {

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-card">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center space-x-2">
            <Waves className="h-8 w-8 text-primary" />
            <div>
              <h1 className="text-2xl font-bold">Ripple Dashboard</h1>
              <p className="text-sm text-muted-foreground">
                Content distribution monitoring and analytics
              </p>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-6">
        <NavigationTabs />
      </main>

      {/* Footer */}
      <footer className="border-t bg-card mt-12">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <p>
              Ripple Dashboard - Monitor your content distribution pipeline
            </p>
            <p>
              Built with CC
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Navigate to="/overview" replace />} />
        <Route path="/overview" element={<DashboardContent />} />
        <Route path="/platforms" element={<DashboardContent />} />
        <Route path="/trends" element={<DashboardContent />} />
        <Route path="/errors" element={<DashboardContent />} />
        <Route path="*" element={<Navigate to="/overview" replace />} />
      </Routes>
    </Router>
  )
}

export default App