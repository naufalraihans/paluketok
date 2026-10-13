// Helper signature webhook DOKU untuk E2E (algoritma sama dengan adapter:
// 5 komponen + Digest sha256 body mentah).
package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

func dokuSignedHeaders(body, reqID string) map[string]string {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	target := "/v1/webhooks/doku"
	digest := base64.StdEncoding.EncodeToString(sha256sum([]byte(body)))
	component := "Client-Id:" + dokuClientID +
		"\nRequest-Id:" + reqID +
		"\nRequest-Timestamp:" + ts +
		"\nRequest-Target:" + target +
		"\nDigest:" + digest
	mac := hmac.New(sha256.New, []byte(dokuSecret))
	mac.Write([]byte(component))
	return map[string]string{
		"Client-Id":         dokuClientID,
		"Request-Id":        reqID,
		"Request-Timestamp": ts,
		"Signature":         "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil)),
	}
}

func sha256sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
