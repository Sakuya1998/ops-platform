package crypto

import "testing"

func TestSecretBoxRoundTrip(t *testing.T) {
	box := NewSecretBox("test-key")
	encrypted, err := box.EncryptString("secret-value")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if encrypted == "secret-value" {
		t.Fatal("expected encrypted value to differ from plaintext")
	}
	decrypted, err := box.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if decrypted != "secret-value" {
		t.Fatalf("expected secret-value, got %s", decrypted)
	}
}

func TestSecretBoxPlaintextCompatibility(t *testing.T) {
	box := NewSecretBox("test-key")
	decrypted, err := box.DecryptString("plain-value")
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if decrypted != "plain-value" {
		t.Fatalf("expected plaintext compatibility, got %s", decrypted)
	}
}
