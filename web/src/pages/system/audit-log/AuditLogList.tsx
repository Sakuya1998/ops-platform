import { useState, useEffect } from "react";
import { Table, Select, Space, Tag, DatePicker, Button } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { api } from "../../../services/api";
import { EVENT_TYPES } from "../../../utils/constants";

const { RangePicker } = DatePicker;

interface AuditLog {
  id: string;
  event_type: string;
  username: string;
  action: string;
  resource_type: string;
  detail: string;
  ip: string;
  created_at: string;
}

type DateRange = [dayjs.Dayjs | null, dayjs.Dayjs | null] | null;

export default function AuditLogList() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [eventType, setEventType] = useState<string | undefined>();
  const [dates, setDates] = useState<DateRange>(null);

  const fetchLogs = async (p = page) => {
    setLoading(true);
    try {
      const params: Record<string, string | number> = { page: p, page_size: 20 };
      if (eventType) params.event_type = eventType;
      if (dates?.[0]) params.start_time = dates[0].startOf("day").toISOString();
      if (dates?.[1]) params.end_time = dates[1].endOf("day").toISOString();
      const res = await api.get("/audit-logs", { params });
      setLogs(res.data.data || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setPage(1);
    fetchLogs(1);
  }, [eventType, dates]);

  const columns = [
    {
      title: "事件类型",
      dataIndex: "event_type",
      key: "event_type",
      width: 160,
      render: (t: string) => {
        const found = EVENT_TYPES.find((e) => e.value === t);
        return <Tag color="blue">{found?.label || t}</Tag>;
      },
    },
    { title: "用户名", dataIndex: "username", key: "username", width: 120 },
    { title: "操作", dataIndex: "action", key: "action", width: 120 },
    { title: "资源类型", dataIndex: "resource_type", key: "resource_type", width: 120 },
    { title: "详情", dataIndex: "detail", key: "detail", ellipsis: true },
    { title: "IP", dataIndex: "ip", key: "ip", width: 140 },
    {
      title: "时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (t: string) => (t ? new Date(t).toLocaleString("zh-CN") : "-"),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>审计日志</h2>
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          placeholder="筛选事件类型"
          allowClear
          style={{ width: 220 }}
          value={eventType}
          onChange={(value) => setEventType(value)}
          options={EVENT_TYPES.map((e) => ({ label: e.label, value: e.value }))}
        />
        <RangePicker value={dates} onChange={(value) => setDates(value as DateRange)} />
        <Button icon={<ReloadOutlined />} onClick={() => fetchLogs()}>刷新</Button>
        <span style={{ color: "#999" }}>共 {total} 条记录</span>
      </Space>
      <Table
        dataSource={logs}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          total,
          pageSize: 20,
          showSizeChanger: false,
          onChange: (nextPage) => {
            setPage(nextPage);
            fetchLogs(nextPage);
          },
        }}
        locale={{ emptyText: "暂无审计日志" }}
        scroll={{ x: 980 }}
      />
    </div>
  );
}