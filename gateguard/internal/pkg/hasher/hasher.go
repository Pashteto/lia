package hasher

import (
	"crypto/sha256"
	"encoding/base64"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func HashToFixed8Chars(input string) string {
	hash := sha256.Sum256([]byte(input))
	hashBase64 := base64.StdEncoding.EncodeToString(hash[:])
	return Base62Encode([]byte(hashBase64))[:8]
}

func Base62Encode(input []byte) string {
	var encoded string
	for _, b := range input {
		encoded += string(base62Chars[b%62])
	}
	return encoded
}
