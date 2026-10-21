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
	ErrInvalidHash         = errors.New("invalid password hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

type PasswordHasher struct {
	time      uint32
	memory    uint32
	threads   uint8
	keyLen    uint32
	dummyHash string
}

func NewPasswordHasher(time uint32, memory uint32, threads uint8, keyLen uint32) *PasswordHasher {
	if time == 0 {
		time = 3
	}
	if memory == 0 {
		memory = 64 * 1024
	}
	if threads == 0 {
		threads = 2
	}
	if keyLen == 0 {
		keyLen = 32
	}

	h := &PasswordHasher{
		time:    time,
		memory:  memory,
		threads: threads,
		keyLen:  keyLen,
	}

	dummy, err := h.HashPassword("dummy-password-for-timing-mitigation")
	if err != nil {
		panic(err)
	}
	h.dummyHash = dummy
	return h
}

func DefaultPasswordHasher() *PasswordHasher {
	return NewPasswordHasher(3, 64*1024, 2, 32)
}

func FastPasswordHasher() *PasswordHasher {
	return NewPasswordHasher(1, 1024, 1, 32)
}

func (h *PasswordHasher) HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, h.time, h.memory, h.threads, h.keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, h.memory, h.time, h.threads, b64Salt, b64Hash), nil
}

func (h *PasswordHasher) VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHash
	}
	if version != argon2.Version {
		return false, ErrIncompatibleVersion
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
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

	keyLen := uint32(len(expectedHash))
	calculatedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	if subtle.ConstantTimeCompare(expectedHash, calculatedHash) == 1 {
		return true, nil
	}

	return false, nil
}

func (h *PasswordHasher) DummyVerify(password string) {
	_, _ = h.VerifyPassword(password, h.dummyHash)
}
