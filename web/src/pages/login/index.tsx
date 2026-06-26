import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Form, Input, Button, Card, message, Tabs, Alert, Divider } from "antd";
import { UserOutlined, LockOutlined, GoogleOutlined, SafetyOutlined } from "@ant-design/icons";
import { useAuthStore } from "../../store/authStore";
import { api } from "../../services/api";

export default function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [oidcProvider, setOidcProvider] = useState("SSO");

  useEffect(() => {
    api.get("/auth/oidc/status").then((res) => {
      if (res.data.data?.enabled) {
        setOidcEnabled(true);
        setOidcProvider(res.data.data.provider_name || "SSO");
      }
    }).catch(() => {});
  }, []);

  const handleLogin = async (values: { username: string; password: string; mfa_code?: string }) => {
    setLoading(true); setError(null);
    try {
      await login(values.username, values.password, values.mfa_code);
      message.success("登录成功"); navigate("/");
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } } };
      setError(e?.response?.data?.message || "用户名或密码错误");
    } finally { setLoading(false); }
  };

  const handleOIDCLogin = () => {
    window.location.href = "/api/v1/auth/oidc/login";
  };

  return (
    <div style={{
      minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center",
      background: "linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)",
    }}>
      <Card style={{ width: 420, boxShadow: "0 8px 32px rgba(0,0,0,0.3)", borderRadius: 12, border: "none" }}>
        <div style={{ textAlign: "center", marginBottom: 24 }}>
          <div style={{ fontSize: 36, fontWeight: 700, color: "#1677ff", marginBottom: 4, letterSpacing: 4 }}>运维平台</div>
          <div style={{ color: "#999", fontSize: 14 }}>Operations Platform</div>
        </div>
        {error && <Alert message={error} type="error" showIcon style={{ marginBottom: 16 }} closable onClose={() => setError(null)} />}
        <Tabs centered items={[
          {
            key: "local", label: "账号密码登录",
            children: (
              <Form onFinish={handleLogin} size="large" autoComplete="off">
                <Form.Item name="username" rules={[{ required: true, message: "请输入用户名" }]}>
                  <Input prefix={<UserOutlined />} placeholder="用户名" />
                </Form.Item>
                <Form.Item name="password" rules={[{ required: true, message: "请输入密码" }]}>
                  <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                </Form.Item>
                <Form.Item name="mfa_code">
                  <Input prefix={<SafetyOutlined />} placeholder="动态验证码" maxLength={6} />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={loading} block size="large">
                    {loading ? "登录中..." : "登 录"}
                  </Button>
                </Form.Item>
              </Form>
            ),
          },
        ]} />
        {oidcEnabled && (
          <>
            <Divider plain>或</Divider>
            <Button block size="large" icon={<GoogleOutlined />} onClick={handleOIDCLogin}>
              使用 {oidcProvider} 登录
            </Button>
          </>
        )}
        <div style={{ textAlign: "center", color: "#bbb", fontSize: 12, marginTop: 16 }}>
          默认账号: admin / admin@2026
        </div>
      </Card>
    </div>
  );
}
