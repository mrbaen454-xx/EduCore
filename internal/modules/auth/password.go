package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrEmptyPassword   = errors.New("password cannot be empty")
	ErrInvalidHash     = errors.New("invalid or malformed password hash format")
	ErrIncompatibleVer = errors.New("incompatible argon2 version")
)

// PasswordHasher defines methods for hashing and verifying passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

// Argon2Hasher implements PasswordHasher using the Argon2id algorithm.
type Argon2Hasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// NewArgon2Hasher creates an Argon2Hasher with the given parameters, falling back to OWASP defaults if unconfigured.
func NewArgon2Hasher(memory uint32, iterations uint32, parallelism uint8, saltLength, keyLength uint32) *Argon2Hasher {
	if memory == 0 {
		memory = 64 * 1024
	}
	if iterations == 0 {
		iterations = 3
	}
	if parallelism == 0 {
		parallelism = 2
	}
	if saltLength == 0 {
		saltLength = 16
	}
	if keyLength == 0 {
		keyLength = 32
	}
	return &Argon2Hasher{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		saltLength:  saltLength,
		keyLength:   keyLength,
	}
}

// Hash creates a standard Argon2id PHC-formatted string from a plaintext password.
// Format: $argon2id$v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>
func (h *Argon2Hasher) Hash(password string) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.iterations,
		h.memory,
		h.parallelism,
		h.keyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.memory,
		h.iterations,
		h.parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// Verify checks a plaintext password against an Argon2id PHC-formatted hash.
// Uses constant-time comparison to protect against timing attacks.
func (h *Argon2Hasher) Verify(password, encodedHash string) (bool, error) {
	if len(password) == 0 {
		return false, ErrEmptyPassword
	}

	parts := strings.Split(encodedHash, "$")
	// Format: ["", "argon2id", "v=19", "m=65536,t=3,p=2", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHash
	}
	if version != argon2.Version {
		return false, ErrIncompatibleVer
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHash
	}

	candidateHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	if subtle.ConstantTimeCompare(candidateHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}
