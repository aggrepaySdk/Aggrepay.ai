package aggrepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

func BodySHA256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// RequestCanonical: METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body).
func RequestCanonical(method, path, timestamp, nonce string, body []byte) string {
	var b strings.Builder
	b.Grow(len(method) + len(path) + len(timestamp) + len(nonce) + 64 + 4)
	b.WriteString(strings.ToUpper(method))
	b.WriteByte('\n')
	b.WriteString(path)
	b.WriteByte('\n')
	b.WriteString(timestamp)
	b.WriteByte('\n')
	b.WriteString(nonce)
	b.WriteByte('\n')
	b.WriteString(BodySHA256Hex(body))
	return b.String()
}

func Sign(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// CallbackCanonical: path всегда /merchant/callback.
func CallbackCanonical(timestamp string, body []byte) string {
	return "POST\n/merchant/callback\n" + timestamp + "\n" + BodySHA256Hex(body)
}

func VerifyCallback(secret, timestamp string, body []byte, signature string) bool {
	if secret == "" || timestamp == "" || signature == "" {
		return false
	}
	expected := Sign(secret, CallbackCanonical(timestamp, body))
	a := []byte(strings.ToLower(expected))
	b := []byte(strings.ToLower(signature))
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
