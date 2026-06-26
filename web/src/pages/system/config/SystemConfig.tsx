import { Card, Form, Switch, Input, Button, message, Divider } from "antd";

export default function SystemConfig() {
  const [form] = Form.useForm();

  const handleSave = async (values: any) => {
    try {
      console.log("Saving config:", values);
      message.success("配置保存成功");
    } catch (e: any) {
      message.error("保存失败");
    }
  };

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>系统配置</h2>
      <Card title="认证配置" style={{ maxWidth: 600 }}>
        <Form form={form} layout="vertical" onFinish={handleSave} initialValues={{
          ldap_enabled: false,
          ldap_host: "",
          ldap_port: 389,
          ldap_base_dn: "",
          oauth_enabled: false,
          oauth_provider: "",
        }}>
          <Divider>LDAP 配置</Divider>
          <Form.Item name="ldap_enabled" label="启用 LDAP" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="ldap_host" label="LDAP 服务器">
            <Input placeholder="ldap.example.com" />
          </Form.Item>
          <Form.Item name="ldap_port" label="端口">
            <Input placeholder="389" />
          </Form.Item>
          <Form.Item name="ldap_base_dn" label="Base DN">
            <Input placeholder="dc=example,dc=com" />
          </Form.Item>
          <Divider>OAuth 配置</Divider>
          <Form.Item name="oauth_enabled" label="启用 OAuth" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="oauth_provider" label="OAuth 提供商">
            <Input placeholder="Keycloak / Okta / 自定义" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">保存配置</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
