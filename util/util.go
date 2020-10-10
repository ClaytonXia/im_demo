package util

import (
	"crypto/rand"
	"fmt"
)

// RandID _
func RandID(len uint64) (idStr string, err error) {
	tmpBytes := make([]byte, len)
	n, err := rand.Read(tmpBytes)
	if err != nil {
		return
	}
	if uint64(n) != len {
		err = fmt.Errorf("rand id err")
		return
	}

	idStr = fmt.Sprintf("%x", tmpBytes)
	return
}
