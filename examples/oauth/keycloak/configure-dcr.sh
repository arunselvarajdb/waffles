#!/bin/bash
# Configure Keycloak DCR policies for MCP clients (Claude Code)
#
# This script configures Keycloak's Dynamic Client Registration (DCR) policies
# to allow MCP clients like Claude Code to authenticate via OAuth 2.1.
#
# Run this after a fresh Keycloak deployment:
#   ./configure-dcr.sh
#
# Requirements:
#   - Keycloak running at localhost:8180
#   - Admin credentials: admin/admin

set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8180}"
KEYCLOAK_ADMIN="${KEYCLOAK_ADMIN:-admin}"
KEYCLOAK_ADMIN_PASSWORD="${KEYCLOAK_ADMIN_PASSWORD:-admin}"
REALM="${REALM:-waffles}"

echo "Configuring Keycloak DCR for realm: $REALM"

# Get admin token
echo "Getting admin token..."
TOKEN=$(curl -s -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$KEYCLOAK_ADMIN" \
  -d "password=$KEYCLOAK_ADMIN_PASSWORD" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to get admin token. Is Keycloak running?"
  exit 1
fi

echo "Admin token obtained"

# Get existing client registration policies
echo "Getting client registration policies..."
POLICIES=$(curl -s "$KEYCLOAK_URL/admin/realms/$REALM/components?type=org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy" \
  -H "Authorization: Bearer $TOKEN")

# 1. Update Trusted Hosts policy to include localhost
echo "Configuring Trusted Hosts policy..."
TRUSTED_HOSTS_ID=$(echo "$POLICIES" | grep -o '"id":"[^"]*","name":"Trusted Hosts"' | head -1 | cut -d'"' -f4)
if [ -n "$TRUSTED_HOSTS_ID" ]; then
  curl -s -X PUT "$KEYCLOAK_URL/admin/realms/$REALM/components/$TRUSTED_HOSTS_ID" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "id": "'"$TRUSTED_HOSTS_ID"'",
      "name": "Trusted Hosts",
      "providerId": "trusted-hosts",
      "providerType": "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
      "parentId": "'"$REALM"'",
      "subType": "anonymous",
      "config": {
        "host-sending-registration-request-must-match": ["false"],
        "client-uris-must-match": ["false"],
        "trusted-hosts": ["localhost", "127.0.0.1"]
      }
    }' > /dev/null
  echo "  Trusted Hosts policy updated"
else
  echo "  Warning: Trusted Hosts policy not found"
fi

# 2. Update Allowed Client Scopes policy
echo "Configuring Allowed Client Scopes policy..."
ALLOWED_SCOPES_ID=$(echo "$POLICIES" | grep -o '"id":"[^"]*","name":"Allowed Client Scopes"' | head -1 | cut -d'"' -f4)
if [ -n "$ALLOWED_SCOPES_ID" ]; then
  curl -s -X PUT "$KEYCLOAK_URL/admin/realms/$REALM/components/$ALLOWED_SCOPES_ID" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "id": "'"$ALLOWED_SCOPES_ID"'",
      "name": "Allowed Client Scopes",
      "providerId": "allowed-client-templates",
      "providerType": "org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy",
      "parentId": "'"$REALM"'",
      "subType": "anonymous",
      "config": {
        "allow-default-scopes": ["true"],
        "allowed-client-scopes": ["openid", "email", "profile", "address", "phone", "roles", "web-origins", "acr", "microprofile-jwt"]
      }
    }' > /dev/null
  echo "  Allowed Client Scopes policy updated"
else
  echo "  Warning: Allowed Client Scopes policy not found"
fi

# 3. Remove Consent Required policy (allows public clients without consent)
echo "Removing Consent Required policy..."
CONSENT_ID=$(echo "$POLICIES" | grep -o '"id":"[^"]*","name":"Consent Required"' | head -1 | cut -d'"' -f4)
if [ -n "$CONSENT_ID" ]; then
  curl -s -X DELETE "$KEYCLOAK_URL/admin/realms/$REALM/components/$CONSENT_ID" \
    -H "Authorization: Bearer $TOKEN" > /dev/null
  echo "  Consent Required policy removed"
else
  echo "  Consent Required policy already removed"
fi

# 4. Delete offline_access scope (Claude Code registers as public client which can't use offline tokens)
echo "Removing offline_access scope..."
OFFLINE_SCOPE_ID=$(curl -s "$KEYCLOAK_URL/admin/realms/$REALM/client-scopes" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"id":"[^"]*","name":"offline_access"' | head -1 | cut -d'"' -f4)
if [ -n "$OFFLINE_SCOPE_ID" ]; then
  curl -s -X DELETE "$KEYCLOAK_URL/admin/realms/$REALM/client-scopes/$OFFLINE_SCOPE_ID" \
    -H "Authorization: Bearer $TOKEN" > /dev/null
  echo "  offline_access scope deleted"
else
  echo "  offline_access scope already removed"
fi

echo ""
echo "DCR configuration complete!"
echo ""
echo "You can now use Claude Code with OAuth:"
echo "  claude mcp add myserver http://localhost:8080/api/v1/gateway/<server-id> --transport http"
