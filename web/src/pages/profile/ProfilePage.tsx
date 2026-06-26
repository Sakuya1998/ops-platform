import { useState } from "react";
import { Card, Form, Input, Button, message, Descriptions, Space, Tag, Typography } from "antd";
import { api } from "../../services/api";
import { useAuthStore } from "../../store/authStore";

const { Text } = Typography;

const passwordRules = [
  {
    pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^\w\s])\S{8,}$/,
    message: "密码至少 8 位，需包含大小写字母、数字和符号",
  },
];

export default function ProfilePage() {
  const [form] = Form.useForm();
  const [mfaForm] = Form.useForm();
  const [disableForm] = Form.useForm();
  const [recoveryForm] = Form.useForm();
  const [mfaSetup, setMfaSetup] = useState<{ secret: string; otpauth_url: string } | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [mfaLoading, setMfaLoading] = useState(false);
  const user = useAuthStore((s) => s.user);
  const loadUser = useAuthStore((s) => s.loadUser);

  const handleChangePassword = async (values: { old_password: string; new_password: string }) => {
    try {
      await api.put("/auth/me/password", {
        old_password: values.old_password,
        new_password: values.new_password,
      });
      message.success("密码修改成功，请使用新密码登录");
      form.resetFields();
    } catch (e: any) {
      message.error(e.response?.data?.message || "密码修改失败");
    }
  };

  const handleSetupMFA = async () => {
    setMfaLoading(true);
    try {
      const res = await api.post("/auth/me/mfa/setup");
      setMfaSetup(res.data.data);
      message.success("MFA 密钥已生成");
    } catch (e: any) {
      message.error(e.response?.data?.message || "生成 MFA 密钥失败");
    } finally {
      setMfaLoading(false);
    }
  };

  const handleConfirmMFA = async (values: { code: string }) => {
    setMfaLoading(true);
    try {
      const res = await api.post("/auth/me/mfa/confirm", { code: values.code });
      message.success("MFA 已启用");
      setRecoveryCodes(res.data.data?.recovery_codes || []);
      setMfaSetup(null);
      mfaForm.resetFields();
      await loadUser();
    } catch (e: any) {
      message.error(e.response?.data?.message || "MFA 验证失败");
    } finally {
      setMfaLoading(false);
    }
  };

  const handleRegenerateRecoveryCodes = async (values: { code: string }) => {
    setMfaLoading(true);
    try {
      const res = await api.post("/auth/me/mfa/recovery-codes", { code: values.code });
      setRecoveryCodes(res.data.data?.recovery_codes || []);
      recoveryForm.resetFields();
      message.success("恢复码已重新生成");
    } catch (e: any) {
      message.error(e.response?.data?.message || "重新生成恢复码失败");
    } finally {
      setMfaLoading(false);
    }
  };

  const handleDisableMFA = async (values: { code: string }) => {
    setMfaLoading(true);
    try {
      await api.delete("/auth/me/mfa", { data: { code: values.code } });
      message.success("MFA 已关闭");
      disableForm.resetFields();
      await loadUser();
    } catch (e: any) {
      message.error(e.response?.data?.message || "关闭 MFA 失败");
    } finally {
      setMfaLoading(false);
    }
  };

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>个人中心</h2>
      <Card title="个人信息" style={{ maxWidth: 720, marginBottom: 16 }}>
        <Descriptions column={1} size="small">
          <Descriptions.Item label="用户名">{user?.username || "-"}</Descriptions.Item>
          <Descriptions.Item label="显示名">{user?.display_name || "-"}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email || "-"}</Descriptions.Item>
          <Descriptions.Item label="组织 ID">{user?.org_id || "-"}</Descriptions.Item>
          <Descriptions.Item label="角色">{user?.roles?.join(", ") || "-"}</Descriptions.Item>
          <Descriptions.Item label="MFA">{user?.mfa_enabled ? <Tag color="green">已启用</Tag> : <Tag>未启用</Tag>}</Descriptions.Item>
        </Descriptions>
      </Card>
      <Card title="多因素认证" style={{ maxWidth: 720, marginBottom: 16 }}>
        {recoveryCodes.length > 0 && (
          <Space direction="vertical" style={{ width: "100%", marginBottom: 16 }}>
            {recoveryCodes.map((code) => <Text key={code} copyable>{code}</Text>)}
          </Space>
        )}
        {!user?.mfa_enabled && (
          <Space direction="vertical" style={{ width: "100%" }}>
            <Button type="primary" onClick={handleSetupMFA} loading={mfaLoading}>生成 MFA 密钥</Button>
            {mfaSetup && (
              <>
                <Text copyable>{mfaSetup.secret}</Text>
                <Text copyable type="secondary">{mfaSetup.otpauth_url}</Text>
                <Form form={mfaForm} layout="inline" onFinish={handleConfirmMFA}>
                  <Form.Item name="code" rules={[{ required: true, len: 6, message: "请输入 6 位验证码" }]}>
                    <Input placeholder="验证码" maxLength={6} />
                  </Form.Item>
                  <Form.Item>
                    <Button htmlType="submit" loading={mfaLoading}>确认启用</Button>
                  </Form.Item>
                </Form>
              </>
            )}
          </Space>
        )}
        {user?.mfa_enabled && (
          <Space direction="vertical" style={{ width: "100%" }}>
            <Form form={recoveryForm} layout="inline" onFinish={handleRegenerateRecoveryCodes}>
              <Form.Item name="code" rules={[{ required: true, len: 6, message: "请输入 6 位验证码" }]}>
                <Input placeholder="验证码" maxLength={6} />
              </Form.Item>
              <Form.Item>
                <Button htmlType="submit" loading={mfaLoading}>重新生成恢复码</Button>
              </Form.Item>
            </Form>
            <Form form={disableForm} layout="inline" onFinish={handleDisableMFA}>
              <Form.Item name="code" rules={[{ required: true, len: 6, message: "请输入 6 位验证码" }]}>
                <Input placeholder="验证码" maxLength={6} />
              </Form.Item>
              <Form.Item>
                <Button danger htmlType="submit" loading={mfaLoading}>关闭 MFA</Button>
              </Form.Item>
            </Form>
          </Space>
        )}
      </Card>
      <Card title="修改密码" style={{ maxWidth: 500 }}>
        <Form form={form} layout="vertical" onFinish={handleChangePassword}>
          <Form.Item
            name="old_password"
            label="当前密码"
            rules={[{ required: true, message: "请输入当前密码" }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[{ required: true, message: "请输入新密码" }, ...passwordRules]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label="确认密码"
            dependencies={["new_password"]}
            rules={[
              { required: true, message: "请确认新密码" },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue("new_password") === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error("两次输入的密码不一致"));
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">修改密码</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
