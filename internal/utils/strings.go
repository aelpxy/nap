package utils

func TruncateID(id string, length int) string {
	if length < 0 {
		length = 0
	}
	if len(id) <= length {
		return id
	}
	return id[:length]
}

func TruncateString(s string, max int) string {
	if max < 3 {
		max = 3
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func MaskSensitive(value string, showChars int) string {
	if showChars < 0 {
		showChars = 0
	}
	if len(value) <= showChars {
		return "****"
	}
	return value[:showChars] + "****"
}
