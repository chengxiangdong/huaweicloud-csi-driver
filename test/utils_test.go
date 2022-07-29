package test

import (
	"testing"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/utils"
)

func TestBytesToGB(t *testing.T) {
	sizeGB := 1
	sizeBytes := utils.BytesToGB(sizeGB)

	expected := int64(sizeGB * 1024 * 1024 * 1024)
	if sizeBytes != expected {
		t.Fatalf("Error in BytesToGB, expected: %v, bug got: %v.", expected, sizeBytes)
	}
}

func TestRoundUpSize(t *testing.T) {
	actualSize := utils.RoundUpSize(1500 * 1024*1024, 1024*1024*1024)
	expectedSize := int64(2)

	if actualSize != expectedSize {
		t.Fatalf("Error in RoundUpSize, expected: %v, bug got: %v.", expectedSize, actualSize)
	}
}

