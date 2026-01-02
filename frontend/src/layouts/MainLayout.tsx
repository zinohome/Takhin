import { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import { Layout, Menu, theme } from 'antd'
import {
  DashboardOutlined,
  DatabaseOutlined,
  ClusterOutlined,
  TeamOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

const { Header, Sider, Content } = Layout

type MenuItem = Required<MenuProps>['items'][number]

const items: MenuItem[] = [
  {
    key: '/dashboard',
    icon: <DashboardOutlined />,
    label: <Link to="/dashboard">Dashboard</Link>,
  },
  {
    key: '/topics',
    icon: <DatabaseOutlined />,
    label: <Link to="/topics">Topics</Link>,
  },
  {
    key: '/consumer-groups',
    icon: <TeamOutlined />,
    label: <Link to="/consumer-groups">Consumer Groups</Link>,
  },
  {
    key: '/brokers',
    icon: <ClusterOutlined />,
    label: <Link to="/brokers">Brokers</Link>,
  },
]

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  // Determine selected key based on current path
  const getSelectedKey = () => {
    if (location.pathname.startsWith('/consumer-groups')) {
      return '/consumer-groups'
    }
    return location.pathname
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: collapsed ? 18 : 24,
            fontWeight: 'bold',
          }}
        >
          {collapsed ? 'T' : 'Takhin'}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[getSelectedKey()]} items={items} />
      </Sider>
      <Layout>
        <Header style={{ padding: 0, background: colorBgContainer }}>
          {collapsed ? (
            <MenuUnfoldOutlined
              style={{ fontSize: 18, marginLeft: 16, cursor: 'pointer' }}
              onClick={() => setCollapsed(!collapsed)}
            />
          ) : (
            <MenuFoldOutlined
              style={{ fontSize: 18, marginLeft: 16, cursor: 'pointer' }}
              onClick={() => setCollapsed(!collapsed)}
            />
          )}
        </Header>
        <Content
          style={{
            margin: '24px 16px',
            padding: 24,
            minHeight: 280,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
