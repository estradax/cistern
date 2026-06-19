package object

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"
)

func GenerateSignature(secret, method, bucketKey, objectKey string, expires int64) string {
	var message string
	if method == "POST" {
		message = fmt.Sprintf("POST\n%s\n%s\n%d", bucketKey, objectKey, expires)
	} else {
		message = fmt.Sprintf("GET\n%s\n%d", objectKey, expires)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func VerifySignature(secret, method, bucketKey, objectKey string, expires int64, signature string) bool {
	if time.Now().Unix() > expires {
		return false
	}

	expected := GenerateSignature(secret, method, bucketKey, objectKey, expires)
	sigBytes, err1 := hex.DecodeString(signature)
	expBytes, err2 := hex.DecodeString(expected)
	if err1 != nil || err2 != nil {
		return false
	}

	return subtle.ConstantTimeCompare(sigBytes, expBytes) == 1
}
