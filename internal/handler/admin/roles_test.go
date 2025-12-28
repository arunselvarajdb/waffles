package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waffles/waffles/internal/service/role"
	"github.com/waffles/waffles/pkg/logger"
)

func TestNewRolesHandler(t *testing.T) {
	svc := &role.Service{}
	log := logger.NewNop()

	handler := NewRolesHandler(svc, log)

	assert.NotNil(t, handler)
	assert.Equal(t, svc, handler.service)
}

func TestRolesHandler_GetRole_MissingID(t *testing.T) {
	router := setupTestRouter()
	handler := NewRolesHandler(&role.Service{}, logger.NewNop())

	// Register route with empty ID pattern (Gin won't match this)
	router.GET("/roles/:id", handler.GetRole)

	// Request with empty ID - Gin routing won't match, returns 404
	req, _ := http.NewRequest(http.MethodGet, "/roles/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// For empty path param, Gin returns 404 as the route doesn't match
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRolesHandler_CreateRole_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	handler := NewRolesHandler(&role.Service{}, logger.NewNop())

	router.POST("/roles", handler.CreateRole)

	// Send invalid JSON
	req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "Invalid request body")
}

func TestRolesHandler_UpdateRole_MissingID(t *testing.T) {
	router := setupTestRouter()
	handler := NewRolesHandler(&role.Service{}, logger.NewNop())

	router.PUT("/roles/:id", handler.UpdateRole)

	// Request with empty ID - returns 404 as route doesn't match
	req, _ := http.NewRequest(http.MethodPut, "/roles/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRolesHandler_UpdateRole_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	handler := NewRolesHandler(&role.Service{}, logger.NewNop())

	router.PUT("/roles/:id", handler.UpdateRole)

	req, _ := http.NewRequest(http.MethodPut, "/roles/role-1", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRolesHandler_DeleteRole_MissingID(t *testing.T) {
	router := setupTestRouter()
	handler := NewRolesHandler(&role.Service{}, logger.NewNop())

	router.DELETE("/roles/:id", handler.DeleteRole)

	// Request with empty ID - returns 404 as route doesn't match
	req, _ := http.NewRequest(http.MethodDelete, "/roles/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRoleCreateRequest_Structure(t *testing.T) {
	req := role.CreateRequest{
		Name:        "test-role",
		Description: "A test role",
		Permissions: []string{"perm-1", "perm-2"},
	}

	assert.Equal(t, "test-role", req.Name)
	assert.Equal(t, "A test role", req.Description)
	assert.Len(t, req.Permissions, 2)
}

func TestRoleUpdateRequest_Structure(t *testing.T) {
	desc := "Updated description"
	req := role.UpdateRequest{
		Description: &desc,
		Permissions: []string{"perm-1"},
	}

	assert.NotNil(t, req.Description)
	assert.Equal(t, "Updated description", *req.Description)
	assert.Len(t, req.Permissions, 1)
}
