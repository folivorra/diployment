package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
)

func Verify(body, secret, signature []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(signature, expectedMAC)
}
