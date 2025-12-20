# Keycloak OAuth Setup for MCP Gateway

This directory contains everything needed to run Keycloak as an OAuth/SSO provider for MCP Gateway.

## Features

- **UI SSO**: Single sign-on for the web interface
- **OAuth 2.1 with DCR**: Dynamic Client Registration for MCP clients like Claude Code
- **Pre-configured realm**: Ready-to-use "waffles" realm with test users

## Quick Start

1. **Start Keycloak:**
   ```bash
   cd examples/oauth/keycloak
   docker-compose up -d
   ```

2. **Wait for Keycloak to start** (check logs):
   ```bash
   docker-compose logs -f keycloak
   ```

3. **Configure DCR** (first time only):
   ```bash
   ./configure-dcr.sh
   ```
   This enables Dynamic Client Registration for MCP clients.

4. **Configure MCP Gateway** (in `.env`):
   ```bash
   AUTH_OAUTH_ENABLED=true
   AUTH_OAUTH_ISSUER=http://localhost:8180/realms/waffles
   AUTH_OAUTH_CLIENT_ID=waffles
   AUTH_OAUTH_CLIENT_SECRET=waffles-secret
   AUTH_OAUTH_DEFAULT_ROLE=viewer
   AUTH_OAUTH_AUTO_CREATE_USERS=true
   ```

5. **Restart MCP Gateway:**
   ```bash
   docker-compose restart gateway
   ```

## Test Users

| Username | Password | Role |
|----------|----------|------|
| testuser | testuser123 | viewer |
| testadmin | testadmin123 | admin |

## Admin Access

- **Admin Console**: http://localhost:8180
- **Credentials**: admin / admin

## Using with Claude Code

Once OAuth is enabled, Claude Code can authenticate via browser:

```bash
# No auth headers needed - Claude Code will open browser for OAuth
claude mcp add myserver http://localhost:8080/api/v1/gateway/<server-id> --transport http
```

Claude Code will:
1. Discover OAuth metadata at `/.well-known/oauth-protected-resource`
2. Register a client via DCR
3. Open browser for authentication
4. Exchange tokens and make authenticated requests

## Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Keycloak + PostgreSQL services |
| `waffles-realm.json` | Pre-configured realm with clients and users |
| `configure-dcr.sh` | Script to enable DCR for MCP clients |

## Troubleshooting

### Claude Code can't authenticate

1. Verify OAuth is enabled:
   ```bash
   curl http://localhost:8080/api/v1/status
   ```
   Look for `"sso": { "enabled": true }`.

2. Check protected resource metadata:
   ```bash
   curl http://localhost:8080/.well-known/oauth-protected-resource
   ```

3. Verify Keycloak is running:
   ```bash
   curl http://localhost:8180/realms/waffles/.well-known/openid-configuration
   ```

### Reset Keycloak

To start fresh:
```bash
docker-compose down -v
docker-compose up -d
./configure-dcr.sh
```
