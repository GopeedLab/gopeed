package util

import (
	"fmt"
	"math"
)

const unknownSize = "unknown"

var unitArr = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

func ByteFmt(size int64) string {
	if size == 0 {
		return unknownSize
	}
	// Handle negative values
	if size < 0 {
		return unknownSize
	}
	fs := float64(size)
	p := int(math.Log(fs) / math.Log(1024))
	// Ensure index is within bounds
	if p < 0 {
		p = 0
	}
	if p >= len(unitArr) {
		p = len(unitArr) - 1
	}
	val := fs / math.Pow(1024, float64(p))
	_, frac := math.Modf(val)
	if frac > 0 {
		return fmt.Sprintf("%.1f%s", math.Floor(val*10)/10, unitArr[p])
	} else {
		return fmt.Sprintf("%d%s", int(val), unitArr[p])
	}
}
