package main

import (
	"fincore/pkg/security"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
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
	h := newHTTPHandler(http.NewServeMux(), nil, openIDConfiguration{}, security.JWKS{}, "/.well-known/jwks.json")

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
