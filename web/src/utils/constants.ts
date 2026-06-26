export const TOKEN_KEY = "token";
export const PAGE_SIZE = 20;

export const PERMISSION_CODES = {
  USER_CREATE: "user:create",
  USER_READ: "user:read",
  USER_UPDATE: "user:update",
  USER_DELETE: "user:delete",
  ROLE_CREATE: "role:create",
  ROLE_READ: "role:read",
  ROLE_UPDATE: "role:update",
  ROLE_DELETE: "role:delete",
  ROLE_ASSIGN: "role:assign",
} as const;

export const EVENT_TYPES = [
  { label: "用户登录", value: "user.login" },
  { label: "登录失败", value: "user.login_failed" },
  { label: "用户登出", value: "user.logout" },
  { label: "创建用户", value: "user.created" },
  { label: "更新用户", value: "user.updated" },
  { label: "删除用户", value: "user.deleted" },
  { label: "用户角色变更", value: "user.role_changed" },
  { label: "创建角色", value: "role.created" },
  { label: "更新角色", value: "role.updated" },
  { label: "删除角色", value: "role.deleted" },
  { label: "角色权限变更", value: "role.permission_changed" },
  { label: "创建组织", value: "org.created" },
  { label: "更新组织", value: "org.updated" },
] as const;
