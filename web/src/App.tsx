import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { NavigationTabs } from '@/components/NavigationTabs'
import { LoginPage } from '@/components/LoginPage'
import { Waves } from 'lucide-react'
import { useState, useEffect } from 'react'

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

function AuthWrapper() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const location = useLocation();

  useEffect(() => {
    // Check if user is authenticated by looking for auth cookie
    const checkAuth = () => {
      const authToken = document.cookie
        .split('; ')
        .find(row => row.startsWith('auth_token='));
      
      setIsAuthenticated(!!authToken);
      setIsLoading(false);
    };

    checkAuth();
  }, []);

  const handleLogin = () => {
    setIsAuthenticated(true);
    // Redirect to original destination or dashboard
    const redirectTo = location.pathname === '/login' ? '/overview' : location.pathname;
    window.location.href = redirectTo;
  };

  const handleLogout = async () => {
    try {
      await fetch('/api/v1/auth/logout', { method: 'POST' });
    } catch (error) {
      console.error('Logout error:', error);
    }
    setIsAuthenticated(false);
    window.location.href = '/login';
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <Waves className="h-8 w-8 text-primary mx-auto mb-4 animate-pulse" />
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated && location.pathname !== '/login') {
    return <LoginPage onLogin={handleLogin} />;
  }

  if (isAuthenticated && location.pathname === '/login') {
    return <Navigate to="/overview" replace />;
  }

  if (location.pathname === '/login') {
    return <LoginPage onLogin={handleLogin} />;
  }

  return <DashboardContent />;
}

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Navigate to="/overview" replace />} />
        <Route path="/login" element={<AuthWrapper />} />
        <Route path="/overview" element={<AuthWrapper />} />
        <Route path="/platforms" element={<AuthWrapper />} />
        <Route path="/trends" element={<AuthWrapper />} />
        <Route path="/errors" element={<AuthWrapper />} />
        <Route path="*" element={<Navigate to="/overview" replace />} />
      </Routes>
    </Router>
  )
}

export default App