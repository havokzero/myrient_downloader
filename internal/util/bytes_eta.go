package util

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// FormatBytes converts a byte count into a human-readable string.
func FormatBytes(bytes int64, decimals int) string {
	if bytes == 0 {
		return "0 Bytes"
	}
	k := float64(1024)
	dm := decimals
	if dm < 0 {
		dm = 0
	}
	sizes := []string{"Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	i := int(math.Floor(math.Log(float64(bytes)) / math.Log(k)))
	value := float64(bytes) / math.Pow(k, float64(i))

	return fmt.Sprintf("%.*f %s", dm, value, sizes[i])
}

// CalculateETA estimates remaining time given current/total and start time.
func CalculateETA(current, total int64, startTime time.Time) string {
	if current == 0 || total == 0 || current >= total {
		return "--"
	}

	elapsed := time.Since(startTime)
	ratio := float64(current) / float64(total)
	if ratio <= 0 {
		return "--"
	}

	estimatedTotal := time.Duration(float64(elapsed) / ratio)
	remaining := estimatedTotal - elapsed
	if remaining < 0 {
		remaining = 0
	}

	sec := int(remaining.Seconds())
	min := sec / 60
	hr := min / 60

	out := ""
	if hr > 0 {
		out += fmt.Sprintf("%dh ", hr)
	}
	if min%60 > 0 {
		out += fmt.Sprintf("%dm ", min%60)
	}
	if sec%60 > 0 || out == "" {
		out += fmt.Sprintf("%ds", sec%60)
	}
	return strings.TrimSpace(out)
}
