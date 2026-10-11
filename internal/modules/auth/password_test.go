package auth

import (
	"testing"
)

func TestArgon2Hasher_HashAndVerify(t *testing.T) {
	hasher := NewArgon2Hasher(16*1024, 1, 1, 16, 32) // Lightweight parameters for fast unit tests

	password := "SecureP@ssword123!"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}

	// Verify correct password
	match, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("expected no error verifying valid password, got %v", err)
	}
	if !match {
		t.Error("expected password to match, got false")
	}

	// Verify wrong password
	wrongMatch, err := hasher.Verify("WrongPassword456!", hash)
	if err != nil {
		t.Fatalf("expected no error verifying wrong password, got %v", err)
	}
	if wrongMatch {
		t.Error("expected wrong password to not match, got true")
	}
}

func TestArgon2Hasher_DifferentSaltsProduceDifferentHashes(t *testing.T) {
	hasher := NewArgon2Hasher(16*1024, 1, 1, 16, 32)
	password := "CommonPassword"

	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed hash 1: %v", err)
	}

	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed hash 2: %v", err)
	}

	if hash1 == hash2 {
		t.Error("two hashes of the same password should not be identical due to random salts")
	}

	match1, _ := hasher.Verify(password, hash1)
	match2, _ := hasher.Verify(password, hash2)
	if !match1 || !match2 {
		t.Error("both distinct hashes should verify correctly against the original password")
	}
}

func TestArgon2Hasher_EmptyPassword(t *testing.T) {
	hasher := NewArgon2Hasher(16*1024, 1, 1, 16, 32)

	_, err := hasher.Hash("")
	if err != ErrEmptyPassword {
		t.Errorf("expected ErrEmptyPassword, got %v", err)
	}

	_, err = hasher.Verify("", "$argon2id$v=19$m=16384,t=1,p=1$c2FsdA$aGFzaA")
	if err != ErrEmptyPassword {
		t.Errorf("expected ErrEmptyPassword on verify, got %v", err)
	}
}

func TestArgon2Hasher_MalformedHash(t *testing.T) {
	hasher := NewArgon2Hasher(16*1024, 1, 1, 16, 32)

	invalidHashes := []string{
		"not-a-hash",
		"$bcrypt$v=19$m=16384,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=99$m=16384,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=abc,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=16384,t=1,p=1$invalid-salt!$aGFzaA",
	}

	for _, invalid := range invalidHashes {
		match, err := hasher.Verify("password", invalid)
		if err == nil {
			t.Errorf("expected error for malformed hash '%s', got match=%v", invalid, match)
		}
	}
}
