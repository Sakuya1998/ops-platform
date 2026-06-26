import { useEffect, useState } from "react";
import { Card, Col, Row, Statistic, Spin } from "antd";
import { TeamOutlined, SafetyOutlined, ApartmentOutlined, AuditOutlined } from "@ant-design/icons";
import { api } from "../../services/api";

interface DashboardStats {
  userCount: number; roleCount: number; orgCount: number; auditCount: number;
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats>({ userCount: 0, roleCount: 0, orgCount: 0, auditCount: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [usersRes, rolesRes, orgsRes, auditRes] = await Promise.all([
          api.get("/users", { params: { page: 1, page_size: 1 } }),
          api.get("/roles"),
          api.get("/organizations"),
          api.get("/audit-logs", { params: { page: 1, page_size: 1 } }),
        ]);
        setStats({
          userCount: usersRes.data.total || 0,
          roleCount: rolesRes.data.data?.length || 0,
          orgCount: orgsRes.data.data?.length || 0,
          auditCount: auditRes.data.total || 0,
        });
      } catch (e) { console.error("Failed to fetch dashboard stats:", e); }
      finally { setLoading(false); }
    };
    fetchStats();
  }, []);

  if (loading) return <div style={{ textAlign: "center", padding: 80 }}><Spin size="large" /></div>;

  return (
    <div>
      <h2 style={{ marginBottom: 24, fontSize: 20 }}>仪表盘</h2>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="用户总数" value={stats.userCount} prefix={<TeamOutlined />} valueStyle={{ color: "#1677ff" }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="角色总数" value={stats.roleCount} prefix={<SafetyOutlined />} valueStyle={{ color: "#52c41a" }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="组织总数" value={stats.orgCount} prefix={<ApartmentOutlined />} valueStyle={{ color: "#faad14" }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="审计事件" value={stats.auditCount} prefix={<AuditOutlined />} valueStyle={{ color: "#722ed1" }} />
          </Card>
        </Col>
      </Row>
    </div>
  );
}