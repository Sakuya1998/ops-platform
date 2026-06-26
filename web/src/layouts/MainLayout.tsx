import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { Layout, Menu, Button, Dropdown, Avatar, theme, Typography } from "antd";
import {
  ApartmentOutlined,
  AuditOutlined,
  BellOutlined,
  DashboardOutlined,
  KeyOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SafetyOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { useAuthStore } from "../store/authStore";
import { useAppStore } from "../store/appStore";

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const menuItems = [
  { key: "/", icon: <DashboardOutlined />, label: "仪表盘" },
  {
    key: "system",
    icon: <SettingOutlined />,
    label: "系统管理",
    children: [
      { key: "/system/users", icon: <TeamOutlined />, label: "用户管理" },
      { key: "/system/roles", icon: <SafetyOutlined />, label: "角色管理" },
      { key: "/system/permissions", icon: <KeyOutlined />, label: "权限管理" },
      { key: "/system/organizations", icon: <ApartmentOutlined />, label: "组织管理" },
      { key: "/system/audit-logs", icon: <AuditOutlined />, label: "审计日志" },
      { key: "/system/notifications", icon: <BellOutlined />, label: "通知管理" },
      { key: "/system/config", icon: <SettingOutlined />, label: "系统配置" },
    ],
  },
];

const pageTitles: Record<string, string> = {
  "/": "仪表盘",
  "/system/users": "用户管理",
  "/system/roles": "角色管理",
  "/system/permissions": "权限管理",
  "/system/organizations": "组织管理",
  "/system/audit-logs": "审计日志",
  "/system/notifications": "通知管理",
  "/system/config": "系统配置",
  "/profile": "个人中心",
};

export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const logout = useAuthStore((s) => s.logout);
  const user = useAuthStore((s) => s.user);
  const collapsed = useAppStore((s) => s.collapsed);
  const toggleCollapsed = useAppStore((s) => s.toggleCollapsed);
  const { token: themeToken } = theme.useToken();
  const pageTitle = pageTitles[location.pathname] || "运维平台";

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const userMenu = {
    items: [
      { key: "profile", icon: <UserOutlined />, label: "个人中心" },
      { type: "divider" as const },
      { key: "logout", icon: <LogoutOutlined />, label: "退出登录", danger: true },
    ],
    onClick: ({ key }: { key: string }) => {
      if (key === "profile") navigate("/profile");
      if (key === "logout") handleLogout();
    },
  };

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        theme="light"
        style={{ borderRight: `1px solid ${themeToken.colorBorderSecondary}` }}
      >
        <div
          style={{
            height: 64,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: collapsed ? 16 : 20,
            fontWeight: 700,
            color: themeToken.colorPrimary,
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
            letterSpacing: 0,
          }}
        >
          {collapsed ? "O" : "运维平台"}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          defaultOpenKeys={["system"]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderInlineEnd: "none" }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: "0 24px",
            background: themeToken.colorBgContainer,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
            height: 64,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={toggleCollapsed}
            />
            <Text strong style={{ fontSize: 16 }}>
              {pageTitle}
            </Text>
          </div>
          <Dropdown menu={userMenu} placement="bottomRight">
            <div
              style={{
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: 8,
                padding: "4px 8px",
                borderRadius: 6,
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = "#f5f5f5")}
              onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
            >
              <Avatar size="small" icon={<UserOutlined />} style={{ backgroundColor: themeToken.colorPrimary }} />
              <Text>{user?.display_name || user?.username || "用户"}</Text>
            </div>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}