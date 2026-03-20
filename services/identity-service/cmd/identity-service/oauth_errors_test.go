package main

import (
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
	err := status.Error(codes.InvalidArgument, "invalid scope")
	code, _ := oauth2ErrorFromGRPC(err)
	if code != "invalid_scope" {
		t.Fatalf("expected invalid_scope, got %q", code)
	}
}

func TestOAuth2ErrorFromGRPC_PermissionDenied(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "no")
	code, _ := oauth2ErrorFromGRPC(err)
	if code != "access_denied" {
		t.Fatalf("expected access_denied, got %q", code)
	}
}
