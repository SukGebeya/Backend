// Package phone normalizes Ethiopian mobile numbers. Ethio Telecom numbers start
// with 9 and Safaricom Ethiopia numbers with 7; both have 9 digits after the
// country code (+251).
package phone

import (
	"errors"
	"strings"
)

var ErrInvalid = errors.New("invalid Ethiopian mobile number")

// Normalize converts any common way of writing an Ethiopian mobile number
// (0911234567, 911234567, 251911234567, +251 91 123 4567, 00251…) into E.164
// form: +251911234567.
func Normalize(raw string) (string, error) {
	var digits strings.Builder
	for i, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '.':
		default:
			return "", ErrInvalid
		}
	}

	d := digits.String()
	d = strings.TrimPrefix(d, "00")
	switch {
	case len(d) == 12 && strings.HasPrefix(d, "251"):
		d = d[3:]
	case len(d) == 10 && strings.HasPrefix(d, "0"):
		d = d[1:]
	}

	if len(d) != 9 || (d[0] != '9' && d[0] != '7') {
		return "", ErrInvalid
	}
	return "+251" + d, nil
}

// Local returns the 10-digit national format (09XXXXXXXX / 07XXXXXXXX) that
// Chapa and Ethiopian mobile-money providers expect.
func Local(raw string) (string, error) {
	normalized, err := Normalize(raw)
	if err != nil {
		return "", err
	}
	return "0" + normalized[4:], nil
}

func IsValid(raw string) bool {
	_, err := Normalize(raw)
	return err == nil
}

// NormalizePtr normalizes an optional phone in place; nil and empty values are left alone.
func NormalizePtr(raw *string) (*string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	normalized, err := Normalize(*raw)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}
