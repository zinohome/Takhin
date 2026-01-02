import { Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from './layouts/MainLayout'
import Dashboard from './pages/Dashboard'
import Topics from './pages/Topics'
import Brokers from './pages/Brokers'
import ConsumerGroups from './pages/ConsumerGroups'
import ConsumerGroupDetail from './pages/ConsumerGroupDetail'

function App() {
  return (
    <Routes>
      <Route path="/" element={<MainLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="topics" element={<Topics />} />
        <Route path="brokers" element={<Brokers />} />
        <Route path="consumer-groups" element={<ConsumerGroups />} />
        <Route path="consumer-groups/:groupId" element={<ConsumerGroupDetail />} />
      </Route>
    </Routes>
  )
}

export default App
