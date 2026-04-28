package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fincore/pkg/security"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "fincore/gen/go/auth/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestOAuth2ErrorRedirect_EncodesParams(t *testing.T) {
	loc, ok := oauth2ErrorRedirect("https://app.example/cb", "st", "invalid_request", "bad")
	if !ok {
		t.Fatalf("expected ok")
	}
	if !strings.Contains(loc, "error=invalid_request") {
		t.Fatalf("expected error in url, got %q", loc)
	}
	if !strings.Contains(loc, "error_description=bad") {
		t.Fatalf("expected error_description in url, got %q", loc)
	}
	if !strings.Contains(loc, "state=st") {
		t.Fatalf("expected state in url, got %q", loc)
	}
}

func TestOAuth2ErrorRedirect_RejectsFragment(t *testing.T) {
	_, ok := oauth2ErrorRedirect("https://app.example/cb#frag", "st", "invalid_request", "bad")
	if ok {
		t.Fatalf("expected not ok")
	}
}

func TestOAuth2ErrorFromGRPC_InvalidScope(t *testing.T) {
	code, desc := oauth2ErrorFromGRPC(status.Error(codes.InvalidArgument, "invalid scope"))
	if code != "invalid_scope" {
		t.Fatalf("expected invalid_scope, got %q", code)
	}
	if desc == "" {
		t.Fatalf("expected description")
	}
}

func TestOAuth2ErrorFromGRPC_PermissionDenied(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "no")
	code, _ := oauth2ErrorFromGRPC(err)
	if code != "access_denied" {
		t.Fatalf("expected access_denied, got %q", code)
	}
}

func TestAuthorizeCSRF_MissingTokenRedirectsError(t *testing.T) {
	h, _ := newHTTPHandler(http.NewServeMux(), nil, openIDConfiguration{}, security.JWKS{}, "/.well-known/jwks.json", nil, "http://example")

	// First GET should set CSRF cookie.
	getReq := httptest.NewRequest(http.MethodGet, "http://example/oauth/authorize?response_type=code&client_id=c1&redirect_uri=https%3A%2F%2Fapp.example%2Fcb&scope=openid&state=s&code_challenge=ch&code_challenge_method=S256", nil)
	getReq.Header.Set("Accept", "text/html")
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	csrfCookie := ""
	for _, c := range getW.Result().Cookies() {
		if c.Name == "fincore_authorize_csrf" {
			csrfCookie = c.Value
			break
		}
	}
	if strings.TrimSpace(csrfCookie) == "" {
		t.Fatalf("expected csrf cookie")
	}

	// POST without csrf_token should redirect with invalid_request.
	form := url.Values{}
	form.Set("client_id", "c1")
	form.Set("redirect_uri", "https://app.example/cb")
	form.Set("scope", "openid")
	form.Set("state", "s")
	form.Set("code_challenge", "ch")
	form.Set("code_challenge_method", "S256")
	form.Set("approve", "yes")
	form.Set("email", "u@example.com")
	form.Set("password", "pw")

	postReq := httptest.NewRequest(http.MethodPost, "http://example/oauth/authorize", io.NopCloser(strings.NewReader(form.Encode())))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Accept", "text/html")
	postReq.AddCookie(&http.Cookie{Name: "fincore_authorize_csrf", Value: csrfCookie})
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", postW.Code)
	}
	loc := postW.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if u.Query().Get("error") != "invalid_request" {
		t.Fatalf("expected invalid_request, got %q", u.Query().Get("error"))
	}
}

type stubAuthClient struct {
	oauthToken func(ctx context.Context, in *authv1.OAuthTokenRequest, opts ...grpc.CallOption) (*authv1.OAuthTokenResponse, error)
}

func (s stubAuthClient) OAuthToken(ctx context.Context, in *authv1.OAuthTokenRequest, opts ...grpc.CallOption) (*authv1.OAuthTokenResponse, error) {
	if s.oauthToken == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	return s.oauthToken(ctx, in, opts...)
}

func (s stubAuthClient) Register(ctx context.Context, in *authv1.RegisterRequest, opts ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) Login(ctx context.Context, in *authv1.LoginRequest, opts ...grpc.CallOption) (*authv1.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest, opts ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) Logout(ctx context.Context, in *authv1.LogoutRequest, opts ...grpc.CallOption) (*authv1.LogoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) LogoutAll(ctx context.Context, in *authv1.LogoutAllRequest, opts ...grpc.CallOption) (*authv1.LogoutAllResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) GrantRole(ctx context.Context, in *authv1.GrantRoleRequest, opts ...grpc.CallOption) (*authv1.GrantRoleResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) RevokeRole(ctx context.Context, in *authv1.RevokeRoleRequest, opts ...grpc.CallOption) (*authv1.RevokeRoleResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) ListUserRoles(ctx context.Context, in *authv1.ListUserRolesRequest, opts ...grpc.CallOption) (*authv1.ListUserRolesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) ValidateToken(ctx context.Context, in *authv1.ValidateTokenRequest, opts ...grpc.CallOption) (*authv1.ValidateTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) OAuthAuthorize(ctx context.Context, in *authv1.OAuthAuthorizeRequest, opts ...grpc.CallOption) (*authv1.OAuthAuthorizeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) CreateOAuthClient(ctx context.Context, in *authv1.CreateOAuthClientRequest, opts ...grpc.CallOption) (*authv1.CreateOAuthClientResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) GetOAuthClient(ctx context.Context, in *authv1.GetOAuthClientRequest, opts ...grpc.CallOption) (*authv1.GetOAuthClientResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) ListOAuthClients(ctx context.Context, in *authv1.ListOAuthClientsRequest, opts ...grpc.CallOption) (*authv1.ListOAuthClientsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) DeleteOAuthClient(ctx context.Context, in *authv1.DeleteOAuthClientRequest, opts ...grpc.CallOption) (*authv1.DeleteOAuthClientResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) RotateOAuthClientSecret(ctx context.Context, in *authv1.RotateOAuthClientSecretRequest, opts ...grpc.CallOption) (*authv1.RotateOAuthClientSecretResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) GetOAuthConsent(ctx context.Context, in *authv1.GetOAuthConsentRequest, opts ...grpc.CallOption) (*authv1.GetOAuthConsentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) StoreOAuthConsent(ctx context.Context, in *authv1.StoreOAuthConsentRequest, opts ...grpc.CallOption) (*authv1.StoreOAuthConsentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s stubAuthClient) ListOAuthConsentHistory(ctx context.Context, in *authv1.ListOAuthConsentHistoryRequest, opts ...grpc.CallOption) (*authv1.ListOAuthConsentHistoryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestOAuthToken_HTTP_InvalidClient_UsesRFCBodyAndWWWAuthenticate(t *testing.T) {
	client := stubAuthClient{oauthToken: func(ctx context.Context, in *authv1.OAuthTokenRequest, opts ...grpc.CallOption) (*authv1.OAuthTokenResponse, error) {
		for _, opt := range opts {
			if ho, ok := opt.(grpc.HeaderCallOption); ok {
				*ho.HeaderAddr = metadata.Pairs("www-authenticate", "Basic realm=\"oauth\"")
			}
		}
		return nil, status.Error(codes.Unauthenticated, "invalid_client")
	}}
	h, _ := newHTTPHandler(http.NewServeMux(), client, openIDConfiguration{}, security.JWKS{}, "/.well-known/jwks.json", nil, "http://example")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "code")
	form.Set("redirect_uri", "https://app.example/cb")
	form.Set("client_id", "c1")
	form.Set("code_verifier", "ver")
	req := httptest.NewRequest(http.MethodPost, "http://example/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if got := w.Header().Get("WWW-Authenticate"); got != "Basic realm=\"oauth\"" {
		t.Fatalf("expected WWW-Authenticate header, got %q", got)
	}
	var body oauthTokenErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error != "invalid_client" {
		t.Fatalf("expected invalid_client, got %q", body.Error)
	}
}

func TestOAuthToken_HTTP_UnsupportedGrantType_ReturnsUnsupportedGrantType(t *testing.T) {
	client := stubAuthClient{oauthToken: func(ctx context.Context, in *authv1.OAuthTokenRequest, opts ...grpc.CallOption) (*authv1.OAuthTokenResponse, error) {
		return nil, status.Error(codes.InvalidArgument, "unsupported grant_type")
	}}
	h, _ := newHTTPHandler(http.NewServeMux(), client, openIDConfiguration{}, security.JWKS{}, "/.well-known/jwks.json", nil, "http://example")

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req := httptest.NewRequest(http.MethodPost, "http://example/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body oauthTokenErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error != "unsupported_grant_type" {
		t.Fatalf("expected unsupported_grant_type, got %q", body.Error)
	}
}

func TestOAuthLogoutCookie_SetsSecureWhenTLS(t *testing.T) {
	h, _ := newHTTPHandler(http.NewServeMux(), stubAuthClient{}, openIDConfiguration{}, security.JWKS{}, "/.well-known/jwks.json", nil, "https://example")

	req := httptest.NewRequest(http.MethodPost, "https://example/oauth/logout", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "fincore_authorize_session" {
			found = true
			if !c.Secure {
				t.Fatalf("expected Secure cookie")
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected session cookie")
	}
}
