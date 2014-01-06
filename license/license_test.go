package license

import "testing"

func TestLicense(t *testing.T) {
	l := New()
	if l.Id == "" {
		t.Error("Should have an id")
	}

	if l.Encryption.Profile != DEFAULT_PROFILE {
		t.Error("Expected %s, got %s", DEFAULT_PROFILE, l.Encryption.Profile)
	}
}
