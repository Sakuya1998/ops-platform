package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type apisixRouteConfig struct {
	Routes    []apisixRoute    `yaml:"routes"`
	Upstreams []apisixUpstream `yaml:"upstreams"`
}

type apisixRuntimeConfig struct {
	Deployment apisixDeploymentConfig `yaml:"deployment"`
}

type apisixDeploymentConfig struct {
	Role          string                `yaml:"role"`
	RoleDataPlane apisixDataPlaneConfig `yaml:"role_data_plane"`
}

type apisixDataPlaneConfig struct {
	ConfigProvider string `yaml:"config_provider"`
}

type apisixRoute struct {
	URI        string `yaml:"uri"`
	Name       string `yaml:"name"`
	UpstreamID string `yaml:"upstream_id"`
}

type apisixUpstream struct {
	ID          string `yaml:"id"`
	ServiceName string `yaml:"service_name"`
}

type composeConfig struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Build       any                         `yaml:"build"`
	Ports       []string                    `yaml:"ports"`
	Expose      []string                    `yaml:"expose"`
	Environment map[string]string           `yaml:"environment"`
	Volumes     []string                    `yaml:"volumes"`
	Healthcheck map[string]any              `yaml:"healthcheck"`
	DependsOn   map[string]composeDependsOn `yaml:"depends_on"`
}

type composeDependsOn struct {
	Condition string `yaml:"condition"`
}

func TestAPISIXClientRoutesOnlyTargetBFF(t *testing.T) {
	cfg := loadAPISIXRoutes(t)

	if len(cfg.Upstreams) != 1 {
		t.Fatalf("client APISIX config should expose exactly one upstream, got %d", len(cfg.Upstreams))
	}
	if cfg.Upstreams[0].ID != "bff-api" || cfg.Upstreams[0].ServiceName != "bff-api" {
		t.Fatalf("client APISIX upstream should be bff-api, got %+v", cfg.Upstreams[0])
	}

	for _, route := range cfg.Routes {
		if route.UpstreamID != "bff-api" {
			t.Fatalf("route %s (%s) should target bff-api, got %q", route.Name, route.URI, route.UpstreamID)
		}
	}
}

func TestAPISIXRoutesCoverBFFClientAPI(t *testing.T) {
	cfg := loadAPISIXRoutes(t)
	routes := map[string]bool{}
	for _, route := range cfg.Routes {
		routes[route.URI] = true
	}

	required := []string{
		"/api/v1/bootstrap",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/token/verify",
		"/api/v1/auth/oidc/login",
		"/api/v1/auth/oidc/callback",
		"/api/v1/auth/oidc/status",
		"/api/v1/auth/oidc/exchange",
		"/api/v1/auth/me",
		"/api/v1/auth/logout",
		"/api/v1/auth/me/password",
		"/api/v1/auth/me/mfa*",
		"/api/v1/auth/sessions*",
		"/api/v1/users/*/roles",
		"/api/v1/users*",
		"/api/v1/roles*",
		"/api/v1/permissions*",
		"/api/v1/api-permissions*",
		"/api/v1/organizations*",
		"/api/v1/audit-logs*",
		"/api/v1/notifications*",
		"/api/v1/notify/templates*",
		"/api/v1/notify/logs*",
		"/api/v1/system/config*",
		"/health",
		"/health/*",
	}

	for _, uri := range required {
		if !routes[uri] {
			t.Fatalf("apisix route %s is required by BFF client API but missing", uri)
		}
	}
}

func TestAPISIXComposeUsesStandaloneDeclarativeRoutes(t *testing.T) {
	runtime := loadAPISIXRuntimeConfig(t)
	if runtime.Deployment.Role != "data_plane" {
		t.Fatalf("APISIX local compose should use file-driven data_plane mode, got role %q", runtime.Deployment.Role)
	}
	if runtime.Deployment.RoleDataPlane.ConfigProvider != "yaml" {
		t.Fatalf("APISIX local compose should load conf/apisix.yaml, got provider %q", runtime.Deployment.RoleDataPlane.ConfigProvider)
	}

	compose := loadComposeConfig(t)
	apisix, ok := compose.Services["apisix"]
	if !ok {
		t.Fatalf("compose service apisix missing")
	}
	foundRoutesMount := false
	for _, volume := range apisix.Volumes {
		if volume == "../apisix/routes.yaml:/usr/local/apisix/conf/apisix.yaml:ro" {
			foundRoutesMount = true
		}
	}
	if !foundRoutesMount {
		t.Fatalf("apisix routes.yaml should be mounted to /usr/local/apisix/conf/apisix.yaml, got %v", apisix.Volumes)
	}
}

func TestComposeDoesNotPublishInternalServicePorts(t *testing.T) {
	cfg := loadComposeConfig(t)
	internalServices := []string{"bff-api", "auth-svc", "iam-svc", "audit-svc", "notify-svc"}
	for _, name := range internalServices {
		service, ok := cfg.Services[name]
		if !ok {
			t.Fatalf("compose service %s missing", name)
		}
		if len(service.Ports) > 0 {
			t.Fatalf("compose service %s must not publish host ports, got %v", name, service.Ports)
		}
		if len(service.Expose) == 0 {
			t.Fatalf("compose service %s should expose ports only to the compose network", name)
		}
	}
}

func TestComposeGoServicesUseRepositoryBuildContext(t *testing.T) {
	cfg := loadComposeConfig(t)
	services := []string{"bff-api", "auth-svc", "iam-svc", "audit-svc", "notify-svc"}

	for _, name := range services {
		service, ok := cfg.Services[name]
		if !ok {
			t.Fatalf("compose service %s missing", name)
		}
		build, ok := service.Build.(map[string]any)
		if !ok {
			t.Fatalf("compose service %s build should be an object with context/dockerfile, got %T", name, service.Build)
		}
		if build["context"] != ".." {
			t.Fatalf("compose service %s build context should be repo root '..', got %v", name, build["context"])
		}
		wantDockerfile := "services/" + name + "/Dockerfile"
		if build["dockerfile"] != wantDockerfile {
			t.Fatalf("compose service %s dockerfile should be %s, got %v", name, wantDockerfile, build["dockerfile"])
		}
	}
}

func TestComposeRegistersServicesWithResolvableAddresses(t *testing.T) {
	cfg := loadComposeConfig(t)
	services := []string{"bff-api", "auth-svc", "iam-svc", "audit-svc", "notify-svc"}

	for _, name := range services {
		service, ok := cfg.Services[name]
		if !ok {
			t.Fatalf("compose service %s missing", name)
		}
		if service.Environment["OPS_SERVICE_ADDRESS"] != name {
			t.Fatalf("compose service %s should register itself in Consul as %q, got %q", name, name, service.Environment["OPS_SERVICE_ADDRESS"])
		}
	}
}

func TestComposeUsesHealthChecksForStartupOrder(t *testing.T) {
	cfg := loadComposeConfig(t)
	servicesRequiringHealthchecks := []string{
		"etcd", "consul", "postgres", "redis", "kafka",
		"bff-api", "auth-svc", "iam-svc", "audit-svc", "notify-svc",
	}

	for _, name := range servicesRequiringHealthchecks {
		service, ok := cfg.Services[name]
		if !ok {
			t.Fatalf("compose service %s missing", name)
		}
		if len(service.Healthcheck) == 0 {
			t.Fatalf("compose service %s should define a healthcheck", name)
		}
	}

	assertHealthyDependency(t, cfg, "auth-svc", "postgres")
	assertHealthyDependency(t, cfg, "auth-svc", "redis")
	assertHealthyDependency(t, cfg, "auth-svc", "kafka")
	assertHealthyDependency(t, cfg, "auth-svc", "consul")

	assertHealthyDependency(t, cfg, "iam-svc", "postgres")
	assertHealthyDependency(t, cfg, "iam-svc", "redis")
	assertHealthyDependency(t, cfg, "iam-svc", "kafka")
	assertHealthyDependency(t, cfg, "iam-svc", "consul")

	assertHealthyDependency(t, cfg, "audit-svc", "postgres")
	assertHealthyDependency(t, cfg, "audit-svc", "kafka")
	assertHealthyDependency(t, cfg, "audit-svc", "consul")

	assertHealthyDependency(t, cfg, "notify-svc", "postgres")
	assertHealthyDependency(t, cfg, "notify-svc", "kafka")
	assertHealthyDependency(t, cfg, "notify-svc", "consul")

	assertHealthyDependency(t, cfg, "bff-api", "auth-svc")
	assertHealthyDependency(t, cfg, "bff-api", "iam-svc")
	assertHealthyDependency(t, cfg, "bff-api", "audit-svc")
	assertHealthyDependency(t, cfg, "bff-api", "notify-svc")
	assertHealthyDependency(t, cfg, "bff-api", "consul")

	assertHealthyDependency(t, cfg, "apisix", "consul")
	assertHealthyDependency(t, cfg, "apisix", "bff-api")
}

func TestLinuxSmokeScriptsExistAndMatchPowerShellFlow(t *testing.T) {
	initDB := readScript(t, "init-db.sh")
	requiredInitDBSnippets := []string{
		"#!/usr/bin/env bash",
		"CREATE DATABASE",
		"auth_svc",
		"iam_svc",
		"audit_svc",
		"notify_svc",
		"services/${svc}/migrations/001_init.sql",
	}
	for _, snippet := range requiredInitDBSnippets {
		if !strings.Contains(initDB, snippet) {
			t.Fatalf("scripts/init-db.sh should contain %q", snippet)
		}
	}

	genProto := readScript(t, "gen-proto.sh")
	requiredGenProtoSnippets := []string{
		"#!/usr/bin/env bash",
		"pkg/proto",
		"protoc",
		"--go_out=.",
		"--go-grpc_out=.",
		"protoc-gen-go",
		"protoc-gen-go-grpc",
	}
	for _, snippet := range requiredGenProtoSnippets {
		if !strings.Contains(genProto, snippet) {
			t.Fatalf("scripts/gen-proto.sh should contain %q", snippet)
		}
	}

	seed := readScript(t, "seed.sh")
	requiredSeedSnippets := []string{
		"#!/usr/bin/env bash",
		"--use-docker-compose",
		"docker compose -f",
		"INSERT INTO organizations",
		"INSERT INTO users",
		"Default credentials: admin / admin@2026",
	}
	for _, snippet := range requiredSeedSnippets {
		if !strings.Contains(seed, snippet) {
			t.Fatalf("scripts/seed.sh should contain %q", snippet)
		}
	}

	smoke := readScript(t, "smoke-compose.sh")
	requiredSmokeSnippets := []string{
		"#!/usr/bin/env bash",
		"docker compose -f",
		"scripts/seed.sh",
		"/api/v1/auth/login",
		"/api/v1/auth/me",
		"/api/v1/users",
		"/api/v1/roles",
		"/api/v1/audit-logs",
	}
	for _, snippet := range requiredSmokeSnippets {
		if !strings.Contains(smoke, snippet) {
			t.Fatalf("scripts/smoke-compose.sh should contain %q", snippet)
		}
	}
}

func assertHealthyDependency(t *testing.T, cfg composeConfig, serviceName, dependencyName string) {
	t.Helper()
	service, ok := cfg.Services[serviceName]
	if !ok {
		t.Fatalf("compose service %s missing", serviceName)
	}
	dependency, ok := service.DependsOn[dependencyName]
	if !ok {
		t.Fatalf("compose service %s should depend on %s", serviceName, dependencyName)
	}
	if dependency.Condition != "service_healthy" {
		t.Fatalf("compose service %s dependency %s should use service_healthy, got %q", serviceName, dependencyName, dependency.Condition)
	}
}

func readScript(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "..", "..", "scripts", name))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read script %s: %v", name, err)
	}
	return string(data)
}

func loadAPISIXRoutes(t *testing.T) apisixRouteConfig {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "..", "..", "apisix", "routes.yaml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read apisix routes: %v", err)
	}
	var cfg apisixRouteConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse apisix routes: %v", err)
	}
	return cfg
}

func loadAPISIXRuntimeConfig(t *testing.T) apisixRuntimeConfig {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "..", "..", "apisix", "config.yaml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read apisix config: %v", err)
	}
	var cfg apisixRuntimeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse apisix config: %v", err)
	}
	return cfg
}

func loadComposeConfig(t *testing.T) composeConfig {
	t.Helper()
	path := filepath.Clean(filepath.Join("..", "..", "..", "..", "deploy", "docker-compose.yml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compose config: %v", err)
	}
	var cfg composeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse compose config: %v", err)
	}
	return cfg
}
