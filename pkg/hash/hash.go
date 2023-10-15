package hash

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
)

func Hash(data any) (string, error) {
	h := sha256.New()
	enc := gob.NewEncoder(h)
	if err := enc.Encode(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
