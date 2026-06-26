import { useState, useEffect } from "react";
import { Table, Button, Space, Modal, Form, Input, Tag, message, Tree } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { api } from "../../../services/api";

interface Role {
  id: string;
  name: string;
  code: string;
  description: string;
  is_system: boolean;
  created_at: string;
}

export default function RoleList() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [permModalOpen, setPermModalOpen] = useState(false);
  const [currentRole, setCurrentRole] = useState<Role | null>(null);
  const [permTree, setPermTree] = useState<any[]>([]);
  const [checkedKeys, setCheckedKeys] = useState<string[]>([]);
  const [form] = Form.useForm();

  const fetchRoles = async () => {
    setLoading(true);
    try {
      const res = await api.get("/roles");
      setRoles(res.data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  const fetchPermTree = async () => {
    try {
      const res = await api.get("/permissions");
      setPermTree(res.data.data || []);
    } catch (e) { console.error(e); }
  };

  useEffect(() => { fetchRoles(); }, []);

  const handleCreate = async (values: any) => {
    try {
      await api.post("/roles", values);
      message.success("角色创建成功");
      setModalOpen(false);
      form.resetFields();
      fetchRoles();
    } catch (e: any) { message.error(e.response?.data?.message || "创建失败"); }
  };

  const handleAssignPerm = async () => {
    if (!currentRole) return;
    try {
      await api.put(`/roles/${currentRole.id}/permissions`, { permission_ids: checkedKeys });
      message.success("权限分配成功");
      setPermModalOpen(false);
    } catch (e: any) { message.error(e.response?.data?.message || "分配失败"); }
  };

  const columns = [
    { title: "角色名称", dataIndex: "name", key: "name" },
    { title: "角色编码", dataIndex: "code", key: "code" },
    { title: "描述", dataIndex: "description", key: "description" },
    {
      title: "系统内置",
      dataIndex: "is_system",
      key: "is_system",
      render: (v: boolean) => (v ? <Tag color="blue">系统</Tag> : <Tag>自定义</Tag>),
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: Role) => (
        <Space>
          <Button
            type="link"
            onClick={() => {
              setCurrentRole(record);
              fetchPermTree();
              setPermModalOpen(true);
            }}
          >
            分配权限
          </Button>
          <Button type="link" danger disabled={record.is_system}>删除</Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
        <h2>角色管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          新建角色
        </Button>
      </div>
      <Table dataSource={roles} columns={columns} rowKey="id" loading={loading} />
      <Modal title="新建角色" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()}>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="角色名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="code" label="角色编码" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea /></Form.Item>
        </Form>
      </Modal>
      <Modal title="分配权限" open={permModalOpen} onCancel={() => setPermModalOpen(false)} onOk={handleAssignPerm}>
        <Tree
          checkable
          treeData={permTree}
          checkedKeys={checkedKeys}
          onCheck={(keys) => setCheckedKeys(keys as string[])}
          fieldNames={{ title: "name", key: "id" }}
        />
      </Modal>
    </div>
  );
}
