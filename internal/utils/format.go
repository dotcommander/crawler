package utils

import (
	"fmt"
	"time"
)

// TruncateURL truncates a URL to the specified maximum length, adding "..." if truncated.
func TruncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// FormatBytes formats a byte count as a human-readable string with optional space before unit.
// Example: FormatBytes(1536, true) returns "1.5 KB", FormatBytes(1536, false) returns "1.5KB"
func FormatBytes(bytes int64, withSpace bool) string {
	const unit = 1024
	if bytes < unit {
		if withSpace {
			return fmt.Sprintf("%d B", bytes)
		}
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	format := "%.1f%cB"
	if withSpace {
		format = "%.1f %cB"
	}
	return fmt.Sprintf(format, float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration as a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
