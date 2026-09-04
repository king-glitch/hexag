package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateUserSessionToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x.%x.%x", b[0:8], b[8:16], b[16:24]), nil
}

// HashToken is the storage-at-rest form of a session token — the raw token
// is only ever handed to the client at issuance and sent back as a bearer
// header; everywhere it is persisted or looked up, it MUST go through this
// first, so a database read (backup, snapshot, compromised replica) does not
// hand out live bearer credentials. Unsalted SHA-256 is deliberate here: the
// input is a 192-bit crypto/rand token (not a low-entropy secret like a
// password), so it doubles as a deterministic lookup key.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
