import { useEffect, useState } from "react";
import {
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Switch,
  message,
} from "antd";
import { ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { api } from "../../../services/api";

type LDAPConfig = {
  enabled: boolean;
  host: string;
  port: number;
  security: "" | "tls" | "starttls";
  bind_dn: string;
  bind_password: string;
  base_dn: string;
  user_filter: string;
  uid_attr: string;
  display_name_attr: string;
  email_attr: string;
  auto_provision: boolean;
  default_org_code: string;
  skip_verify: boolean;
};

type OIDCConfig = {
  enabled: boolean;
  provider_name: string;
  issuer: string;
  client_id: string;
  client_secret: string;
  redirect_uri: string;
  scopes: string[];
  auto_provision: boolean;
  default_org_code: string;
};

type SystemConfigForm = {
  ldap: LDAPConfig;
  oidc: OIDCConfig;
};

const DEFAULT_ORG_ID = "00000000-0000-0000-0000-000000000001";

const defaultValues: SystemConfigForm = {
  ldap: {
    enabled: false,
    host: "",
    port: 389,
    security: "",
    bind_dn: "",
    bind_password: "",
    base_dn: "",
    user_filter: "(uid=%s)",
    uid_attr: "uid",
    display_name_attr: "cn",
    email_attr: "mail",
    auto_provision: true,
    default_org_code: DEFAULT_ORG_ID,
    skip_verify: false,
  },
  oidc: {
    enabled: false,
    provider_name: "",
    issuer: "",
    client_id: "",
    client_secret: "",
    redirect_uri: "",
    scopes: ["openid", "profile", "email"],
    auto_provision: true,
    default_org_code: DEFAULT_ORG_ID,
  },
};

function normalizeConfig(config?: Partial<SystemConfigForm>): SystemConfigForm {
  return {
    ldap: {
      ...defaultValues.ldap,
      ...(config?.ldap || {}),
      port: config?.ldap?.port || defaultValues.ldap.port,
      default_org_code: config?.ldap?.default_org_code || DEFAULT_ORG_ID,
    },
    oidc: {
      ...defaultValues.oidc,
      ...(config?.oidc || {}),
      scopes: config?.oidc?.scopes?.length ? config.oidc.scopes : defaultValues.oidc.scopes,
      default_org_code: config?.oidc?.default_org_code || DEFAULT_ORG_ID,
    },
  };
}

export default function SystemConfigPage() {
  const [form] = Form.useForm<SystemConfigForm>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const res = await api.get("/system/config");
      form.setFieldsValue(normalizeConfig(res.data.data));
    } catch (e: any) {
      message.error(e.response?.data?.message || "加载系统配置失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    form.setFieldsValue(defaultValues);
    loadConfig();
  }, []);

  const handleSave = async (values: SystemConfigForm) => {
    setSaving(true);
    try {
      const payload = normalizeConfig(values);
      await api.put("/system/config", payload);
      message.success("系统配置已保存");
      form.setFieldsValue(payload);
    } catch (e: any) {
      message.error(e.response?.data?.message || "保存系统配置失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>系统配置</h2>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadConfig} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<SaveOutlined />} onClick={() => form.submit()} loading={saving}>
            保存
          </Button>
        </Space>
      </div>

      <Card title="认证提供商配置" loading={loading}>
        <Form<SystemConfigForm>
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={defaultValues}
          disabled={saving}
        >
          <Divider orientation="left">LDAP</Divider>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "enabled"]} label="启用 LDAP" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "auto_provision"]} label="自动创建用户" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "skip_verify"]} label="跳过 TLS 证书校验" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={10}>
              <Form.Item name={["ldap", "host"]} label="LDAP 地址">
                <Input placeholder="ldap.example.com" />
              </Form.Item>
            </Col>
            <Col xs={12} md={4}>
              <Form.Item name={["ldap", "port"]} label="端口">
                <InputNumber min={1} max={65535} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item name={["ldap", "security"]} label="连接安全">
                <Select
                  options={[
                    { label: "Plain", value: "" },
                    { label: "TLS", value: "tls" },
                    { label: "StartTLS", value: "starttls" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item name={["ldap", "bind_dn"]} label="Bind DN">
                <Input placeholder="cn=admin,dc=example,dc=com" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name={["ldap", "bind_password"]} label="Bind Password">
                <Input.Password placeholder="服务账号密码" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item name={["ldap", "base_dn"]} label="Base DN">
                <Input placeholder="dc=example,dc=com" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name={["ldap", "user_filter"]} label="用户过滤器">
                <Input placeholder="(uid=%s)" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "uid_attr"]} label="账号属性">
                <Input placeholder="uid" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "display_name_attr"]} label="显示名属性">
                <Input placeholder="cn" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={["ldap", "email_attr"]} label="邮箱属性">
                <Input placeholder="mail" />
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left">OIDC</Divider>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name={["oidc", "enabled"]} label="启用 OIDC" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={["oidc", "auto_provision"]} label="自动创建用户" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name={["oidc", "provider_name"]} label="提供商名称">
                <Input placeholder="Keycloak / Okta / Authing" />
              </Form.Item>
            </Col>
            <Col xs={24} md={16}>
              <Form.Item name={["oidc", "issuer"]} label="Issuer">
                <Input placeholder="https://idp.example.com/realms/ops" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item name={["oidc", "client_id"]} label="Client ID">
                <Input placeholder="ops-platform" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name={["oidc", "client_secret"]} label="Client Secret">
                <Input.Password placeholder="客户端密钥" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item name={["oidc", "redirect_uri"]} label="回调地址">
                <Input placeholder="http://localhost:3000/api/v1/auth/oidc/callback" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name={["oidc", "scopes"]} label="Scopes">
                <Select mode="tags" tokenSeparators={[",", " "]} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>
    </div>
  );
}
