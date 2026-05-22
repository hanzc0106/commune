package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	pinSaltSize = 16
	pinKeySize  = 32
	pinTime     = 1
	pinMemory   = 64 * 1024
	pinThreads  = 4
)

func HashPIN(pin string) (string, error) {
	if len(pin) < 4 {
		return "", errors.New("PIN must be at least 4 characters")
	}
	salt := make([]byte, pinSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pin), salt, pinTime, pinMemory, pinThreads, pinKeySize)
	return fmt.Sprintf(
		"argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		pinMemory,
		pinTime,
		pinThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPIN(encodedHash string, pin string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 || parts[0] != "argon2id" {
		return false
	}
	if _, err := parseVersion(parts[1]); err != nil {
		return false
	}

	var memory uint32
	var time uint32
	var threads uint8
	for _, param := range strings.Split(parts[2], ",") {
		key, value, ok := strings.Cut(param, "=")
		if !ok {
			return false
		}
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return false
		}
		switch key {
		case "m":
			memory = uint32(parsed)
		case "t":
			time = uint32(parsed)
		case "p":
			threads = uint8(parsed)
		}
	}
	if memory == 0 || time == 0 || threads == 0 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	actualKey := argon2.IDKey([]byte(pin), salt, time, memory, threads, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1
}

func parseVersion(value string) (int, error) {
	versionText, ok := strings.CutPrefix(value, "v=")
	if !ok {
		return 0, errors.New("missing version")
	}
	return strconv.Atoi(versionText)
}
