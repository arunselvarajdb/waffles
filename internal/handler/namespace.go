package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/repository"
	"github.com/waffles/waffles/pkg/logger"
)

// NamespaceHandler handles namespace API requests
type NamespaceHandler struct {
	namespaceRepo NamespaceRepoInterface
	logger        logger.Logger
}

// NewNamespaceHandler creates a new namespace handler
func NewNamespaceHandler(namespaceRepo *repository.NamespaceRepository, log logger.Logger) *NamespaceHandler {
	var repo NamespaceRepoInterface
	if namespaceRepo != nil {
		repo = namespaceRepo
	}

	return &NamespaceHandler{
		namespaceRepo: repo,
		logger:        log,
	}
}

// NewNamespaceHandlerWithInterface creates a new namespace handler with interface (for testing).
func NewNamespaceHandlerWithInterface(namespaceRepo NamespaceRepoInterface, log logger.Logger) *NamespaceHandler {
	return &NamespaceHandler{
		namespaceRepo: namespaceRepo,
		logger:        log,
	}
}

// ListNamespaces returns all namespaces
// GET /api/v1/namespaces
func (h *NamespaceHandler) ListNamespaces(c *gin.Context) {
	namespaces, err := h.namespaceRepo.List(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list namespaces")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list namespaces"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"namespaces": namespaces,
		"count":      len(namespaces),
	})
}

// CreateNamespace creates a new namespace
// POST /api/v1/namespaces
func (h *NamespaceHandler) CreateNamespace(c *gin.Context) {
	var req domain.NamespaceCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ns, err := h.namespaceRepo.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create namespace"})
		return
	}

	c.JSON(http.StatusCreated, ns)
}

// GetNamespace returns a single namespace
// GET /api/v1/namespaces/:id
func (h *NamespaceHandler) GetNamespace(c *gin.Context) {
	id := c.Param("id")

	ns, err := h.namespaceRepo.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		h.logger.Error().Err(err).Str("id", id).Msg("Failed to get namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get namespace"})
		return
	}

	c.JSON(http.StatusOK, ns)
}

// UpdateNamespace updates a namespace
// PUT /api/v1/namespaces/:id
func (h *NamespaceHandler) UpdateNamespace(c *gin.Context) {
	id := c.Param("id")

	var req domain.NamespaceUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ns, err := h.namespaceRepo.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		h.logger.Error().Err(err).Str("id", id).Msg("Failed to update namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update namespace"})
		return
	}

	c.JSON(http.StatusOK, ns)
}

// DeleteNamespace deletes a namespace
// DELETE /api/v1/namespaces/:id
func (h *NamespaceHandler) DeleteNamespace(c *gin.Context) {
	id := c.Param("id")

	if err := h.namespaceRepo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		h.logger.Error().Err(err).Str("id", id).Msg("Failed to delete namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Namespace deleted"})
}

// AddServer adds a server to a namespace
// POST /api/v1/namespaces/:id/servers
func (h *NamespaceHandler) AddServer(c *gin.Context) {
	namespaceID := c.Param("id")

	var req domain.AddServerToNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify namespace exists
	_, err := h.namespaceRepo.Get(c.Request.Context(), namespaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		h.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to get namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify namespace"})
		return
	}

	if err := h.namespaceRepo.AddServerToNamespace(c.Request.Context(), req.ServerID, namespaceID); err != nil {
		h.logger.Error().Err(err).
			Str("server_id", req.ServerID).
			Str("namespace_id", namespaceID).
			Msg("Failed to add server to namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add server to namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server added to namespace"})
}

// RemoveServer removes a server from a namespace
// DELETE /api/v1/namespaces/:id/servers/:server_id
func (h *NamespaceHandler) RemoveServer(c *gin.Context) {
	namespaceID := c.Param("id")
	serverID := c.Param("server_id")

	if err := h.namespaceRepo.RemoveServerFromNamespace(c.Request.Context(), serverID, namespaceID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server not found in namespace"})
			return
		}
		h.logger.Error().Err(err).
			Str("server_id", serverID).
			Str("namespace_id", namespaceID).
			Msg("Failed to remove server from namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove server from namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server removed from namespace"})
}

// ListServers lists all servers in a namespace
// GET /api/v1/namespaces/:id/servers
func (h *NamespaceHandler) ListServers(c *gin.Context) {
	namespaceID := c.Param("id")

	members, err := h.namespaceRepo.GetNamespaceServers(c.Request.Context(), namespaceID)
	if err != nil {
		h.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to list namespace servers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list namespace servers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": members,
		"count":   len(members),
	})
}

// SetRoleAccess sets a role's access level to a namespace
// POST /api/v1/namespaces/:id/access
func (h *NamespaceHandler) SetRoleAccess(c *gin.Context) {
	namespaceID := c.Param("id")

	var req domain.SetRoleAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate access level
	if !req.AccessLevel.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid access level. Must be 'view' or 'execute'"})
		return
	}

	// Get role ID from role name
	roleID, err := h.namespaceRepo.GetRoleIDByName(c.Request.Context(), req.RoleName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role not found"})
			return
		}
		h.logger.Error().Err(err).Str("role_name", req.RoleName).Msg("Failed to get role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role"})
		return
	}

	// Verify namespace exists
	_, err = h.namespaceRepo.Get(c.Request.Context(), namespaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
			return
		}
		h.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to get namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify namespace"})
		return
	}

	if err := h.namespaceRepo.SetRoleNamespaceAccess(c.Request.Context(), roleID, namespaceID, req.AccessLevel); err != nil {
		h.logger.Error().Err(err).
			Str("role_id", roleID).
			Str("namespace_id", namespaceID).
			Str("access_level", string(req.AccessLevel)).
			Msg("Failed to set role access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set role access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Role access set",
		"role":         req.RoleName,
		"access_level": req.AccessLevel,
	})
}

// RemoveRoleAccess removes a role's access to a namespace
// DELETE /api/v1/namespaces/:id/access/:role_id
func (h *NamespaceHandler) RemoveRoleAccess(c *gin.Context) {
	namespaceID := c.Param("id")
	roleID := c.Param("role_id")

	if err := h.namespaceRepo.RemoveRoleNamespaceAccess(c.Request.Context(), roleID, namespaceID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role access not found"})
			return
		}
		h.logger.Error().Err(err).
			Str("role_id", roleID).
			Str("namespace_id", namespaceID).
			Msg("Failed to remove role access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove role access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role access removed"})
}

// ListRoleAccess lists all role access entries for a namespace
// GET /api/v1/namespaces/:id/access
func (h *NamespaceHandler) ListRoleAccess(c *gin.Context) {
	namespaceID := c.Param("id")

	accesses, err := h.namespaceRepo.GetNamespaceRoleAccess(c.Request.Context(), namespaceID)
	if err != nil {
		h.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to list role access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list role access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_entries": accesses,
		"count":          len(accesses),
	})
}

// Legacy type alias for backwards compatibility
type ServerGroupHandler = NamespaceHandler

// Legacy constructor alias
var NewServerGroupHandler = NewNamespaceHandler

// Legacy method aliases - these wrap the new methods for backwards compatibility
func (h *NamespaceHandler) ListGroups(c *gin.Context) {
	h.ListNamespaces(c)
}

func (h *NamespaceHandler) CreateGroup(c *gin.Context) {
	h.CreateNamespace(c)
}

func (h *NamespaceHandler) GetGroup(c *gin.Context) {
	h.GetNamespace(c)
}

func (h *NamespaceHandler) UpdateGroup(c *gin.Context) {
	h.UpdateNamespace(c)
}

func (h *NamespaceHandler) DeleteGroup(c *gin.Context) {
	h.DeleteNamespace(c)
}
