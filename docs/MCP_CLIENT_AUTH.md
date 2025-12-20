# MCP Client Authentication

This guide explains how to authenticate MCP clients (like Claude Code) with MCP Gateway.

## Overview

MCP Gateway supports multiple authentication methods for MCP clients:

1. **API Keys** (Default, recommended for scripts and automation)
2. **Session Cookies** (For browser-based MCP clients sharing UI session)
3. **OAuth 2.1 with DCR** (For clients like Claude Code that support OAuth)

## Authentication Methods

### 1. API Key Authentication

The simplest way to authenticate MCP clients. API keys are managed through the Gateway UI.

#### Creating an API Key

1. Log in to the Gateway UI at `http://localhost:8080`
2. Go to **Settings > API Keys**
3. Click **Create New API Key**
4. Copy the generated key (it will only be shown once)

#### Using API Keys with MCP Clients

```bash
# Claude Code
claude mcp add myserver http://localhost:8080/api/v1/gateway/<server-id> \
  --transport http \
  --header "Authorization: Bearer mcpgw_xxxx"

# Or using X-API-Key header
claude mcp add myserver http://localhost:8080/api/v1/gateway/<server-id> \
  --transport http \
  --header "X-API-Key: mcpgw_xxxx"
```

### 2. OAuth 2.1 with Dynamic Client Registration

Some MCP clients (like Claude Code) support OAuth 2.1 with Dynamic Client Registration (DCR). This provides seamless authentication through a browser flow.

#### Requirements

- An identity provider that supports DCR (e.g., Keycloak)
- `auth.oauth.enabled: true` in Gateway config

#### DCR-Capable Identity Providers

| Provider | DCR Support | Notes |
|----------|-------------|-------|
| Keycloak | Native | Recommended - full DCR support |
| Zitadel | Native | Cloud-native alternative |
| Ory Hydra | Native | Requires separate login app |
| Okta | Limited | Enterprise plan may be required |
| Auth0 | No | DCR not supported |
| Authentik | No | DCR not supported |

## Setup with Keycloak (Recommended)

### Quick Start

1. Start Keycloak:
   ```bash
   cd examples/oauth/keycloak
   docker-compose up -d
   ```

2. Wait for Keycloak to start (check logs):
   ```bash
   docker-compose logs -f keycloak
   ```

3. **First time only** - Configure DCR policies for MCP clients:
   ```bash
   ./configure-dcr.sh
   ```
   This configures Keycloak to allow Claude Code and other MCP clients to register dynamically.

4. Configure Gateway OAuth settings in `.env`:
   ```bash
   AUTH_OAUTH_ENABLED=true
   AUTH_OAUTH_ISSUER=http://localhost:8180/realms/waffles
   AUTH_OAUTH_CLIENT_ID=waffles
   AUTH_OAUTH_CLIENT_SECRET=waffles-secret
   AUTH_OAUTH_DEFAULT_ROLE=viewer
   AUTH_OAUTH_AUTO_CREATE_USERS=true
   ```

5. Restart the Gateway:
   ```bash
   docker-compose restart gateway
   ```

### Test Users

The Keycloak realm includes test users:

| Username | Password | Role |
|----------|----------|------|
| testuser | testuser123 | viewer |
| testadmin | testadmin123 | admin |

### Using with Claude Code

Once OAuth is enabled, Claude Code can authenticate via browser:

```bash
# No auth headers needed - Claude Code will open browser for OAuth flow
claude mcp add myserver http://localhost:8080/api/v1/gateway/<server-id> \
  --transport http
```

Claude Code will:
1. Discover OAuth metadata at `/.well-known/oauth-protected-resource`
2. Use DCR to register a client with Keycloak
3. Open browser for user authentication
4. Exchange tokens and make authenticated requests

## Configuration Reference

### MCP Auth Config

Controls which authentication methods are accepted for MCP clients:

```yaml
auth:
  mcp_auth:
    api_key: true    # Accept API keys (mcpgw_xxx prefix)
    session: true    # Accept session cookies from browser
    oauth: true      # Enable OAuth/DCR for MCP clients
                     # Set to false to require API keys even when UI SSO is enabled
```

**UI-Only SSO Mode**: To enable SSO for the web UI while requiring API keys for MCP clients:
```yaml
auth:
  oauth:
    enabled: true    # UI SSO enabled
  mcp_auth:
    api_key: true    # MCP clients must use API keys
    oauth: false     # Disable OAuth for MCP clients
```

### OAuth Config

Configures SSO/OIDC for both UI and MCP clients:

```yaml
auth:
  oauth:
    enabled: false              # Enable OAuth/SSO
    base_url: "http://localhost:8080"  # Gateway URL for callbacks
    issuer: ""                  # OIDC issuer URL
    client_id: ""               # OAuth client ID
    client_secret: ""           # OAuth client secret
    scopes:                     # OIDC scopes to request
      - openid
      - email
      - profile
    default_role: "viewer"      # Role for new OAuth users
    auto_create_users: true     # Create user on first login
    allowed_domains: []         # Restrict to email domains
```

### Environment Variables

All config options can be set via environment variables:

```bash
# MCP Client Auth
AUTH_MCP_AUTH_API_KEY=true
AUTH_MCP_AUTH_SESSION=true
AUTH_MCP_AUTH_OAUTH=true      # Set to false for UI-only SSO

# OAuth/SSO
AUTH_OAUTH_ENABLED=true
AUTH_OAUTH_BASE_URL=http://localhost:8080
AUTH_OAUTH_ISSUER=http://localhost:8180/realms/waffles
AUTH_OAUTH_CLIENT_ID=waffles
AUTH_OAUTH_CLIENT_SECRET=waffles-secret
AUTH_OAUTH_DEFAULT_ROLE=viewer
AUTH_OAUTH_AUTO_CREATE_USERS=true
```

## OAuth Discovery Endpoints

MCP Gateway exposes OAuth metadata endpoints per RFC 9728:

- `/.well-known/oauth-protected-resource` - Protected Resource Metadata
- `/.well-known/oauth-authorization-server` - Proxied Authorization Server Metadata

These endpoints allow MCP clients to discover authentication requirements automatically.

## Troubleshooting

### Claude Code Can't Authenticate

1. Verify OAuth is enabled:
   ```bash
   curl http://localhost:8080/api/v1/status
   ```

   Look for `"sso": { "enabled": true }` in the response.

2. Check the protected resource metadata:
   ```bash
   curl http://localhost:8080/.well-known/oauth-protected-resource
   ```

3. Verify Keycloak is running:
   ```bash
   curl http://localhost:8180/realms/waffles/.well-known/openid-configuration
   ```

### API Key Not Working

1. Ensure the key starts with `mcpgw_` prefix
2. Check if API key auth is enabled: `auth.mcp_auth.api_key: true`
3. Verify the key hasn't expired

### Session Cookie Not Working

1. Ensure session auth is enabled: `auth.mcp_auth.session: true`
2. Check if you're logged in to the Gateway UI
3. Verify cookies are being sent with requests

## Security Considerations

- **API Keys**: Store securely, rotate regularly, use scoped keys when possible
- **OAuth**: Use HTTPS in production, verify redirect URIs
- **Session Cookies**: Protected by HttpOnly, Secure, and SameSite flags

## Architecture

```
+----------------+     +-----------------+     +-------------+
|  Claude Code   |     |   MCP Gateway   |     |  Keycloak   |
|  (MCP Client)  |---->|  (Resource Svr) |<--->| (Auth Svr)  |
+----------------+     +--------+--------+     +-------------+
                               |
                               v
                       +---------------+
                       |  Gateway DB   |
                       | - Users       |
                       | - Roles       |
                       | - Namespaces  |
                       +---------------+
```

**Key Points:**
- Keycloak handles authentication (who you are)
- Gateway DB handles authorization (what you can do)
- Users are synced from Keycloak to Gateway DB on first login
- Roles and namespace access are managed in Gateway
