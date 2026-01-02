import { Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from './layouts/MainLayout'
import Dashboard from './pages/Dashboard'
import Topics from './pages/Topics'
import Brokers from './pages/Brokers'

function App() {
  return (
    <Routes>
      <Route path="/" element={<MainLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="topics" element={<Topics />} />
        <Route path="brokers" element={<Brokers />} />
      </Route>
    </Routes>
  )
}

export default App
