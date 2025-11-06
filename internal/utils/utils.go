package utils

import (
	"strings"
	"unicode/utf8"
)

func IsEmailValid(email string) bool {

	if email == "" {
		return false
	}

	if !strings.Contains(email, "@") {
		return false
	}

	firstPart, secondPart := strings.Split(email, "@")[0], strings.Split(email, "@")[1]
	if firstPart == "" || secondPart == "" {
		return false
	}

	if strings.Contains(secondPart, ".") && len(strings.Split(secondPart, ".")) > 1 {
		return true
	} else {
		return false
	}
}

func IsPasswordStrong(password string) bool {

	if utf8.RuneCountInString(password) < 6 {
		return false
	}

	const SPECIAL_CHARS = "!@#$%^&*()-_=+[]{}|:;'<>,.?/"
	const NUMERICAL_CHARS = "0123456789"
	const LOWERCASE_CHARS = "abcdefghijklmnopqrstuvwxyz"
	const UPPERCASE_CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	hasSpecialChars := false
	hasNumericalChars := false
	hasLowercaseChars := false
	hasUpperCaseChars := false
	isPasswordStrong := false

	for _, c := range password {

		if hasSpecialChars && hasNumericalChars && hasLowercaseChars && hasUpperCaseChars {
			isPasswordStrong = true
			break
		}

		if !hasSpecialChars && strings.Contains(SPECIAL_CHARS, string(c)) {
			hasSpecialChars = true
		}

		if !hasNumericalChars && strings.Contains(NUMERICAL_CHARS, string(c)) {
			hasNumericalChars = true
		}

		if !hasLowercaseChars && strings.Contains(LOWERCASE_CHARS, string(c)) {
			hasLowercaseChars = true
		}

		if !hasUpperCaseChars && strings.Contains(UPPERCASE_CHARS, string(c)) {
			hasUpperCaseChars = true
		}
	}

	if !isPasswordStrong {
		return hasSpecialChars && hasNumericalChars && hasLowercaseChars && hasUpperCaseChars
	} else {
		return isPasswordStrong
	}
}
