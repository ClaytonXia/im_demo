package util

import (
	"fmt"
	"testing"
)

func TestRandID(t *testing.T) {
	idStr, err := RandID(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(idStr)
}
