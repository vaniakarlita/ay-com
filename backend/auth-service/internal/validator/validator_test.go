package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var fixedNow = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid with space", "Adrian Yu", false},
		{"exactly four characters", "Adri", true},
		{"five characters", "Adria", false},
		{"too short", "Bob", true},
		{"contains number", "Adrian1", true},
		{"contains symbol", "Adrian!", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "Abcdef1!", false},
		{"too short", "Ab1!", true},
		{"no uppercase", "abcdef1!", true},
		{"no lowercase", "ABCDEF1!", true},
		{"no number", "Abcdefg!", true},
		{"no symbol", "Abcdefg1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestValidateAge(t *testing.T) {
	tests := []struct {
		name    string
		dob     string
		wantErr bool
	}{
		{"exactly 13 today", "2013-11-15", false},
		{"turns 13 tomorrow", "2013-11-16", true},
		{"adult", "2000-01-15", false},
		{"child", "2020-01-01", true},
		{"bad format", "15-01-2000", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAge(tt.dob, fixedNow)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func validRegistration() Registration {
	return Registration{
		Name:             "Adrian Yu",
		Username:         "adrian_yu",
		Email:            "adrian.yu@gmail.com",
		Password:         "Abcdef1!",
		Gender:           "male",
		DateOfBirth:      "2000-01-15",
		ProfileURL:       "https://example.com/p.png",
		BannerURL:        "https://example.com/b.png",
		SecurityQuestion: SecurityQuestions[0],
		SecurityAnswer:   "bobby",
	}
}

func TestRegistrationValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Registration)
		wantErr bool
	}{
		{"valid", func(r *Registration) {}, false},
		{"email without .com", func(r *Registration) { r.Email = "adrian@gmail.co" }, true},
		{"email without at", func(r *Registration) { r.Email = "adrian.gmail.com" }, true},
		{"bad username", func(r *Registration) { r.Username = "a b" }, true},
		{"gender other", func(r *Registration) { r.Gender = "other" }, true},
		{"missing profile picture", func(r *Registration) { r.ProfileURL = "" }, true},
		{"missing banner", func(r *Registration) { r.BannerURL = "" }, true},
		{"unknown question", func(r *Registration) { r.SecurityQuestion = "Favorite color?" }, true},
		{"empty answer", func(r *Registration) { r.SecurityAnswer = "  " }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validRegistration()
			tt.mutate(&r)
			err := r.Validate(fixedNow)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}