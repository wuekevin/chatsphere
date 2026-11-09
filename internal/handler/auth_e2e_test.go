package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/handler"
	"github.com/wuekevin/chatsphere/internal/testutil"
)

func init() { gin.SetMode(gin.TestMode) }

func buildAuthServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testutil.MustDB(t)
	cfg := &config.Config{
		Server: config.ServerConfig{SecretKey: "testsecret"},
	}
	r := gin.New()
	r.Use(gin.Recovery())
	handler.RegisterAuth(r, db, cfg)
	return httptest.NewServer(r)
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body interface{}) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// TestRegisterAndLogin verifies a user can register and then log in with correct credentials.
func TestRegisterAndLogin(t *testing.T) {
	srv := buildAuthServer(t)
	defer srv.Close()

	resp := postJSON(t, srv, "/api/auth/register", map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "Password123!",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", resp.StatusCode)
	}

	resp = postJSON(t, srv, "/api/auth/login", map[string]string{
		"username": "testuser",
		"password": "Password123!",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["token"] == nil || body["token"] == "" {
		t.Errorf("expected JWT token in response, got: %v", body)
	}
}

// TestLoginWrongPassword verifies incorrect credentials return 401.
func TestLoginWrongPassword(t *testing.T) {
	srv := buildAuthServer(t)
	defer srv.Close()

	postJSON(t, srv, "/api/auth/register", map[string]string{
		"username": "user2",
		"email":    "user2@example.com",
		"password": "CorrectPass1!",
	})

	resp := postJSON(t, srv, "/api/auth/login", map[string]string{
		"username": "user2",
		"password": "WrongPass!",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login with wrong password: status = %d, want 401", resp.StatusCode)
	}
}

// TestDuplicateUsername verifies registering the same username twice returns 409.
func TestDuplicateUsername(t *testing.T) {
	srv := buildAuthServer(t)
	defer srv.Close()

	postJSON(t, srv, "/api/auth/register", map[string]string{
		"username": "dupuser",
		"email":    "dup@example.com",
		"password": "Pass123!",
	})
	resp := postJSON(t, srv, "/api/auth/register", map[string]string{
		"username": "dupuser",
		"email":    "dup2@example.com",
		"password": "Pass123!",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate register: status = %d, want 409", resp.StatusCode)
	}
}
