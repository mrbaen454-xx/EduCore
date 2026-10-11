package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"educore/internal/modules/auth"
)

func TestAuth_RefreshTokenRotationAndReuseDetection(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	// 1. Initial login to obtain first refresh token
	loginPayload := auth.LoginRequest{
		Identifier: "admin@example.com",
		Password:   "AdminSecret123!",
	}
	body, _ := json.Marshal(loginPayload)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(loginRec, loginReq)

	var loginRes struct {
		Data auth.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(loginRec.Body.Bytes(), &loginRes)
	initialRefreshToken := loginRes.Data.RefreshToken

	if len(initialRefreshToken) == 0 {
		t.Fatal("expected non-empty initial refresh token")
	}

	// 2. Perform token refresh (rotation)
	refreshPayload := auth.RefreshTokenRequest{RefreshToken: initialRefreshToken}
	refBody, _ := json.Marshal(refreshPayload)
	refReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refBody))
	refReq.Header.Set("Content-Type", "application/json")
	refRec := httptest.NewRecorder()
	server.Router().ServeHTTP(refRec, refReq)

	if refRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on refresh, got %d: %s", refRec.Code, refRec.Body.String())
	}

	var refRes struct {
		Data auth.TokenRefreshResponse `json:"data"`
	}
	_ = json.Unmarshal(refRec.Body.Bytes(), &refRes)
	secondRefreshToken := refRes.Data.RefreshToken

	if len(secondRefreshToken) == 0 || secondRefreshToken == initialRefreshToken {
		t.Error("expected new rotated refresh token different from initial token")
	}

	// 3. Test Reuse Detection: Attempting to use initialRefreshToken again MUST be rejected!
	reuseReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refBody))
	reuseReq.Header.Set("Content-Type", "application/json")
	reuseRec := httptest.NewRecorder()
	server.Router().ServeHTTP(reuseRec, reuseReq)

	if reuseRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on token reuse attempt, got %d", reuseRec.Code)
	}

	// 4. Because reuse detection revoked all sessions, even secondRefreshToken should now be revoked
	secondBody, _ := json.Marshal(auth.RefreshTokenRequest{RefreshToken: secondRefreshToken})
	secondReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(secondBody))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	server.Router().ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after reuse detection triggered global revocation, got %d", secondRec.Code)
	}
}

func TestAuth_Logout(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	// 1. Login
	loginPayload := auth.LoginRequest{
		Identifier: "admin@example.com",
		Password:   "AdminSecret123!",
	}
	body, _ := json.Marshal(loginPayload)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(loginRec, loginReq)

	var loginRes struct {
		Data auth.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(loginRec.Body.Bytes(), &loginRes)
	refreshToken := loginRes.Data.RefreshToken

	// 2. Logout with refresh token
	logoutPayload := auth.LogoutRequest{RefreshToken: refreshToken}
	logoutBody, _ := json.Marshal(logoutPayload)
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(logoutBody))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutRec := httptest.NewRecorder()
	server.Router().ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d: %s", logoutRec.Code, logoutRec.Body.String())
	}

	// 3. Attempting to refresh using logged-out token must fail
	refBody, _ := json.Marshal(auth.RefreshTokenRequest{RefreshToken: refreshToken})
	refReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refBody))
	refReq.Header.Set("Content-Type", "application/json")
	refRec := httptest.NewRecorder()
	server.Router().ServeHTTP(refRec, refReq)

	if refRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 using logged-out refresh token, got %d", refRec.Code)
	}
}
