# MCP Gateway - Web UI Mockup

This is a **static HTML mockup** of the MCP Gateway admin interface. It serves as a visual prototype to demonstrate the UI/UX for managing MCP servers. This mockup is non-functional but shows all the key features and workflows.

## 🎯 Purpose

- **Visual Demonstration**: Show stakeholders what the complete UI will look like
- **User Experience Testing**: Validate UI/UX decisions before implementation
- **Design Reference**: Serve as a blueprint for frontend development
- **Quick Prototyping**: Rapidly iterate on design without backend integration

## 🚀 How to View the Mockup

Simply open the `index.html` file in any modern web browser:

```bash
# From the web directory
open index.html

# Or if you prefer a local server (optional)
python3 -m http.server 8000
# Then visit: http://localhost:8000
```

No build process, dependencies, or server required - just open the HTML file!

## ✨ Features Demonstrated

## 🔐 Access Control & User Roles

The mockup demonstrates a complete **Role-Based Access Control (RBAC)** system with two distinct user roles:

### **Admin Role** (`index.html`, `users.html`)
**Full Access - Complete Gateway Management**

Capabilities:
- ✅ **Server Management**: Full CRUD operations on MCP servers
  - Add new servers with all auth types (none, basic, bearer, oauth)
  - Edit existing server configurations
  - Delete servers
  - Toggle active/inactive status
- ✅ **User Management**: Complete user administration
  - Create new users (admin or viewer roles)
  - Edit user details and permissions
  - Assign/unassign servers to users
  - Activate/deactivate user accounts
  - Delete users
- ✅ **Server Assignment**: Control which users can access which servers
- ✅ **View All Metrics**: Access to all system metrics and health data

### **Viewer Role** (`user-dashboard.html`)
**Limited Access - Assigned Servers Only**

Capabilities:
- ✅ **View Assigned Servers**: See only servers they've been granted access to
- ✅ **Configure Tools**: Select which MCP tools to use for each assigned server
- ✅ **Monitor Health**: View health status of assigned servers
- ❌ **No CRUD Operations**: Cannot add, edit, or delete servers
- ❌ **No User Management**: Cannot manage users or permissions

### Access Control Flow

```
Login (login.html)
    │
    ├─> Admin Login ──> Admin Dashboard (index.html)
    │                      │
    │                      ├─> Servers Page (CRUD operations)
    │                      ├─> Users Page (users.html)
    │                      │      └─> Assign Servers to Users
    │                      └─> Metrics (all servers)
    │
    └─> Viewer Login ──> User Dashboard (user-dashboard.html)
                           │
                           ├─> View Assigned Servers (read-only)
                           └─> Configure Tools (for assigned servers)
```

### Pages Overview

| Page | File | Role | Description |
|------|------|------|-------------|
| **Login** | `login.html` | Both | Authentication page with demo login buttons |
| **Admin Dashboard** | `index.html` | Admin | Server management with full CRUD |
| **User Management** | `users.html` | Admin | Manage users and assign server permissions |
| **User Dashboard** | `user-dashboard.html` | Viewer | View assigned servers and configure tools |

### Authentication Method

**SSO/OAuth Only** - No traditional username/password login

The mockup uses **enterprise-grade SSO/OAuth authentication** with support for:
- **Google OAuth 2.0** - Google Workspace accounts
- **Microsoft Azure AD** - Microsoft 365 accounts
- **GitHub OAuth** - GitHub organization accounts
- **Okta** - Okta SSO
- **SAML 2.0** - Generic SAML SSO for any provider

**How it works**:
1. User clicks SSO provider button
2. Redirects to OAuth provider for authentication
3. User authenticates with organization credentials
4. OAuth callback returns user info + role mapping
5. JWT token generated with role (admin or viewer)
6. User redirected to appropriate dashboard

**Role Determination**:
- Organization's SSO attributes determine user role
- Mapping can be based on: groups, email domains, custom claims
- Example: `security-team@company.com` → Admin role
- Example: `developers@company.com` → Viewer role

### Demo Login (Mockup)

From `login.html`, use these demo buttons to simulate SSO:
- **Demo: Login as Admin** → `index.html` (Full access)
- **Demo: Login as Viewer** → `user-dashboard.html` (Limited access)

### 1. **Admin: Server Management Dashboard**
- Statistics cards showing:
  - Total servers count
  - Active servers count
  - Healthy servers count
  - Recent request activity
- Quick action button to add new servers

### 2. **Server Management**
- **Server List Table** with columns:
  - Visual status indicator (green/yellow/red dot)
  - Server name and description
  - Server URL
  - Protocol version
  - Authentication type badge
  - Health status badge with last checked time
  - Active/Inactive toggle switch
  - Action buttons (Edit, Delete)

- **Search & Filtering**:
  - Search by server name
  - Filter by status (All, Active, Inactive)
  - Filter by health (All, Healthy, Degraded, Unhealthy)
  - Clear filters button

- **Pagination**:
  - Page navigation (Previous, 1, 2, 3, Next)
  - Shows current page and total results
  - Responsive design (different views for mobile/desktop)

### 3. **Add Server Modal**
Comprehensive form with all fields:
- Server name (required)
- Description (textarea)
- Server URL (with validation hint)
- Protocol version (dropdown: 1.0.0, 0.9.0, 0.8.0)
- Authentication type (radio buttons):
  - None
  - Basic Auth (shows username/password fields)
  - Bearer Token (shows token field)
- Health check interval (seconds)
- Timeout (seconds)
- Max connections
- Tags (comma-separated)
- Active checkbox (default: checked)

**Interactive Features**:
- Conditional auth fields (show/hide based on auth type selection)
- Form validation hints
- Save and Cancel buttons

### 4. **Edit Server Modal**
- Same fields as Add Server
- Pre-populated with existing server data
- Additional "Delete Server" button (danger style)
- Update and Cancel buttons

### 5. **Delete Confirmation Modal**
- Warning icon
- Confirmation message with server name
- Delete (danger) and Cancel buttons
- Prevents accidental deletions

### 6. **Responsive Design**
- Mobile-friendly layout
- Collapsible navigation on small screens
- Stacked table on mobile
- Full-width modals on small screens

## 🎨 Design System

### Colors (Tailwind CSS)
- **Primary**: Blue (`#2563eb`) - Actions, links
- **Success**: Green (`#10b981`) - Healthy status, active states
- **Warning**: Yellow (`#f59e0b`) - Degraded status
- **Danger**: Red (`#dc2626`) - Errors, delete actions
- **Neutral**: Gray shades for backgrounds and text

### Components
- **Buttons**: Primary, Secondary, Danger styles
- **Forms**: Text inputs, textareas, selects, radio buttons, checkboxes
- **Tables**: Striped rows, hover states, responsive
- **Modals**: Centered with backdrop overlay
- **Badges**: Status indicators for auth type and health
- **Toggle Switches**: For active/inactive state

## 🔧 Technical Details

### Technologies Used
- **HTML5**: Semantic markup
- **Tailwind CSS**: Utility-first CSS framework (via CDN)
- **Vanilla JavaScript**: Modal interactions, form handling

### File Structure
```
web/
├── login.html                  # Login page with role selection
├── index.html                  # Admin dashboard (server management)
├── users.html                  # Admin user management page
├── user-dashboard.html         # Viewer dashboard (assigned servers)
├── README.md                   # This documentation
└── assets/
    ├── js/
    │   └── mockup.js           # Modal and interaction logic
    └── css/
        └── (empty - Tailwind CDN used)
```

### MCP Authentication Standards

The mockup follows **MCP Protocol authentication standards** with 4 supported auth types:

#### 1. **None** (No Authentication)
```json
{
  "auth_type": "none",
  "auth_config": null
}
```
Use for: Public MCP servers, internal trusted networks

#### 2. **Basic Auth** (HTTP Basic Authentication)
```json
{
  "auth_type": "basic",
  "auth_config": {
    "username": "admin",
    "password": "secret123"
  }
}
```
Use for: Simple username/password authentication

#### 3. **Bearer Token** (JWT or API Key)
```json
{
  "auth_type": "bearer",
  "auth_config": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```
Use for: JWT tokens, API keys, personal access tokens

#### 4. **OAuth 2.0** (Client Credentials Flow)
```json
{
  "auth_type": "oauth",
  "auth_config": {
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "token_url": "https://oauth.example.com/token",
    "scopes": "read write admin"
  }
}
```
Use for: OAuth 2.0 providers (Google, GitHub, etc.)

**Note**: OAuth implementation is partially complete in the backend. The UI mockup shows all 4 types for completeness.

### Sample Data
The mockup includes 5 example servers with varied states:
1. **Filesystem Server** - Healthy, Active, No Auth
2. **Database Tools** - Degraded, Active, Basic Auth
3. **API Gateway** - Unhealthy, Inactive, Bearer Token
4. **Slack Integration** - Healthy, Active, Bearer Token
5. **GitHub Integration** - Healthy, Active, Bearer Token

**Sample Users**:
- **Admin User** (admin@example.com) - Admin role, access to all servers
- **John Doe** (john.doe@example.com) - Viewer role, 3 assigned servers
- **Jane Smith** (jane.smith@example.com) - Viewer role, 0 assigned servers
- **Bob Johnson** (bob.johnson@example.com) - Viewer role, inactive, 1 server

## 🔌 API Endpoints (Reference)

This mockup visualizes the following backend API endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/servers` | List all servers (with filters, pagination) |
| POST | `/api/v1/servers` | Create a new server |
| GET | `/api/v1/servers/:id` | Get server details |
| PUT | `/api/v1/servers/:id` | Update server |
| DELETE | `/api/v1/servers/:id` | Delete server |
| PATCH | `/api/v1/servers/:id/toggle` | Toggle active/inactive |
| GET | `/api/v1/servers/:id/health` | Get health status |
| POST | `/api/v1/servers/:id/health` | Check health now |

## 🚧 Limitations (This is a Mockup!)

**What DOESN'T work**:
- ❌ No actual API calls
- ❌ Forms don't submit data (show mockup alerts)
- ❌ Data doesn't persist (refresh = reset)
- ❌ Search/filters are visual only
- ❌ Pagination doesn't change data
- ❌ Toggle switches don't save state
- ❌ No real authentication

**What DOES work**:
- ✅ Modal open/close interactions
- ✅ Auth field show/hide based on selection
- ✅ Responsive layout
- ✅ Hover states and visual feedback
- ✅ Form validation hints (visual)

## 🎯 Next Steps: Converting to a Functional App

To make this a real, functional application using Vue.js:

### 1. **Setup Vue 3 Project**
```bash
npm create vue@latest mcp-gateway-ui
cd mcp-gateway-ui
npm install
```

### 2. **Install Dependencies**
```bash
npm install axios                  # HTTP client
npm install pinia                  # State management
npm install vue-router             # Routing
npm install -D tailwindcss         # CSS framework
npx tailwindcss init
```

### 3. **Convert HTML to Vue Components**
- `App.vue` - Main layout
- `components/Dashboard.vue` - Dashboard stats
- `components/ServerList.vue` - Server table
- `components/ServerModal.vue` - Add/Edit modal
- `components/DeleteConfirmModal.vue` - Delete confirmation
- `components/SearchFilters.vue` - Search and filter bar
- `components/Pagination.vue` - Pagination controls

### 4. **Add State Management (Pinia)**
```javascript
// stores/servers.js
export const useServerStore = defineStore('servers', {
  state: () => ({
    servers: [],
    loading: false,
    filters: { status: 'all', health: 'all', search: '' }
  }),
  actions: {
    async fetchServers() { /* API call */ },
    async createServer(data) { /* API call */ },
    async updateServer(id, data) { /* API call */ },
    async deleteServer(id) { /* API call */ },
    async toggleServer(id) { /* API call */ }
  }
})
```

### 5. **Add API Integration**
```javascript
// services/api.js
import axios from 'axios'

const API_BASE = 'http://localhost:8080/api/v1'

export default {
  servers: {
    list: (params) => axios.get(`${API_BASE}/servers`, { params }),
    get: (id) => axios.get(`${API_BASE}/servers/${id}`),
    create: (data) => axios.post(`${API_BASE}/servers`, data),
    update: (id, data) => axios.put(`${API_BASE}/servers/${id}`, data),
    delete: (id) => axios.delete(`${API_BASE}/servers/${id}`),
    toggle: (id) => axios.patch(`${API_BASE}/servers/${id}/toggle`),
    health: (id) => axios.get(`${API_BASE}/servers/${id}/health`),
    checkHealth: (id) => axios.post(`${API_BASE}/servers/${id}/health`)
  }
}
```

### 6. **Add Form Validation**
```bash
npm install vee-validate yup
```

### 7. **Add Real-time Updates (Optional)**
- WebSocket connection for live updates
- Or polling every 30 seconds for health status

### 8. **Add Authentication (If Required)**
- JWT token management
- Protected routes
- Login/logout flow

## 📝 Notes for Developers

1. **Color Consistency**: All colors use Tailwind's color palette. Primary actions are blue, destructive actions are red.

2. **Accessibility**: When converting to Vue, add proper ARIA labels, focus management, and keyboard navigation.

3. **Error Handling**: The real app should include error states, loading spinners, and user-friendly error messages.

4. **Validation**: Add client-side validation for all forms (URL format, required fields, etc.)

5. **Optimistic Updates**: For better UX, update UI immediately and rollback on error.

6. **Confirmation Dialogs**: Keep delete confirmations and other destructive actions.

7. **Mobile Experience**: Test thoroughly on mobile devices, especially modals and tables.

## 📊 Mockup Statistics

- **Total HTML Lines**: ~800
- **JavaScript Lines**: ~70
- **Sample Servers**: 5
- **Modals**: 3 (Add, Edit, Delete)
- **Form Fields**: 11 in Add/Edit modal
- **Interactive Elements**: Buttons, toggles, radio buttons, checkboxes
- **Development Time**: ~6 hours (as planned)

## 🤝 Contributing

When adding new features to this mockup:
1. Keep the visual design consistent with existing components
2. Use Tailwind utility classes (avoid custom CSS)
3. Update this README with new features
4. Test on multiple screen sizes (mobile, tablet, desktop)
5. Ensure modals work with keyboard (Escape to close)

## 📜 License

Part of the MCP Gateway project.

---

**Created**: 2025-12-13
**Status**: Static Mockup (Non-functional)
**Ready for**: Vue.js conversion and API integration
