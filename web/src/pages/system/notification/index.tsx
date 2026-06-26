import { useEffect, useMemo, useState } from "react";
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
} from "antd";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { api } from "../../../services/api";

const DEFAULT_ORG_ID = "00000000-0000-0000-0000-000000000001";
const channelOptions = [
  { label: "邮件", value: "email" },
  { label: "钉钉", value: "dingtalk" },
  { label: "企业微信", value: "wechat" },
  { label: "飞书", value: "feishu" },
  { label: "Webhook", value: "webhook" },
];

interface Channel {
  id: string;
  org_id: string;
  name: string;
  channel_type: string;
  config: string;
  is_enabled: boolean;
  created_at: string;
}

interface Template {
  id: string;
  name: string;
  channel_type: string;
  title_template: string;
  body_template: string;
  created_at: string;
}

interface NotifyLog {
  id: string;
  event_type: string;
  channel_id: string;
  recipient: string;
  title: string;
  status: string;
  error_msg: string;
  created_at: string;
}

function parseConfig(value?: string) {
  if (!value) return {};
  try {
    return JSON.parse(value);
  } catch {
    return {};
  }
}

function formatTime(value?: string) {
  return value ? new Date(value).toLocaleString("zh-CN") : "-";
}

export default function NotificationPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [logs, setLogs] = useState<NotifyLog[]>([]);
  const [logTotal, setLogTotal] = useState(0);
  const [loading, setLoading] = useState({ channels: false, templates: false, logs: false });
  const [channelModalOpen, setChannelModalOpen] = useState(false);
  const [templateModalOpen, setTemplateModalOpen] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [editingTemplate, setEditingTemplate] = useState<Template | null>(null);
  const [logPage, setLogPage] = useState(1);
  const [logStatus, setLogStatus] = useState<string | undefined>();
  const [channelForm] = Form.useForm();
  const [templateForm] = Form.useForm();

  const fetchChannels = async () => {
    setLoading((s) => ({ ...s, channels: true }));
    try {
      const res = await api.get("/notifications");
      setChannels(res.data.data || []);
    } finally {
      setLoading((s) => ({ ...s, channels: false }));
    }
  };

  const fetchTemplates = async () => {
    setLoading((s) => ({ ...s, templates: true }));
    try {
      const res = await api.get("/notify/templates");
      setTemplates(res.data.data || []);
    } finally {
      setLoading((s) => ({ ...s, templates: false }));
    }
  };

  const fetchLogs = async (page = logPage, status = logStatus) => {
    setLoading((s) => ({ ...s, logs: true }));
    try {
      const res = await api.get("/notify/logs", {
        params: { page, page_size: 20, status },
      });
      setLogs(res.data.data || []);
      setLogTotal(res.data.total || 0);
    } finally {
      setLoading((s) => ({ ...s, logs: false }));
    }
  };

  useEffect(() => {
    fetchChannels();
    fetchTemplates();
    fetchLogs(1);
  }, []);

  const openCreateChannel = () => {
    setEditingChannel(null);
    channelForm.setFieldsValue({ channel_type: "webhook", is_enabled: true, config: "{}" });
    setChannelModalOpen(true);
  };

  const openEditChannel = (channel: Channel) => {
    setEditingChannel(channel);
    channelForm.setFieldsValue({
      name: channel.name,
      channel_type: channel.channel_type,
      is_enabled: channel.is_enabled,
      config: JSON.stringify(parseConfig(channel.config), null, 2),
    });
    setChannelModalOpen(true);
  };

  const saveChannel = async () => {
    const values = await channelForm.validateFields();
    let config: Record<string, unknown>;
    try {
      config = JSON.parse(values.config || "{}");
    } catch {
      message.error("配置 JSON 格式不正确");
      return;
    }
    const payload = {
      org_id: DEFAULT_ORG_ID,
      name: values.name,
      channel_type: values.channel_type,
      is_enabled: values.is_enabled,
      config,
    };
    if (editingChannel) {
      await api.put(`/notifications/${editingChannel.id}`, payload);
      message.success("通知渠道已更新");
    } else {
      await api.post("/notifications", payload);
      message.success("通知渠道已创建");
    }
    setChannelModalOpen(false);
    fetchChannels();
  };

  const deleteChannel = async (id: string) => {
    await api.delete(`/notifications/${id}`);
    message.success("通知渠道已删除");
    fetchChannels();
  };

  const toggleChannel = async (channel: Channel, enabled: boolean) => {
    await api.put(`/notifications/${channel.id}`, {
      org_id: channel.org_id || DEFAULT_ORG_ID,
      name: channel.name,
      channel_type: channel.channel_type,
      is_enabled: enabled,
      config: parseConfig(channel.config),
    });
    message.success(enabled ? "通知渠道已启用" : "通知渠道已禁用");
    fetchChannels();
  };

  const openCreateTemplate = () => {
    setEditingTemplate(null);
    templateForm.setFieldsValue({ channel_type: "webhook", title_template: "{{.EventType}}", body_template: "{{.Detail}}" });
    setTemplateModalOpen(true);
  };

  const openEditTemplate = (tmpl: Template) => {
    setEditingTemplate(tmpl);
    templateForm.setFieldsValue(tmpl);
    setTemplateModalOpen(true);
  };

  const saveTemplate = async () => {
    const values = await templateForm.validateFields();
    if (editingTemplate) {
      await api.put(`/notify/templates/${editingTemplate.id}`, values);
      message.success("通知模板已更新");
    } else {
      await api.post("/notify/templates", values);
      message.success("通知模板已创建");
    }
    setTemplateModalOpen(false);
    fetchTemplates();
  };

  const deleteTemplate = async (id: string) => {
    await api.delete(`/notify/templates/${id}`);
    message.success("通知模板已删除");
    fetchTemplates();
  };

  const channelCols: ColumnsType<Channel> = [
    { title: "名称", dataIndex: "name", key: "name" },
    { title: "类型", dataIndex: "channel_type", key: "channel_type", width: 120, render: (t) => <Tag>{t}</Tag> },
    {
      title: "状态",
      dataIndex: "is_enabled",
      key: "is_enabled",
      width: 110,
      render: (_, record) => <Switch checked={record.is_enabled} onChange={(checked) => toggleChannel(record, checked)} />,
    },
    { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "action",
      width: 140,
      render: (_, record) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openEditChannel(record)} />
          <Popconfirm title="确认删除该通知渠道？" onConfirm={() => deleteChannel(record.id)}>
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const templateCols: ColumnsType<Template> = [
    { title: "名称", dataIndex: "name", key: "name" },
    { title: "渠道类型", dataIndex: "channel_type", key: "channel_type", width: 120, render: (t) => <Tag>{t}</Tag> },
    { title: "标题模板", dataIndex: "title_template", key: "title_template", ellipsis: true },
    { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "action",
      width: 140,
      render: (_, record) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openEditTemplate(record)} />
          <Popconfirm title="确认删除该通知模板？" onConfirm={() => deleteTemplate(record.id)}>
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const logCols: ColumnsType<NotifyLog> = [
    { title: "事件类型", dataIndex: "event_type", key: "event_type", width: 160, render: (t) => <Tag color="blue">{t}</Tag> },
    { title: "接收方", dataIndex: "recipient", key: "recipient", width: 180, ellipsis: true },
    { title: "标题", dataIndex: "title", key: "title", ellipsis: true },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 110,
      render: (s) =>
        s === "success" ? (
          <Tag color="green" icon={<CheckCircleOutlined />}>成功</Tag>
        ) : (
          <Tag color="red" icon={<CloseCircleOutlined />}>失败</Tag>
        ),
    },
    { title: "错误信息", dataIndex: "error_msg", key: "error_msg", ellipsis: true },
    { title: "时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatTime },
  ];

  const tabItems = useMemo(
    () => [
      {
        key: "channels",
        label: "通知渠道",
        children: (
          <>
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
              <Space>
                <Button icon={<ReloadOutlined />} onClick={fetchChannels}>刷新</Button>
              </Space>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreateChannel}>新建渠道</Button>
            </div>
            <Table dataSource={channels} columns={channelCols} rowKey="id" loading={loading.channels} pagination={false} />
          </>
        ),
      },
      {
        key: "templates",
        label: "通知模板",
        children: (
          <>
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
              <Button icon={<ReloadOutlined />} onClick={fetchTemplates}>刷新</Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreateTemplate}>新建模板</Button>
            </div>
            <Table dataSource={templates} columns={templateCols} rowKey="id" loading={loading.templates} pagination={false} />
          </>
        ),
      },
      {
        key: "logs",
        label: "发送记录",
        children: (
          <>
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
              <Select
                allowClear
                style={{ width: 160 }}
                placeholder="发送状态"
                value={logStatus}
                options={[{ label: "成功", value: "success" }, { label: "失败", value: "failed" }]}
                onChange={(value) => { setLogStatus(value); setLogPage(1); fetchLogs(1, value); }}
              />
              <Button icon={<ReloadOutlined />} onClick={() => fetchLogs()}>刷新</Button>
            </div>
            <Table
              dataSource={logs}
              columns={logCols}
              rowKey="id"
              loading={loading.logs}
              pagination={{ current: logPage, pageSize: 20, total: logTotal, showSizeChanger: false }}
              onChange={(pagination: TablePaginationConfig) => {
                const nextPage = pagination.current || 1;
                setLogPage(nextPage);
                fetchLogs(nextPage);
              }}
            />
          </>
        ),
      },
    ],
    [channels, templates, logs, loading, logPage, logTotal, logStatus],
  );

  return (
    <div>
      <Tabs items={tabItems} />

      <Modal
        title={editingChannel ? "编辑通知渠道" : "新建通知渠道"}
        open={channelModalOpen}
        onCancel={() => setChannelModalOpen(false)}
        onOk={saveChannel}
        width={680}
      >
        <Form form={channelForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入渠道名称" }]}>
            <Input placeholder="例如：默认告警 Webhook" />
          </Form.Item>
          <Form.Item name="channel_type" label="类型" rules={[{ required: true, message: "请选择渠道类型" }]}>
            <Select options={channelOptions} />
          </Form.Item>
          <Form.Item name="is_enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="config" label="配置 JSON" rules={[{ required: true, message: "请输入配置 JSON" }]}>
            <Input.TextArea rows={8} placeholder={'{"url":"https://example.com/webhook"}'} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingTemplate ? "编辑通知模板" : "新建通知模板"}
        open={templateModalOpen}
        onCancel={() => setTemplateModalOpen(false)}
        onOk={saveTemplate}
        width={680}
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入模板名称" }]}>
            <Input placeholder="例如：登录审计通知" />
          </Form.Item>
          <Form.Item name="channel_type" label="渠道类型" rules={[{ required: true, message: "请选择渠道类型" }]}>
            <Select options={channelOptions} />
          </Form.Item>
          <Form.Item name="title_template" label="标题模板">
            <Input placeholder="{{.EventType}}" />
          </Form.Item>
          <Form.Item name="body_template" label="正文模板" rules={[{ required: true, message: "请输入正文模板" }]}>
            <Input.TextArea rows={6} placeholder="{{.Detail}}" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}