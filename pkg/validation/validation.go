package validation

import (
	"regexp"
	"strings"
	"unicode"
)

// ValidationError represents a validation error with a user-friendly message
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidateFullName validates the full name input
// Requirements: non-empty, no emojis, no digits, only letters and spaces
func ValidateFullName(name string) *ValidationError {
	name = strings.TrimSpace(name)

	if name == "" {
		return NewValidationError("full_name", "❌ Ism-familiya bo'sh bo'lmasligi kerak")
	}

	if len(name) < 3 {
		return NewValidationError("full_name", "❌ Ism-familiya juda qisqa")
	}

	if len(name) > 100 {
		return NewValidationError("full_name", "❌ Ism-familiya juda uzun")
	}

	// Check for digits
	for _, r := range name {
		if unicode.IsDigit(r) {
			return NewValidationError("full_name", "❌ Ism-familiyada raqamlar bo'lmasligi kerak")
		}
	}

	// Check for emojis and special characters
	if containsEmoji(name) {
		return NewValidationError("full_name", "❌ Ism-familiyada emoji yoki maxsus belgilar bo'lmasligi kerak")
	}

	// Check that it contains only letters, spaces, and common name characters
	validNameRegex := regexp.MustCompile(`^[\p{L}\s\-'.]+$`)
	if !validNameRegex.MatchString(name) {
		return NewValidationError("full_name", "❌ Ism-familiyada faqat harflar va bo'sh joy bo'lishi kerak")
	}

	// Should have at least first and last name (2 words minimum)
	words := strings.Fields(name)
	if len(words) < 2 {
		return NewValidationError("full_name", "❌ Iltimos, to'liq ism-familiyangizni kiriting (masalan: Abdullayev Abdulloh)")
	}

	return nil
}

// ValidateAge validates the age input
// Requirements: between 16 and 65
func ValidateAge(ageStr string) (int, *ValidationError) {
	ageStr = strings.TrimSpace(ageStr)

	if ageStr == "" {
		return 0, NewValidationError("age", "❌ Yoshingizni kiriting")
	}

	// Parse age
	var age int
	for _, r := range ageStr {
		if !unicode.IsDigit(r) {
			return 0, NewValidationError("age", "❌ Yosh faqat raqamlardan iborat bo'lishi kerak")
		}
	}

	// Convert to int
	for _, r := range ageStr {
		age = age*10 + int(r-'0')
	}

	if age < 16 {
		return 0, NewValidationError("age", "❌ Yosh 16 dan kichik bo'lmasligi kerak")
	}

	if age > 65 {
		return 0, NewValidationError("age", "❌ Yosh 65 dan katta bo'lmasligi kerak")
	}

	return age, nil
}

// ValidateWeight validates the weight input
// Requirements: between 30 and 200 kg
func ValidateWeight(weightStr string) (int, *ValidationError) {
	weightStr = strings.TrimSpace(weightStr)

	if weightStr == "" {
		return 0, NewValidationError("weight", "❌ Vazningizni kiriting")
	}

	// Parse weight
	var weight int
	for _, r := range weightStr {
		if !unicode.IsDigit(r) {
			return 0, NewValidationError("weight", "❌ Vazn faqat raqamlardan iborat bo'lishi kerak")
		}
	}

	for _, r := range weightStr {
		weight = weight*10 + int(r-'0')
	}

	if weight < 30 {
		return 0, NewValidationError("weight", "❌ Vazn 30 kg dan kam bo'lmasligi kerak")
	}

	if weight > 200 {
		return 0, NewValidationError("weight", "❌ Vazn 200 kg dan oshmasligi kerak")
	}

	return weight, nil
}

// ValidateHeight validates the height input
// Requirements: between 100 and 250 cm
func ValidateHeight(heightStr string) (int, *ValidationError) {
	heightStr = strings.TrimSpace(heightStr)

	if heightStr == "" {
		return 0, NewValidationError("height", "❌ Bo'yingizni kiriting")
	}

	// Parse height
	var height int
	for _, r := range heightStr {
		if !unicode.IsDigit(r) {
			return 0, NewValidationError("height", "❌ Bo'y faqat raqamlardan iborat bo'lishi kerak")
		}
	}

	for _, r := range heightStr {
		height = height*10 + int(r-'0')
	}

	if height < 100 {
		return 0, NewValidationError("height", "❌ Bo'y 100 sm dan kam bo'lmasligi kerak")
	}

	if height > 250 {
		return 0, NewValidationError("height", "❌ Bo'y 250 sm dan oshmasligi kerak")
	}

	return height, nil
}

// ParseBodyParams parses weight and height from a single input string
// Expected format: "70 175" or "70kg 175cm" or "70/175"
func ParseBodyParams(input string) (weight int, height int, err *ValidationError) {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	// Remove common suffixes
	input = strings.ReplaceAll(input, "kg", "")
	input = strings.ReplaceAll(input, "cm", "")
	input = strings.ReplaceAll(input, "sm", "")

	// Split by common separators
	var parts []string
	if strings.Contains(input, "/") {
		parts = strings.Split(input, "/")
	} else if strings.Contains(input, ",") {
		parts = strings.Split(input, ",")
	} else {
		parts = strings.Fields(input)
	}

	if len(parts) != 2 {
		return 0, 0, NewValidationError("body_params", "❌ Iltimos, vazn va bo'yni kiriting\nMasalan: 70 175")
	}

	weight, err = ValidateWeight(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}

	height, err = ValidateHeight(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}

	return weight, height, nil
}

// ValidatePhone validates phone number format
// Accepts formats: +998991234567 or 991234567
func ValidatePhone(phone string) *ValidationError {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return NewValidationError("phone", "❌ Telefon raqam bo'sh bo'lmasligi kerak")
	}

	// Remove spaces, dashes, parentheses
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	// Extract only digits
	var digits strings.Builder
	hasPlus := strings.HasPrefix(cleaned, "+")

	for _, r := range cleaned {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}

	phoneDigits := digits.String()

	// Check valid formats:
	// 1. +998XXXXXXXXX (13 digits total with country code)
	// 2. 998XXXXXXXXX (12 digits)
	// 3. 9XXXXXXXXX (9 digits, starts with 9)

	if hasPlus {
		// Format: +998XXXXXXXXX
		if len(phoneDigits) != 12 {
			return NewValidationError("phone", "❌ Noto'g'ri format! To'g'ri format: +998991234567 yoki 991234567")
		}
		if !strings.HasPrefix(phoneDigits, "998") {
			return NewValidationError("phone", "❌ Telefon raqam 998 bilan boshlanishi kerak!")
		}
		// Check operator code (second part after 998 should start with 9)
		if len(phoneDigits) > 3 && phoneDigits[3] != '9' {
			return NewValidationError("phone", "❌ Noto'g'ri operator kodi! To'g'ri format: +998991234567")
		}
	} else {
		// Format without +
		if len(phoneDigits) == 12 {
			// 998XXXXXXXXX format
			if !strings.HasPrefix(phoneDigits, "998") {
				return NewValidationError("phone", "❌ Telefon raqam 998 bilan boshlanishi kerak!")
			}
			if phoneDigits[3] != '9' {
				return NewValidationError("phone", "❌ Noto'g'ri operator kodi! To'g'ri format: 998991234567")
			}
		} else if len(phoneDigits) == 9 {
			// 9XXXXXXXXX format
			if phoneDigits[0] != '9' {
				return NewValidationError("phone", "❌ Telefon raqam 9 bilan boshlanishi kerak! To'g'ri format: 991234567")
			}
		} else {
			return NewValidationError("phone", "❌ Noto'g'ri format! To'g'ri format: +998991234567 yoki 991234567")
		}
	}

	return nil
}

// containsEmoji checks if the string contains emoji characters
func containsEmoji(s string) bool {
	for _, r := range s {
		// Basic emoji ranges
		if r >= 0x1F300 && r <= 0x1F9FF { // Miscellaneous Symbols and Pictographs, Emoticons, etc.
			return true
		}
		if r >= 0x2600 && r <= 0x26FF { // Miscellaneous Symbols
			return true
		}
		if r >= 0x2700 && r <= 0x27BF { // Dingbats
			return true
		}
		if r >= 0x1F600 && r <= 0x1F64F { // Emoticons
			return true
		}
		if r >= 0x1F680 && r <= 0x1F6FF { // Transport and Map Symbols
			return true
		}
	}
	return false
}

// NormalizeFullName normalizes the full name (proper case, trim spaces)
func NormalizeFullName(name string) string {
	name = strings.TrimSpace(name)
	words := strings.Fields(name)

	for i, word := range words {
		if len(word) > 0 {
			// Capitalize first letter, lowercase the rest
			runes := []rune(strings.ToLower(word))
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}

	return strings.Join(words, " ")
}

// NormalizePhone normalizes phone number to standard format +998XXXXXXXXX
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	// Remove all non-digit characters except +
	var digits strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}

	phoneDigits := digits.String()

	// Normalize to +998XXXXXXXXX format
	if len(phoneDigits) == 9 {
		// 9XXXXXXXXX -> +998XXXXXXXXX
		return "+998" + phoneDigits
	} else if len(phoneDigits) == 12 && strings.HasPrefix(phoneDigits, "998") {
		// 998XXXXXXXXX -> +998XXXXXXXXX
		return "+" + phoneDigits
	} else if strings.HasPrefix(phone, "+") {
		// Already has +, just return cleaned version
		return "+" + phoneDigits
	}

	// Default: add + if not present
	return "+" + phoneDigits
}
