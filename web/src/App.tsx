import { lazy, Suspense } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import MainLayout from "./layouts/MainLayout";
import { useAuthStore } from "./store/authStore";

const LoginPage = lazy(() => import("./pages/login"));
const DashboardPage = lazy(() => import("./pages/dashboard"));
const UserList = lazy(() => import("./pages/system/user/UserList"));
const RoleList = lazy(() => import("./pages/system/role/RoleList"));
const PermissionTree = lazy(() => import("./pages/system/permission/PermissionTree"));
const OrganizationList = lazy(() => import("./pages/system/organization/OrganizationList"));
const AuditLogList = lazy(() => import("./pages/system/audit-log/AuditLogList"));
const NotificationPage = lazy(() => import("./pages/system/notification"));
const SystemConfig = lazy(() => import("./pages/system/config/SystemConfigPage"));
const ProfilePage = lazy(() => import("./pages/profile/ProfilePage"));

function PageLoader() {
  return <div style={{ padding: 24 }}>加载中...</div>;
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  const user = useAuthStore((s) => s.user);
  if (!token) return <Navigate to="/login" replace />;
  if (user?.must_change_password && window.location.pathname !== "/profile") {
    return <Navigate to="/profile" replace />;
  }
  return <>{children}</>;
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  if (token) return <Navigate to="/" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route
          path="/login"
          element={
            <PublicRoute>
              <LoginPage />
            </PublicRoute>
          }
        />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <MainLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<DashboardPage />} />
          <Route path="system/users" element={<UserList />} />
          <Route path="system/roles" element={<RoleList />} />
          <Route path="system/permissions" element={<PermissionTree />} />
          <Route path="system/organizations" element={<OrganizationList />} />
          <Route path="system/audit-logs" element={<AuditLogList />} />
          <Route path="system/notifications" element={<NotificationPage />} />
          <Route path="system/config" element={<SystemConfig />} />
          <Route path="profile" element={<ProfilePage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
