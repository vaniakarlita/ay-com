package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManager() *Manager {
	return NewManager("access-secret", "refresh-secret", 15*time.Minute, time.Hour)
}

func TestRefreshTokenRoundTrip(t *testing.T) {
	m := newManager()

	raw, jti, err := m.NewRefresh("user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, jti)

	userID, gotJTI, err := m.ParseRefresh(raw)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
	assert.Equal(t, jti, gotJTI)
}

func TestRefreshTokenIDsAreUnique(t *testing.T) {
	m := newManager()

	_, first, err := m.NewRefresh("user-1")
	require.NoError(t, err)
	_, second, err := m.NewRefresh("user-1")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestParseRefreshRejectsInvalidTokens(t *testing.T) {
	m := newManager()
	valid, _, err := m.NewRefresh("user-1")
	require.NoError(t, err)

	other := NewManager("access-secret", "another-secret", time.Minute, time.Hour)
	foreign, _, err := other.NewRefresh("user-1")
	require.NoError(t, err)

	expired := NewManager("access-secret", "refresh-secret", time.Minute, -time.Minute)
	old, _, err := expired.NewRefresh("user-1")
	require.NoError(t, err)

	access, err := m.NewAccess("user-1")
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"garbage", "not-a-token"},
		{"tampered", valid + "x"},
		{"signed with another secret", foreign},
		{"expired", old},
		{"access token used as refresh", access},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := m.ParseRefresh(tt.input)
			assert.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestAccessTokenIsNotEmpty(t *testing.T) {
	raw, err := newManager().NewAccess("user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}