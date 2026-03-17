package security

import "testing"

func TestHashAndVerifyPassword_RoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	ok, err := VerifyPassword("correct horse battery staple", h)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if !ok {
		t.Fatalf("expected password to verify")
	}

	ok, err = VerifyPassword("wrong", h)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatalf("expected wrong password to fail")
	}
}

func TestVerifyPassword_ErrorsOnEmptyInputs(t *testing.T) {
	if _, err := HashPassword(" "); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := VerifyPassword("", "x"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := VerifyPassword("x", " "); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseArgon2idHash_InvalidFormats(t *testing.T) {
	cases := []string{
		"",
		"$argon2id$v=19$m=65536,t=3,p=2$saltonly",
		"$bcrypt$v=19$m=65536,t=3,p=2$c2FsdA$hash",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$hash",
		"$argon2id$v=19$m=bad,t=3,p=2$c2FsdA$hash",
		"$argon2id$v=19$m=65536,t=3,p=2$%%%$hash",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$%%%",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$",
	}
	for _, tc := range cases {
		_, _, _, err := parseArgon2idHash(tc)
		if err == nil {
			t.Fatalf("expected error for %q", tc)
		}
	}
}
