import { useState, useEffect } from "react";
import { Table, Button, Space, Modal, Form, Input, Select, Tag, message, Popconfirm, Checkbox } from "antd";
import { KeyOutlined, PlusOutlined, ReloadOutlined, UserSwitchOutlined } from "@ant-design/icons";
import { api } from "../../../services/api";

interface User {
  id: string; username: string; display_name: string; email: string;
  phone: string; status: string; source: string; created_at: string;
}

interface Role {
  id: string; name: string; code: string;
}

const passwordRules = [
  {
    pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^\w\s])\S{8,}$/,
    message: "密码至少 8 位，需包含大小写字母、数字和符号",
  },
];

export default function UserList() {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const [form] = Form.useForm();
  const [passwordForm] = Form.useForm();

  const fetchUsers = async (p = page) => {
    setLoading(true);
    try {
      const res = await api.get("/users", { params: { page: p, page_size: 20 } });
      setUsers(res.data.data || []);
      setTotal(res.data.total || 0);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };
  const fetchRoles = async () => {
    try { const res = await api.get("/roles"); setRoles(res.data.data || []); }
    catch (e) { console.error(e); }
  };

  useEffect(() => { fetchUsers(); fetchRoles(); }, []);

  const handleCreate = async (values: any) => {
    try {
      await api.post("/users", { ...values, org_id: "00000000-0000-0000-0000-000000000001" });
      message.success("用户创建成功");
      setModalOpen(false); form.resetFields(); fetchUsers();
    } catch (e: any) { message.error(e.response?.data?.message || "创建失败"); }
  };

  const handleToggleStatus = async (id: string, currentStatus: string) => {
    const newStatus = currentStatus === "active" ? "disabled" : "active";
    try {
      await api.put(`/users/${id}/status`, { status: newStatus });
      message.success(newStatus === "active" ? "用户已启用" : "用户已禁用");
      fetchUsers();
    } catch (e: any) { message.error("操作失败"); }
  };

  const handleAssignRoles = async () => {
    if (!currentUser) return;
    try {
      await api.put(`/users/${currentUser.id}/roles`, { role_ids: selectedRoles });
      message.success("角色分配成功");
      setRoleModalOpen(false); fetchUsers();
    } catch (e: any) { message.error(e.response?.data?.message || "分配失败"); }
  };

  const openResetPasswordModal = (user: User) => {
    setCurrentUser(user);
    passwordForm.resetFields();
    passwordForm.setFieldsValue({ must_change_password: true });
    setResetPasswordOpen(true);
  };

  const handleResetPassword = async (values: { new_password: string; must_change_password?: boolean }) => {
    if (!currentUser) return;
    try {
      await api.put(`/users/${currentUser.id}/password/reset`, {
        new_password: values.new_password,
        must_change_password: values.must_change_password ?? true,
      });
      message.success("密码已重置");
      setResetPasswordOpen(false);
      passwordForm.resetFields();
      fetchUsers();
    } catch (e: any) {
      message.error(e.response?.data?.message || "重置失败");
    }
  };

  const openRoleModal = async (user: User) => {
    setCurrentUser(user);
    try {
      const res = await api.get(`/users/${user.id}/roles`);
      setSelectedRoles(res.data.data?.map((r: Role) => r.id) || []);
    } catch { setSelectedRoles([]); }
    setRoleModalOpen(true);
  };

  const columns = [
    { title: "用户名", dataIndex: "username", key: "username", width: 120 },
    { title: "显示名", dataIndex: "display_name", key: "display_name", width: 120 },
    { title: "邮箱", dataIndex: "email", key: "email", width: 200, ellipsis: true },
    { title: "手机号", dataIndex: "phone", key: "phone", width: 130 },
    {
      title: "状态", dataIndex: "status", key: "status", width: 80,
      render: (s: string) => {
        const colors: Record<string, string> = { active: "green", disabled: "red", locked: "orange" };
        const labels: Record<string, string> = { active: "启用", disabled: "禁用", locked: "锁定" };
        return <Tag color={colors[s] || "default"}>{labels[s] || s}</Tag>;
      },
    },
    {
      title: "来源", dataIndex: "source", key: "source", width: 80,
      render: (s: string) => s === "ldap" ? "LDAP" : s === "oauth" ? "OAuth" : "本地",
    },
    {
      title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180,
      render: (t: string) => t ? new Date(t).toLocaleString("zh-CN") : "-",
    },
    {
      title: "操作", key: "action", width: 300,
      render: (_: any, record: User) => (
        <Space>
          <Button type="link" size="small" onClick={() => openRoleModal(record)} icon={<UserSwitchOutlined />}>角色</Button>
          <Button type="link" size="small" onClick={() => openResetPasswordModal(record)} icon={<KeyOutlined />}>重置密码</Button>
          <Popconfirm
            title={record.status === "active" ? "确定禁用此用户？" : "确定启用此用户？"}
            onConfirm={() => handleToggleStatus(record.id, record.status)}
          >
            <Button type="link" size="small" danger={record.status === "active"}>
              {record.status === "active" ? "禁用" : "启用"}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>用户管理</h2>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => fetchUsers()}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建用户</Button>
        </Space>
      </div>
      <Table
        dataSource={users}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{ current: page, total, pageSize: 20, showSizeChanger: true, onChange: (p) => { setPage(p); fetchUsers(p); } }}
        locale={{ emptyText: "暂无用户数据" }}
        scroll={{ x: 1000 }}
      />
      <Modal title="新建用户" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()} width={500}>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: "请输入用户名" }]}><Input placeholder="登录账号" /></Form.Item>
          <Form.Item name="display_name" label="显示名"><Input placeholder="显示名称" /></Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ type: "email", message: "请输入有效邮箱" }]}><Input placeholder="user@example.com" /></Form.Item>
          <Form.Item name="phone" label="手机号"><Input placeholder="手机号码" /></Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: "请输入密码" }, ...passwordRules]}><Input.Password placeholder="设置密码" /></Form.Item>
        </Form>
      </Modal>
      <Modal
        title={currentUser ? `重置密码 - ${currentUser.display_name || currentUser.username}` : "重置密码"}
        open={resetPasswordOpen}
        onCancel={() => setResetPasswordOpen(false)}
        onOk={() => passwordForm.submit()}
        width={480}
      >
        <Form form={passwordForm} layout="vertical" onFinish={handleResetPassword}>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: "请输入新密码" }, ...passwordRules]}>
            <Input.Password placeholder="设置新密码" />
          </Form.Item>
          <Form.Item name="must_change_password" valuePropName="checked">
            <Checkbox>下次登录强制修改密码</Checkbox>
          </Form.Item>
        </Form>
      </Modal>
      <Modal title={currentUser ? `分配角色 - ${currentUser.display_name || currentUser.username}` : "分配角色"} open={roleModalOpen} onCancel={() => setRoleModalOpen(false)} onOk={handleAssignRoles} width={450}>
        <Select
          mode="multiple"
          style={{ width: "100%" }}
          placeholder="选择角色"
          value={selectedRoles}
          onChange={setSelectedRoles}
          options={roles.map((r) => ({ label: `${r.name} (${r.code})`, value: r.id }))}
        />
        {roles.length === 0 && <div style={{ color: "#999", marginTop: 8 }}>暂无角色数据，请先创建角色</div>}
      </Modal>
    </div>
  );
}
