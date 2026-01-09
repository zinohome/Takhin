import { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import { Layout, Menu, theme, Breadcrumb, Dropdown, Space } from 'antd'
import {
  DashboardOutlined,
  DatabaseOutlined,
  ClusterOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  ApiOutlined,
  TeamOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

const { Header, Sider, Content, Footer } = Layout

type MenuItem = Required<MenuProps>['items'][number]

const menuItems: MenuItem[] = [
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
    key: '/brokers',
    icon: <ClusterOutlined />,
    label: <Link to="/brokers">Brokers</Link>,
  },
  {
    key: 'consumers',
    icon: <TeamOutlined />,
    label: 'Consumer Groups',
    children: [
      {
        key: '/consumers',
        label: <Link to="/consumers">All Groups</Link>,
      },
    ],
  },
  {
    key: '/configuration',
    icon: <SettingOutlined />,
    label: <Link to="/configuration">Configuration</Link>,
  },
]

const userMenuItems: MenuProps['items'] = [
  {
    key: 'settings',
    icon: <SettingOutlined />,
    label: 'Settings',
  },
  {
    key: 'api',
    icon: <ApiOutlined />,
    label: 'API Keys',
  },
  {
    type: 'divider',
  },
  {
    key: 'logout',
    icon: <LogoutOutlined />,
    label: 'Logout',
    danger: true,
  },
]

const getBreadcrumbItems = (pathname: string) => {
  const paths = pathname.split('/').filter(p => p)
  const breadcrumbs = [
    {
      title: <Link to="/">Home</Link>,
    },
  ]

  paths.forEach((path, index) => {
    const url = `/${paths.slice(0, index + 1).join('/')}`
    breadcrumbs.push({
      title: <Link to={url}>{path.charAt(0).toUpperCase() + path.slice(1)}</Link>,
    })
  })

  return breadcrumbs
}

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  const selectedKeys = [location.pathname]
  const openKeys = menuItems
    .filter(item => item && 'children' in item && item.children)
    .filter(item => {
      const children = (item as { children: MenuItem[] }).children
      return children?.some(
        child => child && 'key' in child && selectedKeys.includes(child.key as string)
      )
    })
    .map(item => item!.key as string)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        breakpoint="lg"
        onBreakpoint={broken => {
          if (broken && !collapsed) {
            setCollapsed(true)
          }
        }}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: collapsed ? 20 : 24,
            fontWeight: 'bold',
            transition: 'all 0.2s',
          }}
        >
          {collapsed ? 'T' : 'Takhin'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={selectedKeys}
          defaultOpenKeys={openKeys}
          items={menuItems}
        />
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 200, transition: 'all 0.2s' }}>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            boxShadow: '0 1px 4px rgba(0,21,41,.08)',
            position: 'sticky',
            top: 0,
            zIndex: 1,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            {collapsed ? (
              <MenuUnfoldOutlined
                style={{ fontSize: 18, cursor: 'pointer' }}
                onClick={() => setCollapsed(!collapsed)}
              />
            ) : (
              <MenuFoldOutlined
                style={{ fontSize: 18, cursor: 'pointer' }}
                onClick={() => setCollapsed(!collapsed)}
              />
            )}
            <Breadcrumb items={getBreadcrumbItems(location.pathname)} />
          </div>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Space style={{ cursor: 'pointer' }}>
              <UserOutlined style={{ fontSize: 16 }} />
              <span>Admin</span>
            </Space>
          </Dropdown>
        </Header>
        <Content
          style={{
            margin: '24px 16px 0',
            overflow: 'initial',
          }}
        >
          <div
            style={{
              padding: 24,
              minHeight: 360,
              background: colorBgContainer,
              borderRadius: borderRadiusLG,
            }}
          >
            <Outlet />
          </div>
        </Content>
        <Footer style={{ textAlign: 'center', padding: '12px 16px' }}>
          Takhin Console ©{new Date().getFullYear()} - Kafka-compatible streaming platform
        </Footer>
      </Layout>
    </Layout>
  )
}
