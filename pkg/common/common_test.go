package common

import (
	"testing"

	"github.com/chnsz/golangsdk"
	"github.com/stretchr/testify/assert"
)

func TestIsNotFound(t *testing.T) {
	err404 := golangsdk.ErrDefault404{}
	if !IsNotFound(err404) {
		t.Errorf("Error in TestIsNotFound")
	}
}

func TestWaitForCompleted(t *testing.T) {
	n := 0

	err := WaitForCompleted(func() (bool, error) {
		if n++; n == 2 {
			return true, nil
		}
		return false, nil
	})
	assert.Nil(t, err)
}
