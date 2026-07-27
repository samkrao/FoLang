package helpers

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"

	"crypto/rand"

	"crypto/hmac"
	"crypto/sha256"
	"regexp"

	"github.com/google/uuid"
)

// SX is an alias for any, used as the generic serialization type.
type SX any

// Charsets
const (
	startCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	bodyCharset  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
)

// Generate a valid random C++ identifier (with HMAC-based suffix)
func generateCppIdentifier(prefix string, key []byte, input string, saltLen, suffixLen int) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// HMAC(input + salt)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(input))
	mac.Write(salt)
	hash := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Clean hash for C++ identifier
	clean := sanitizeCppIdentifier(hash)
	if len(clean) < suffixLen {
		suffixLen = len(clean)
	}

	// Combine
	return prefix + "_" + clean[:suffixLen], nil
}

// Sanitize to C++-legal identifier chars
func sanitizeCppIdentifier(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(s, "_")
}

// GenID generates a unique C++ identifier by combining a name with an HMAC-based suffix.
func GenID(name string, in string) (string, error) {
	key := append(genUUID(), []byte(in)...)
	input := name

	id, err := generateCppIdentifier("co", key, input, 8, 20)

	return id, err
}

func genUUID() []byte {
	u := uuid.New() // UUIDv4 by default
	return u[:]

}

// ToGOB64 encodes the given value to a base64-encoded GOB string.
func ToGOB64(m SX) string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

// FromGOB64 decodes a base64-encoded GOB string back into a value.
func FromGOB64(str string) SX {
	m := new(SX)
	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println(`failed base64 Decode`, err)
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		fmt.Println(`failed gob Decode`, err)
	}
	return m
}

func init_[T any](t_ T) {
	gob.Register(t_)

}
