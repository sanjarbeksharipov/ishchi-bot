package helper

// valueOrDefault returns the value if not empty, otherwise returns the default
func ValueOrDefault(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}
