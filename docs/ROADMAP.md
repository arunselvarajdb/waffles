# MCP Gateway Feature Roadmap

## Overview

This document outlines the implementation plan for key features to make MCP Gateway production-ready: User Management, Metrics Dashboard, Guardrails/Security, and Kubernetes deployment.

**Key Decisions:**

- **Deployment Target**: Kubernetes (production-first)
- **Rate Limiting**: AWS managed services (API Gateway / WAF) - NOT in application scope
- **Authentication**: OAuth, LDAP/AD, Local DB (configurable, can enable multiple)
- **API Keys**: Application-managed with scopes (not AWS API Gateway keys)
- **Development Approach**: Parallel tracks (security, user management, observability)
- **Email Integration**: Later phase (admin-only flows initially)
- **MCP Server Model**: Internal/self-hosted only (no external registries)

---

## Parallel Development Tracks

### Track A: Security Guardrails

#### A.1 Rate Limiting (AWS Managed - Out of Scope)

> **Note**: Rate limiting will be handled by AWS managed services (API Gateway, WAF, ALB) at the infrastructure level, not in application code.

**AWS Options (infrastructure config, not application code):**

- **AWS API Gateway**: Built-in rate limiting per API key / per client
- **AWS WAF**: Rate-based rules with automatic DDoS protection
- **AWS ALB**: Connection limits and request throttling

This removes the need for Redis and application-level rate limiting middleware.

#### A.2 Authentication Providers (Configurable)

**Three authentication modes (config-driven, can enable multiple):**

| Mode | Config | Account Security | Use Case |
|------|--------|------------------|----------|
| **OAuth/OIDC** | `auth.oauth.enabled` | Managed by IdP | SSO with Keycloak, Okta, Azure AD |
| **LDAP/AD** | `auth.ldap.enabled` | Managed by AD | Direct AD integration |
| **Local DB** | `auth.local.enabled` | App handles lockout | Development, small deployments |

**OAuth/OIDC (existing):**

```yaml
auth:
  oauth:
    enabled: true
    provider: "keycloak"  # or "okta", "azure", "generic"
    # ... existing OAuth config
```

**LDAP/AD (new):**

```yaml
auth:
  ldap:
    enabled: true
    url: "ldaps://ad.example.com:636"
    base_dn: "dc=example,dc=com"
    bind_dn: "cn=svc-mcp,ou=services,dc=example,dc=com"
    bind_password: "${LDAP_BIND_PASSWORD}"
    user_filter: "(sAMAccountName={username})"
    group_filter: "(member={dn})"
    group_mappings:
      "CN=MCP-Admins,OU=Groups,DC=example,DC=com": "admin"
      "CN=MCP-Operators,OU=Groups,DC=example,DC=com": "operator"
      "CN=MCP-Viewers,OU=Groups,DC=example,DC=com": "viewer"
```

**Local DB (existing, enhanced):**

```yaml
auth:
  local:
    enabled: true  # Default for dev
    lockout:
      max_attempts: 5
      duration: "15m"
    password_policy:
      min_length: 12
      require_uppercase: true
      require_number: true
      require_special: true
```

**Files:**

- `internal/service/auth/ldap.go` (new - LDAP authentication)
- `internal/service/auth/provider.go` (new - auth provider interface)
- `internal/config/config.go` (add LDAP config)
- `internal/handler/auth.go` (support multiple providers)

#### A.3 HTTP Security Headers

New middleware `internal/handler/middleware/security.go`:

```text
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

#### A.4 SSRF Protection

- Validate server URLs against private IP ranges before registration
- Block: localhost, 127.x.x.x, 10.x.x.x, 192.168.x.x, 172.16-31.x.x, 169.254.x.x

**Files:**

- `internal/service/registry/validation.go` (new)

#### A.5 Tool Execution Validation

- Enforce `AllowedTools` field in gateway proxy
- Parse tool call requests, validate against whitelist
- Return 403 if tool not allowed

**Files:**

- `internal/service/gateway/service.go` (add validation before proxy)

---

### Track B: User Management Admin Interface

#### B.1 Backend API Endpoints

New handlers in `internal/handler/admin/`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/users` | GET | List all users (paginated, searchable) |
| `/api/v1/admin/users/:id` | GET | Get user details with roles |
| `/api/v1/admin/users` | POST | Create user (admin sets temp password) |
| `/api/v1/admin/users/:id` | PUT | Update user (name, email, active) |
| `/api/v1/admin/users/:id` | DELETE | Deactivate user |
| `/api/v1/admin/users/:id/roles` | PUT | Assign/revoke roles |
| `/api/v1/admin/users/:id/reset-password` | POST | Admin resets to temp password |
| `/api/v1/admin/users/:id/sessions` | GET | List active sessions |
| `/api/v1/admin/users/:id/sessions/:sid` | DELETE | Revoke session |

**Files:**

- `internal/handler/admin/users.go` (new)
- `internal/handler/admin/sessions.go` (new)
- `internal/service/user/service.go` (new - user CRUD service)

#### B.2 Role Management API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/roles` | GET | List all roles with user counts |
| `/api/v1/admin/roles/:id` | GET | Get role with permissions |
| `/api/v1/admin/roles` | POST | Create custom role |
| `/api/v1/admin/roles/:id` | PUT | Update role permissions |
| `/api/v1/admin/roles/:id` | DELETE | Delete custom role (not built-in) |

**Files:**

- `internal/handler/admin/roles.go` (new)
- `internal/service/role/service.go` (new)

#### B.3 Frontend Components

**Views:**

- `web-app/src/views/AdminUsers.vue` - User list with search/filter
- `web-app/src/views/AdminRoles.vue` - Role and permission management

**Components:**

- `web-app/src/components/users/UserTable.vue`
- `web-app/src/components/users/UserFormModal.vue`
- `web-app/src/components/users/RoleAssignmentModal.vue`
- `web-app/src/components/users/SessionsTable.vue`

**Stores:**

- `web-app/src/stores/users.ts` (new - Pinia store)

#### B.4 User Lifecycle (Phase 2 - Email)

- Invitation system via email (deferred)
- Password reset via email (deferred)
- For now: Admin creates users with temp passwords

---

### Track C: Metrics Dashboard & Observability

#### C.1 Backend: Wire Up Missing Collectors

- Instantiate `ServerHealthCollector` in `cmd/server/main.go`
- Schedule periodic health checks (30-second interval)

**Files:**

- `cmd/server/main.go` (add collector initialization)

#### C.2 Prometheus Configuration

Create `monitoring/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: 'waffles'
    static_configs:
      - targets: ['waffles:9090']
```

#### C.3 Grafana Dashboards

Create `monitoring/grafana/dashboards/`:

| Dashboard | Key Panels |
|-----------|------------|
| `waffles-overview.json` | Request rate, latency percentiles, error rate, active connections |
| `waffles-servers.json` | Per-server health, proxy latency, request count |
| `waffles-database.json` | Connection pool (open/idle/waiting), query latency |

#### C.4 Frontend Metrics View

**Option A: Embed Grafana** (recommended for production)

- Grafana iframe with anonymous auth
- Configure Grafana for embedding

**Option B: Custom Vue Dashboard**

- `web-app/src/views/MetricsDashboard.vue`
- API endpoints to query aggregated metrics
- Chart.js for visualization

**Files (Option B):**

- `internal/handler/metrics.go` (new - metrics query API)
- `web-app/src/views/MetricsDashboard.vue`
- `web-app/src/components/metrics/RequestChart.vue`
- `web-app/src/components/metrics/ServerHealthGrid.vue`

#### C.5 Alerting Rules

Create `monitoring/prometheus/alerts.yml`:

```yaml
groups:
  - name: waffles
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
      - alert: ServerUnhealthy
        expr: gateway_server_health_status == 0
      - alert: HighLatency
        expr: histogram_quantile(0.99, http_request_duration_seconds) > 1
```

---

### Track D: Kubernetes Operator & Production Deploy

#### D.1 Helm Chart

Create `deploy/helm/waffles/`:

```text
deploy/helm/waffles/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── ingress.yaml
│   ├── serviceaccount.yaml
│   ├── hpa.yaml (horizontal pod autoscaler)
│   └── servicemonitor.yaml (Prometheus Operator)
```

**Key Helm Features:**

- PostgreSQL via Bitnami subchart or external DB
- Ingress with TLS (cert-manager annotations)
- Resource limits and requests
- Health/readiness probes
- ServiceMonitor for Prometheus Operator

#### D.2 Kubernetes Operator (Phase 2)

**CRD: MCPServer**

```yaml
apiVersion: mcp.gateway.io/v1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  image: my-mcp-server:latest
  resources:
    limits: { cpu: "500m", memory: "256Mi" }
  secrets:
    - name: api-key
      envVar: MCP_API_KEY
  healthCheck:
    path: /health
    interval: 30s
```

**Operator Logic:**

- Watch MCPServer CRDs
- Create Deployment + Service for each
- Register with gateway via API
- Monitor health, update status

**Files:**

- `deploy/operator/` (kubebuilder scaffolding)
- `api/v1/mcpserver_types.go`
- `controllers/mcpserver_controller.go`

#### D.3 Client Auto-Configuration

Endpoint: `GET /api/v1/client-config/:client_type`

Supported clients:

- Claude Code (`claude-code`)
- Cursor (`cursor`)
- Continue (`continue`)
- GitHub Copilot Chat (`copilot`)

Returns ready-to-use configuration JSON/YAML

**Files:**

- `internal/handler/clientconfig.go` (new)

---

### Track E: Internal MCP Server Management

> **Goal**: Secure lifecycle for internal/self-hosted MCP servers only.

#### E.1 Internal-Only Server Model

**Constraint**: All MCP servers are deployed internally (no external URLs)

| Approach | Description |
|----------|-------------|
| **K8s Operator** | Gateway provisions server containers via CRD |
| **Docker Compose** | Gateway manages containers via Docker API |
| **Manual** | Admin registers pre-deployed internal services |

**URL Validation** (enforce internal-only):

```go
var allowedNetworks = []string{
    "10.0.0.0/8",      // Private
    "172.16.0.0/12",   // Private
    "192.168.0.0/16",  // Private
    "*.svc.cluster.local",  // K8s internal
}

func validateInternalURL(url string) error {
    // MUST be in allowed networks
    // REJECT external/public IPs
}
```

**Config:**

```yaml
security:
  servers:
    internal_only: true              # Enforce internal URLs
    allowed_networks: ["10.0.0.0/8", "*.cluster.local"]
    require_k8s_service: false       # Require K8s service discovery
```

**Files:**

- `internal/service/registry/validation.go` (URL validation)

#### E.2 Server Provisioning Modes

**Mode 1: Operator-Managed (Recommended)**

```yaml
# MCPServer CRD - Gateway creates the container
apiVersion: mcp.gateway.io/v1
kind: MCPServer
spec:
  image: internal-registry/weather-mcp:v1.2
  resources: { limits: { cpu: "500m", memory: "256Mi" } }
  secrets:
    - name: api-key
      envVar: WEATHER_API_KEY
```

- Gateway K8s operator creates Deployment + Service
- Auto-registers in gateway database
- Full lifecycle control (deploy, update, delete)

**Mode 2: Pre-Deployed Registration**

```json
{
    "name": "weather-service",
    "url": "http://weather-mcp.mcp-servers.svc.cluster.local:8080",
    "description": "Internal weather MCP server"
}
```

- Admin deploys server separately
- Registers URL with gateway
- Gateway validates URL is internal

#### E.3 Container Isolation & Security

Each MCP server runs in isolated container:

```yaml
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault

resources:
  limits:
    cpu: "500m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
```

**Network Policies:**

```yaml
# Only gateway can talk to MCP servers
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mcp-server-isolation
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: waffles
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: waffles
```

**Files:**

- `deploy/operator/templates/deployment.yaml`
- `deploy/operator/templates/networkpolicy.yaml`

#### E.4 Secret Injection (Not Stored in Gateway)

**Principle**: Gateway doesn't store MCP server secrets

| Method | Description |
|--------|-------------|
| **K8s Secrets** | Mount secrets directly into server pods |
| **External Secrets Operator** | Sync from Vault/AWS SM to K8s secrets |
| **Environment Variables** | Injected at container runtime |

```yaml
# MCPServer CRD with secret references
spec:
  secrets:
    - name: weather-api-key      # K8s Secret name
      envVar: WEATHER_API_KEY    # Env var in container
    - name: database-creds
      mountPath: /secrets/db     # File mount
```

**Gateway stores only:**

- Server URL (internal)
- Health check config
- Tool/resource restrictions
- NO authentication credentials for the server itself

#### E.5 Server Lifecycle States

| State | Description | Transitions |
|-------|-------------|-------------|
| `provisioning` | Container being created | → `healthy` or `failed` |
| `healthy` | Passing health checks | → `degraded`, `unhealthy` |
| `degraded` | Partial failures | → `healthy`, `unhealthy` |
| `unhealthy` | Failing health checks | → `healthy`, `stopped` |
| `stopped` | Manually disabled | → `provisioning` (restart) |
| `failed` | Provisioning failed | → `provisioning` (retry) |

#### E.6 Tool Allowlisting

Since servers are internal, focus on **what tools are exposed**:

```go
type MCPServer struct {
    AllowedTools []string  // Empty = all tools allowed
    ToolPolicy   ToolPolicy
}

type ToolPolicy struct {
    Mode          string   // "allowlist" or "denylist"
    AllowedTools  []string // If allowlist mode
    DeniedTools   []string // If denylist mode
    RequireReview []string // Tools that need admin approval before use
}
```

**Dangerous tool patterns** (auto-flag for review):

- `*exec*`, `*shell*`, `*eval*`
- `*file_write*`, `*delete*`
- `*network*`, `*http_request*`

#### E.7 Audit & Compliance

Track all server lifecycle events:

```sql
CREATE TABLE server_events (
    id UUID PRIMARY KEY,
    server_id UUID REFERENCES mcp_servers(id),
    event_type VARCHAR(50),
    triggered_by UUID REFERENCES users(id),
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Files:**

- `internal/domain/server_event.go` (new)
- `internal/repository/server_event.go` (new)

---

### Track F: Least Privilege & Credential Scoping

> **Goal**: Limit token scope, avoid broad permissions, treat auth requests with skepticism.
> **Decision**: Application-managed API keys (not AWS API Gateway keys) for full control over MCP-specific scoping.

#### F.1 Scoped API Keys

**Current state**: API keys inherit ALL user permissions (no scoping)

**New model**: API keys have explicit capability restrictions

```go
type APIKey struct {
    Scopes         []string `json:"scopes"`
    AllowedServers []string `json:"allowed_servers"`
    AllowedTools   []string `json:"allowed_tools"`
    Namespaces     []string `json:"namespaces"`
    IPWhitelist    []string `json:"ip_whitelist"`
    ReadOnly       bool     `json:"read_only"`
}
```

**Predefined scopes:**

| Scope | Description |
|-------|-------------|
| `servers:read` | List/view servers |
| `servers:write` | Create/update/delete servers |
| `gateway:execute` | Call MCP tools/resources |
| `audit:read` | View audit logs |
| `users:read` | View users (admin) |
| `users:write` | Manage users (admin) |

**Database migration:**

```sql
ALTER TABLE api_keys ADD COLUMN scopes TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN allowed_servers UUID[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN allowed_tools TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN namespaces UUID[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN ip_whitelist TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN read_only BOOLEAN DEFAULT false;
```

**Files:**

- `internal/domain/apikey.go` (add scope fields)
- `internal/database/migrations/000009_apikey_scopes.up.sql` (new)
- `internal/handler/middleware/auth.go` (enforce scopes)

#### F.2 Scope Enforcement Middleware

```go
func (m *AuthMiddleware) RequireScope(scope string) gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := GetAPIKeyFromContext(c)
        if apiKey != nil {
            if !apiKey.HasScope(scope) {
                c.AbortWithStatus(403)
                return
            }
            if !apiKey.IsIPAllowed(c.ClientIP()) {
                c.AbortWithStatus(403)
                return
            }
        }
        c.Next()
    }
}
```

**Files:**

- `internal/handler/middleware/scope.go` (new)
- `internal/server/router.go` (add scope middleware to routes)

#### F.3 API Key Management UI

**User-facing view** (users manage their own keys):

- `web-app/src/views/ApiKeys.vue` - List user's API keys, create/revoke
- Create key wizard with scope selection
- Copy key to clipboard (shown only once)
- View key usage stats

**Components:**

- `web-app/src/components/apikeys/ApiKeyTable.vue`
- `web-app/src/components/apikeys/CreateKeyModal.vue`
- `web-app/src/components/apikeys/ScopeSelector.vue`
- `web-app/src/components/apikeys/ServerPicker.vue`

**Files:**

- `web-app/src/views/ApiKeys.vue` (new)
- `web-app/src/components/apikeys/*.vue` (new)
- `web-app/src/stores/apikeys.ts` (new)

---

### Track G: Guardrail Policy Engine

> **Goal**: Prevent attacks via configurable input/output validation policies.
> **Reference**: [Enkrypt AI Secure MCP Gateway](https://github.com/enkryptai/secure-mcp-gateway)

#### G.1 Guardrail Architecture

```text
Request Flow with Guardrails:

User Request
  → Authentication
  → Rate Limiting (AWS)
  → INPUT GUARDRAILS ←── Policy Engine
      ├── Prompt Injection Detection
      ├── PII Detection & Redaction
      ├── Keyword Filtering
      └── Custom Policy Validation
  → MCP Server (if passed)
  → OUTPUT GUARDRAILS ←── Policy Engine
      ├── PII Detection (re-check)
      ├── Policy Adherence Check
      └── PII Un-redaction (restore)
  → Response to User
```

**Core Components:**

```go
type GuardrailEngine struct {
    policies    []Policy
    detectors   map[string]Detector
    redactors   map[string]Redactor
    cache       Cache
}

type GuardrailResult struct {
    Passed     bool
    Violations []Violation
    Redactions []Redaction
}
```

**Files:**

- `internal/service/guardrails/engine.go` (new)
- `internal/service/guardrails/policy.go` (new)
- `internal/handler/middleware/guardrails.go` (new)

#### G.2 Input Protection Detectors

| Detector | Description | Action |
|----------|-------------|--------|
| **Prompt Injection** | Detect encoded/embedded commands | Block request |
| **PII Detection** | Find SSN, credit cards, emails, phones | Redact before proxy |
| **Keyword Filter** | Match against forbidden terms | Block or warn |
| **Toxicity Detection** | Hostile/abusive language | Block request |

**Files:**

- `internal/service/guardrails/detectors/injection.go`
- `internal/service/guardrails/detectors/pii.go`
- `internal/service/guardrails/detectors/keyword.go`

#### G.3 Policy Configuration

**Per-Server Guardrail Policies:**

```yaml
policies:
  - name: "strict-financial"
    description: "High security for financial data"
    servers: ["payment-mcp", "banking-mcp"]
    input:
      block_injection: true
      redact_pii: true
      forbidden_keywords: ["password", "secret", "key"]
    output:
      check_pii_leakage: true

  - name: "standard"
    description: "Default policy for internal tools"
    servers: ["*"]
    input:
      block_injection: true
      redact_pii: false
```

**Database schema:**

```sql
CREATE TABLE guardrail_policies (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE,
    description TEXT,
    config JSONB,
    is_default BOOLEAN DEFAULT false,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Files:**

- `internal/domain/guardrail_policy.go` (new)
- `internal/repository/guardrail_policy.go` (new)
- `internal/database/migrations/000010_guardrails.up.sql` (new)

#### G.4 Guardrail Summary

| Attack Vector | Guardrail | Action |
|---------------|-----------|--------|
| Prompt Injection | InjectionDetector | Block |
| PII Leakage | PIIDetector + Redactor | Redact/Block |
| Data Exfiltration | OutputValidator | Block |
| Malicious Tools | ToolValidator | Block/Require approval |

---

## Phase 2: Advanced Features (Future)

- **Circuit Breaker**: Per-server circuit breaker (gobreaker library)
- **Credential Rotation**: API key rotation with grace period
- **Audit Log Retention**: Configurable retention with S3 archival
- **Email Integration**: SMTP configuration, password reset flow

---

## Implementation Sequence

### Sprint 1: Foundation (Security + Helm)

| Task | Track | Files |
|------|-------|-------|
| LDAP/AD authentication (configurable) | A | `internal/service/auth/ldap.go`, `provider.go` |
| HTTP security headers | A | `internal/handler/middleware/security.go` |
| Helm chart base | D | `deploy/helm/waffles/*` |

> **Note**: Rate limiting via AWS API Gateway/WAF - infrastructure config only

### Sprint 2: User Management + Metrics Setup

| Task | Track | Files |
|------|-------|-------|
| Admin user API (CRUD) | B | `internal/handler/admin/users.go` |
| Role management API | B | `internal/service/role/service.go` |
| Wire ServerHealthCollector | C | `cmd/server/main.go` |
| Prometheus + Grafana configs | C | `monitoring/prometheus/*`, `monitoring/grafana/*` |

### Sprint 3: Frontend + Server Trust Foundation

| Task | Track | Files |
|------|-------|-------|
| Admin Users Vue page | B | `web-app/src/views/AdminUsers.vue` |
| Admin Roles Vue page | B | `web-app/src/views/AdminRoles.vue` |
| SSRF protection | A/E | `internal/service/registry/validation.go` |

### Sprint 4: API Key Scoping

| Task | Track | Files |
|------|-------|-------|
| API key scopes migration | F | `internal/database/migrations/000009_apikey_scopes.up.sql` |
| Scope enforcement middleware | F | `internal/handler/middleware/scope.go` |
| API key management UI | F | `web-app/src/views/ApiKeys.vue` |

### Sprint 5: Guardrail Engine Core

| Task | Track | Files |
|------|-------|-------|
| Guardrail policy schema | G | `internal/database/migrations/000010_guardrails.up.sql` |
| Guardrail engine | G | `internal/service/guardrails/engine.go` |
| Prompt injection detector | G | `internal/service/guardrails/detectors/injection.go` |
| PII detector & redactor | G | `internal/service/guardrails/detectors/pii.go` |

### Sprint 6: Guardrails Admin + Polish

| Task | Track | Files |
|------|-------|-------|
| Guardrail middleware | G | `internal/handler/middleware/guardrails.go` |
| Guardrail admin UI | G | `web-app/src/views/AdminGuardrails.vue` |

### Sprint 7: K8s Operator

| Task | Track | Files |
|------|-------|-------|
| MCPServer CRD | D | `deploy/operator/api/v1/mcpserver_types.go` |
| Operator controller | D | `deploy/operator/controllers/mcpserver_controller.go` |
| Client auto-config endpoint | D | `internal/handler/clientconfig.go` |

---

## Key Files Reference

**Existing Security (to extend):**

- `internal/handler/middleware/auth.go` - Authentication middleware
- `internal/handler/middleware/authz.go` - Authorization
- `internal/service/authz/casbin.go` - RBAC policies
- `internal/server/router.go` - Middleware chain setup

**Existing User Management:**

- `internal/domain/user.go` - User model
- `internal/repository/user.go` - User repository
- `internal/handler/auth.go` - Auth handlers

**Existing Metrics:**

- `internal/metrics/prometheus.go` - Metrics registry
- `internal/metrics/collectors.go` - Collectors

**Gateway:**

- `internal/service/gateway/service.go` - Proxy logic
- `internal/domain/server.go` - Server model

---

## Dependencies to Add

```go
// go.mod additions
github.com/go-ldap/ldap/v3             // LDAP authentication
github.com/sony/gobreaker              // Circuit breaker (Phase 2)
// Note: No Redis needed - rate limiting via AWS managed services
```

```json
// package.json additions (web-app)
"chart.js": "^4.x"                     // For metrics charts
"vue-chartjs": "^5.x"                  // Vue Chart.js wrapper
```

---

## Container Security Hardening

**Dockerfile improvements:**

```dockerfile
USER 65532:65532                       # Non-root
COPY --chmod=555 server /app/server   # Read-only binary
```

**Kubernetes SecurityContext:**

```yaml
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
```
