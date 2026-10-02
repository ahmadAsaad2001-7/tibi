package kashier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// VerifySignature checks the HMAC-SHA256 signature over the raw body
// against the shared webhook secret.
//
// Header name and encoding (hex vs base64) must be confirmed against
// Kashier's docs. This assumes: hex-encoded HMAC-SHA256, header name
// "Kashier-Signature".
func VerifySignature(secret string, rawBody []byte, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}
