# SSO/OIDC Configuration

MCP Gateway supports Single Sign-On (SSO) via any OIDC-compliant identity provider. This guide covers setting up SSO with various providers.

## Supported Providers

The gateway works with any OIDC-compliant provider:
- **Authentik** - Self-hosted, open-source (recommended for internal tools)
- **Keycloak** - Self-hosted, enterprise features
- **Okta** - Cloud-hosted
- **Auth0** - Cloud-hosted
- **Azure AD** - Microsoft cloud
- **Google Workspace** - Google cloud

---

## Quick Start with Authentik

Authentik is recommended for self-hosted internal tools.

### 1. Install Authentik

Follow the official Authentik Docker Compose installation guide:

**https://docs.goauthentik.io/install-config/install/docker-compose/**

After installation, access the Authentik admin interface (default: http://localhost:9000).

### 2. Create OAuth Application in Authentik

1. Login to Authentik admin
2. Go to **Applications** > **Applications** > **Create**
3. Fill in:
   - **Name**: MCP Gateway
   - **Slug**: mcp-gateway
   - **Provider**: Create a new OAuth2/OIDC Provider
4. Configure the OAuth2 Provider:
   - **Name**: MCP Gateway Provider
   - **Authorization flow**: default-authentication-flow
   - **Client type**: Confidential
   - **Client ID**: (auto-generated, copy this)
   - **Client Secret**: (auto-generated, copy this)
   - **Redirect URIs**: `http://localhost:8080/api/v1/auth/sso/callback`
   - **Scopes**: openid, email, profile

### 3. Configure MCP Gateway

Update `configs/config.yaml`:

```yaml
auth:
  enabled: true
  oauth:
    enabled: true
    base_url: http://localhost:8080
    issuer: http://localhost:9000/application/o/mcp-gateway/
    client_id: <your-client-id>
    client_secret: <your-client-secret>
    default_role: viewer
    auto_create_users: true
    scopes:
      - openid
      - email
      - profile
```

### 4. Restart MCP Gateway

```bash
docker-compose restart gateway
```

The login page will now show a "Sign in with SSO" button.

---

## Configuration Reference

### config.yaml OAuth Settings

```yaml
auth:
  oauth:
    # Enable SSO login (works alongside local password auth)
    enabled: true

    # Your MCP Gateway's public URL (for OAuth callbacks)
    base_url: https://gateway.example.com

    # OIDC Issuer URL - provider's .well-known/openid-configuration base
    # Examples:
    #   Authentik:  https://auth.example.com/application/o/mcp-gateway/
    #   Keycloak:   https://auth.example.com/realms/myrealm
    #   Okta:       https://mycompany.okta.com/oauth2/default
    #   Auth0:      https://mycompany.auth0.com/
    #   Azure AD:   https://login.microsoftonline.com/{tenant-id}/v2.0
    #   Google:     https://accounts.google.com
    issuer: ""

    # OAuth client credentials from your provider
    client_id: ""
    client_secret: ""

    # Scopes to request (defaults shown)
    scopes:
      - openid
      - email
      - profile

    # Default role for new SSO users
    default_role: viewer

    # Create user records on first SSO login
    auto_create_users: true

    # Optional: Restrict to specific email domains
    allowed_domains:
      - example.com
      - company.org
```

### Environment Variables (Recommended for Secrets)

All settings can be overridden with environment variables. **Use environment variables for `client_id` and `client_secret` to avoid committing secrets to git.**

Create a `.env` file (copy from `.env.example`):

```bash
# .env (do not commit this file)
AUTH_OAUTH_ENABLED=true
AUTH_OAUTH_BASE_URL=http://localhost:8080
AUTH_OAUTH_ISSUER=http://localhost:9000/application/o/mcp-gateway/
AUTH_OAUTH_CLIENT_ID=your-client-id
AUTH_OAUTH_CLIENT_SECRET=your-client-secret
AUTH_OAUTH_DEFAULT_ROLE=viewer
AUTH_OAUTH_AUTO_CREATE_USERS=true
```

For Docker Compose, these are already configured as commented examples in `docker-compose.yml`. Uncomment and set the values, or use a `.env` file:

```bash
# Run with .env file
docker-compose --env-file .env up -d
```

---

## Provider-Specific Setup

### Authentik

**Installation**: https://docs.goauthentik.io/install-config/install/docker-compose/

**Issuer URL format**: `https://{authentik-domain}/application/o/{app-slug}/`

**Redirect URI**: `{base_url}/api/v1/auth/sso/callback`

### Keycloak

**Issuer URL format**: `https://{keycloak-domain}/realms/{realm}`

1. Create a new client in your realm
2. Set client type to "confidential"
3. Add redirect URI: `{base_url}/api/v1/auth/sso/callback`
4. Enable "Standard Flow" (Authorization Code)

### Okta

**Issuer URL format**: `https://{org}.okta.com/oauth2/default`

1. Create a new OIDC application
2. Set Sign-in redirect URI: `{base_url}/api/v1/auth/sso/callback`
3. Copy Client ID and Secret

### Azure AD

**Issuer URL format**: `https://login.microsoftonline.com/{tenant-id}/v2.0`

1. Register an application in Azure Portal
2. Add redirect URI under "Authentication"
3. Create a client secret under "Certificates & secrets"

### Google Workspace

**Issuer URL**: `https://accounts.google.com`

1. Create OAuth credentials in Google Cloud Console
2. Add authorized redirect URI: `{base_url}/api/v1/auth/sso/callback`
3. Enable "Google+ API" or "People API"

---

## How It Works

1. User clicks "Sign in with SSO" on the login page
2. Browser redirects to the identity provider
3. User authenticates with their provider credentials
4. Provider redirects back to `/api/v1/auth/sso/callback` with auth code
5. Gateway exchanges code for tokens and fetches user info
6. User is created (if `auto_create_users: true`) or matched by email
7. Session is created and user is logged in

---

## Security Considerations

1. **HTTPS Required**: In production, both the gateway and identity provider should use HTTPS
2. **Client Secret**: Keep `client_secret` secure; use environment variables in production
3. **Allowed Domains**: Use `allowed_domains` to restrict which email domains can authenticate
4. **Role Assignment**: New SSO users get `default_role`; upgrade roles manually in the admin UI
5. **Session Security**: Set `cookie_secure: true` in production for HTTPS-only cookies

---

## Troubleshooting

### SSO button not appearing
- Check that `auth.oauth.enabled: true` in config
- Verify the issuer URL is accessible (try fetching `{issuer}/.well-known/openid-configuration`)
- Check gateway logs for OIDC discovery errors

### Redirect URI mismatch
- Ensure `base_url` in config matches exactly what you registered with the provider
- The callback URL must be: `{base_url}/api/v1/auth/sso/callback`

### User created with wrong role
- Check `default_role` setting
- Users can be promoted via the admin interface

### "Email domain not allowed" error
- Check `allowed_domains` list
- If empty, all domains are allowed
