import { useState, useEffect } from "react";
import { Table, Button, Space, Modal, Form, Input, message, Tag } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { api } from "../../../services/api";

interface Organization {
  id: string;
  name: string;
  code: string;
  description: string;
  status: string;
  created_at: string;
}

export default function OrganizationList() {
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchOrgs = async () => {
    setLoading(true);
    try {
      const res = await api.get("/organizations");
      setOrgs(res.data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchOrgs(); }, []);

  const handleCreate = async (values: any) => {
    try {
      await api.post("/organizations", values);
      message.success("组织创建成功");
      setModalOpen(false);
      form.resetFields();
      fetchOrgs();
    } catch (e: any) { message.error(e.response?.data?.message || "创建失败"); }
  };

  const columns = [
    { title: "组织名称", dataIndex: "name", key: "name" },
    { title: "组织编码", dataIndex: "code", key: "code" },
    { title: "描述", dataIndex: "description", key: "description" },
    { title: "状态", dataIndex: "status", key: "status",
      render: (s: string) => <Tag color={s === "active" ? "green" : "red"}>{s === "active" ? "启用" : "禁用"}</Tag>
    },
    { title: "创建时间", dataIndex: "created_at", key: "created_at",
      render: (t: string) => new Date(t).toLocaleString("zh-CN")
    },
  ];

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
        <h2>组织管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建组织</Button>
      </div>
      <Table dataSource={orgs} columns={columns} rowKey="id" loading={loading} />
      <Modal title="新建组织" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()}>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="组织名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="code" label="组织编码" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
