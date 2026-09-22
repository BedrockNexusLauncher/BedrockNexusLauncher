package xbox

import (
	"context"
	"testing"
)

func TestValidateWAMIdentity(t *testing.T) {
	if err := validateWAMIdentity("id-1", "id-1", "tok"); err != nil {
		t.Fatalf("expected nil for matching identity, got %v", err)
	}
	if err := validateWAMIdentity("id-1", "id-2", "tok"); err != ErrAccountChanged {
		t.Fatalf("expected ErrAccountChanged, got %v", err)
	}
	if err := validateWAMIdentity("", "", ""); err != ErrAuthenticationFailed {
		t.Fatalf("expected ErrAuthenticationFailed for empty, got %v", err)
	}
}

func TestWamShouldForceSignInUI(t *testing.T) {
	if !wamShouldForceSignInUI(1234, errSignInUIRequired) {
		t.Fatal("expected forced prompt for window-scoped UI-required failure")
	}
	if wamShouldForceSignInUI(0, errSignInUIRequired) {
		t.Fatal("silent request must never open a dialog")
	}
	if wamShouldForceSignInUI(1234, context.Canceled) {
		t.Fatal("dismissal is final, must not retry")
	}
}
