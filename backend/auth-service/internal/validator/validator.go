package validator

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var SecurityQuestions = []string{
	"What was the name of your first pet?",
	"What city were you born in?",
	"What is your favorite video game?",
	"What was the name of your first school?",
	"What was your childhood nickname?",
}

var (
	UsernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,15}$`)
	EmailRe    = regexp.MustCompile(`^[^\s@]+@[^\s@.]+\.com$`)
)

type Registration struct {
	Name             string
	Username         string
	Email            string
	Password         string
	Gender           string
	DateOfBirth      string
	ProfileURL       string
	BannerURL        string
	SecurityQuestion string
	SecurityAnswer   string
}

func (r Registration) Validate(now time.Time) error {
	if err := ValidateName(r.Name); err != nil {
		return err
	}
	if !UsernameRe.MatchString(r.Username) {
		return errors.New("username must be 3-15 characters: letters, numbers, or underscore")
	}
	if !EmailRe.MatchString(r.Email) {
		return errors.New("email must match [name]@[domain].com")
	}
	if err := ValidatePassword(r.Password); err != nil {
		return err
	}
	if r.Gender != "male" && r.Gender != "female" {
		return errors.New("gender must be male or female")
	}
	if err := ValidateAge(r.DateOfBirth, now); err != nil {
		return err
	}
	if strings.TrimSpace(r.ProfileURL) == "" || strings.TrimSpace(r.BannerURL) == "" {
		return errors.New("profile picture and banner are required")
	}
	if !validQuestion(r.SecurityQuestion) {
		return errors.New("invalid security question")
	}
	if strings.TrimSpace(r.SecurityAnswer) == "" {
		return errors.New("security answer is required")
	}
	return nil
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) <= 4 {
		return errors.New("name must be more than 4 characters")
	}
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsSpace(c) {
			return errors.New("name must not contain symbols or numbers")
		}
	}
	return nil
}

func ValidatePassword(p string) error {
	var upper, lower, digit, symbol bool
	for _, c := range p {
		switch {
		case unicode.IsUpper(c):
			upper = true
		case unicode.IsLower(c):
			lower = true
		case unicode.IsDigit(c):
			digit = true
		default:
			symbol = true
		}
	}
	switch {
	case len(p) < 8:
		return errors.New("password must be at least 8 characters")
	case !upper:
		return errors.New("password needs an uppercase letter")
	case !lower:
		return errors.New("password needs a lowercase letter")
	case !digit:
		return errors.New("password needs a number")
	case !symbol:
		return errors.New("password needs a symbol")
	}
	return nil
}

func ValidateAge(dob string, now time.Time) error {
	birth, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return errors.New("date of birth must use format YYYY-MM-DD")
	}
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	if age < 13 {
		return errors.New("you must be at least 13 years old")
	}
	return nil
}

func validQuestion(q string) bool {
	for _, item := range SecurityQuestions {
		if item == q {
			return true
		}
	}
	return false
}