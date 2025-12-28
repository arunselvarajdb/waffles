package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/pkg/logger"
)

func TestNewSessionsHandler(t *testing.T) {
	log := logger.NewNop()

	handler := NewSessionsHandler(log)

	assert.NotNil(t, handler)
}

func TestSessionInfo_Structure(t *testing.T) {
	session := SessionInfo{
		SessionID: "sess-123",
		UserID:    "user-456",
		CreatedAt: "2024-01-01T10:00:00Z",
		ExpiresAt: "2024-01-02T10:00:00Z",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		IsCurrent: true,
	}

	assert.Equal(t, "sess-123", session.SessionID)
	assert.Equal(t, "user-456", session.UserID)
	assert.Equal(t, "2024-01-01T10:00:00Z", session.CreatedAt)
	assert.Equal(t, "2024-01-02T10:00:00Z", session.ExpiresAt)
	assert.Equal(t, "192.168.1.1", session.IPAddress)
	assert.Equal(t, "Mozilla/5.0", session.UserAgent)
	assert.True(t, session.IsCurrent)
}

func TestSessionsHandler_ListUserSessions(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "list sessions with valid user ID",
			userID:         "user-123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				// Should return empty sessions (placeholder implementation)
				sessions, ok := resp["sessions"].([]interface{})
				require.True(t, ok)
				assert.Empty(t, sessions)
				// Should include message about placeholder
				assert.Contains(t, resp["message"], "not yet implemented")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			handler := NewSessionsHandler(logger.NewNop())

			router.GET("/users/:id/sessions", handler.ListUserSessions)

			req, _ := http.NewRequest(http.MethodGet, "/users/"+tt.userID+"/sessions", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestSessionsHandler_ListUserSessions_MissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewSessionsHandler(logger.NewNop())

	router.GET("/users/:id/sessions", handler.ListUserSessions)

	// Empty ID - Gin treats "//" as matching empty param, handler returns 400
	req, _ := http.NewRequest(http.MethodGet, "/users//sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Handler returns 400 for empty ID
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSessionsHandler_RevokeSession(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		sessionID      string
		expectedStatus int
	}{
		{
			name:           "revoke session with valid IDs",
			userID:         "user-123",
			sessionID:      "sess-456",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			handler := NewSessionsHandler(logger.NewNop())

			router.DELETE("/users/:id/sessions/:sid", handler.RevokeSession)

			req, _ := http.NewRequest(http.MethodDelete, "/users/"+tt.userID+"/sessions/"+tt.sessionID, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Contains(t, resp["message"], "not yet implemented")
		})
	}
}

func TestSessionsHandler_RevokeSession_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewSessionsHandler(logger.NewNop())

	router.DELETE("/users/:id/sessions/:sid", handler.RevokeSession)

	// Missing session ID - route won't match
	req, _ := http.NewRequest(http.MethodDelete, "/users/user-123/sessions/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSessionsHandler_RevokeAllUserSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewSessionsHandler(logger.NewNop())

	router.DELETE("/users/:id/sessions", handler.RevokeAllUserSessions)

	req, _ := http.NewRequest(http.MethodDelete, "/users/user-123/sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"], "not yet implemented")
}

func TestSessionsHandler_RevokeAllUserSessions_MissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewSessionsHandler(logger.NewNop())

	router.DELETE("/users/:id/sessions", handler.RevokeAllUserSessions)

	// Empty ID - Gin treats "//" as matching empty param, handler returns 400
	req, _ := http.NewRequest(http.MethodDelete, "/users//sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Handler returns 400 for empty ID
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
