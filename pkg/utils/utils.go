package utils

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// MyDuration is the encoding.TextUnmarshaler interface for time.Duration
type MyDuration struct {
	time.Duration
}

// UnmarshalText is used to convert from text to Duration
func (d *MyDuration) UnmarshalText(text []byte) error {
	res, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = res
	return nil
}

func ParseEndpoint(ep string) (string, string, error) {
	ep = strings.ToLower(ep)
	if strings.HasPrefix(ep, "unix://") || strings.HasPrefix(ep, "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

// RoundUpSize calculates how many allocation units are needed to accommodate a volume of given size.
// E.g. when user wants 1500Mi volume, while HuaweiCLoud EVS allocates volumes in gibibyte-sized chunks,
// RoundUpSize(1500 * 1024*1024, 1024*1024*1024) returns '2'
// (2 GiB is the smallest allocatable volume that can hold 1500MiB)
func RoundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	roundedUp := volumeSizeBytes / allocationUnitBytes
	if volumeSizeBytes%allocationUnitBytes > 0 {
		roundedUp++
	}
	return roundedUp
}

func BytesToGB(size interface{}) int64 {
	value := reflect.ValueOf(size)

	switch value.Kind() {
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		return value.Int() * 1024 * 1024 * 1024
	}

	return -1
}
