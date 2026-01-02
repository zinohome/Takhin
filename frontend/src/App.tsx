import { Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from './layouts/MainLayout'
import Dashboard from './pages/Dashboard'
import Topics from './pages/Topics'
import Messages from './pages/Messages'
import Brokers from './pages/Brokers'
import Consumers from './pages/Consumers'
import Configuration from './pages/Configuration'
import './App.css'

function App() {
  return (
    <div className="app-container">
      <Routes>
        <Route path="/" element={<MainLayout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="topics" element={<Topics />} />
          <Route path="topics/:topicName" element={<Topics />} />
          <Route path="topics/:topicName/messages" element={<Messages />} />
          <Route path="brokers" element={<Brokers />} />
          <Route path="brokers/:brokerId" element={<Brokers />} />
          <Route path="consumers" element={<Consumers />} />
          <Route path="consumers/:groupId" element={<Consumers />} />
          <Route path="configuration" element={<Configuration />} />
        </Route>
      </Routes>
    </div>
  )
}

export default App
