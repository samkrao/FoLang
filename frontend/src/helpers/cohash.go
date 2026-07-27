package helpers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/gofrs/uuid"
)

// Salt holds the global salt bytes used for hashing operations.
var Salt []byte = nil

// GenUUID generates and returns a new random UUID v4 string.
func GenUUID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

// Allowed alphabet: A–Z, a–z, 0–9, _
const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"

var base = big.NewInt(int64(len(alphabet))) // 63

// base63Encode encodes arbitrary bytes to exactly 'length' base-63 chars.
func base63Encode(data []byte, length int) string {
	n := new(big.Int).SetBytes(data) // big-endian
	out := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		var q, r big.Int
		q.DivMod(n, base, &r)
		out[i] = alphabet[r.Int64()]
		n.Set(&q)
	}
	return string(out)
}

// HMAC-SHA256(input, salt) -> base63(length)
func hashCode(input string, salt []byte, length int) string {
	m := hmac.New(sha256.New, salt)
	m.Write([]byte(input))
	d := m.Sum(nil) // 32 bytes
	return base63Encode(d, length)
}

// GenerateSalt returns cryptographically-random salt bytes (n >= 16 recommended).
func GenerateSalt(n int) ([]byte, error) {
	s := make([]byte, n)
	_, err := rand.Read(s)
	return s, err
}

// Hashmain computes an HMAC-SHA256 hash of str, using the provided salt or generating a new one.
func Hashmain(str string, salt_prov []byte, length int) (string, []byte) {
	// 1) Load salt from env for reproducibility; otherwise generate a new one.
	var salt []byte // put raw bytes here if you like
	if len(salt_prov) == 0 {
		var err error
		salt, err = GenerateSalt(32) // 256-bit salt
		if err != nil {
			panic(err)
		}
		fmt.Println("INFO: generated a new random salt; persist it for stable hashes.")
	} else {
		salt = salt_prov
	}

	return hashCode(str, salt, length), salt
}
