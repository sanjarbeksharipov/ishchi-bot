package helper

import "fmt"

// valueOrDefault returns the value if not empty, otherwise returns the default
func ValueOrDefault(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}

// FormatMoney formats an integer with space as thousands separator.
// Example: 10000 -> "10 000", 1500000 -> "1 500 000"
func FormatMoney(n int) string {
	if n < 0 {
		return "-" + FormatMoney(-n)
	}
	s := fmt.Sprintf("%d", n)
	result := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ' ')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
