package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParseRoundTrip(t *testing.T) {
	issuer := NewTokenIssuer("a-test-secret", time.Hour)
	want := User{ID: "9f1c", Email: "someone@example.com"}

	raw, err := issuer.Issue(want)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	got, err := issuer.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got != want {
		t.Errorf("round trip changed the user: got %+v, want %+v", got, want)
	}
}

// A token whose header claims alg "none" carries no signature at all.
//
// jwt/v5 already rejects these on its own, so this test does not prove that
// WithValidMethods is what stops them — it was verified to pass with the pin
// removed. It is here as a guard on behaviour: if Parse is ever rewritten in
// a way that starts trusting the header, this fails.
func TestParseRejectsUnsignedToken(t *testing.T) {
	issuer := NewTokenIssuer("a-test-secret", time.Hour)

	forged, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "somebody-elses-id",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building the forged token: %v", err)
	}

	if _, err := issuer.Parse(forged); err == nil {
		t.Fatal("Parse accepted an unsigned token")
	}
}

func TestParseRejectsAnotherSecret(t *testing.T) {
	mine := NewTokenIssuer("my-secret", time.Hour)
	theirs := NewTokenIssuer("their-secret", time.Hour)

	raw, err := theirs.Issue(User{ID: "9f1c", Email: "someone@example.com"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := mine.Parse(raw); err == nil {
		t.Fatal("Parse accepted a token signed with a different secret")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	// A negative lifetime issues a token that expired before it existed.
	issuer := NewTokenIssuer("a-test-secret", -time.Minute)

	raw, err := issuer.Issue(User{ID: "9f1c", Email: "someone@example.com"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := issuer.Parse(raw); err == nil {
		t.Fatal("Parse accepted an expired token")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	issuer := NewTokenIssuer("a-test-secret", time.Hour)

	for _, raw := range []string{"", "not-a-token", "a.b.c"} {
		if _, err := issuer.Parse(raw); err == nil {
			t.Errorf("Parse(%q) returned no error", raw)
		}
	}
}
