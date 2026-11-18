package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/validator"
)

type fixture struct {
	svc       *AuthService
	users     *MockUserStore
	codeStore *MockCodeStore
	mailer    *MockMailer
	tokens    *MockTokenStore
	issuer    *MockTokenIssuer
}

func newFixture() *fixture {
	f := &fixture{
		users:     &MockUserStore{},
		codeStore: &MockCodeStore{},
		mailer:    &MockMailer{},
		tokens:    &MockTokenStore{},
		issuer:    &MockTokenIssuer{},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	f.svc = New(f.users, f.codeStore, f.mailer, f.tokens, f.issuer, log)
	return f
}

func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, want, status.Code(err))
}

func hash(t *testing.T, plain string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(b)
}

func registerRequest() *authpb.RegisterRequest {
	return &authpb.RegisterRequest{
		Name:              "Adrian Yu",
		Username:          "adrian_yu",
		Email:             "Adrian.Yu@Gmail.com",
		Password:          "Abcdef1!",
		Gender:            "male",
		DateOfBirth:       "2000-01-15",
		ProfilePictureUrl: "https://example.com/p.png",
		BannerUrl:         "https://example.com/b.png",
		SecurityQuestion:  validator.SecurityQuestions[0],
		SecurityAnswer:    "Bobby",
	}
}

func TestRegisterSuccess(t *testing.T) {
	f := newFixture()
	ctx := context.Background()

	f.users.On("Create", mock.Anything, mock.MatchedBy(func(u repository.NewUser) bool {
		return u.Email == "adrian.yu@gmail.com" &&
			u.PasswordHash != "Abcdef1!" &&
			u.SecurityAnswerHash != "Bobby"
	})).Return("user-1", nil)
	f.codeStore.On("Save", mock.Anything, "adrian.yu@gmail.com", mock.Anything, CodeTTL).Return(nil)
	f.mailer.On("SendVerification", mock.Anything, "adrian.yu@gmail.com", mock.Anything).Return(nil)

	res, err := f.svc.Register(ctx, registerRequest())

	require.NoError(t, err)
	assert.Equal(t, "user-1", res.UserId)
	f.users.AssertExpectations(t)
	f.codeStore.AssertExpectations(t)
	f.mailer.AssertExpectations(t)
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	f := newFixture()
	req := registerRequest()
	req.Password = "weak"

	_, err := f.svc.Register(context.Background(), req)

	requireCode(t, err, codes.InvalidArgument)
	f.users.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestRegisterDuplicates(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"email taken", repository.ErrEmailTaken},
		{"username taken", repository.ErrUsernameTaken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			f.users.On("Create", mock.Anything, mock.Anything).Return("", tt.err)

			_, err := f.svc.Register(context.Background(), registerRequest())

			requireCode(t, err, codes.AlreadyExists)
			f.mailer.AssertNotCalled(t, "SendVerification", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("correct code activates the account", func(t *testing.T) {
		f := newFixture()
		f.codeStore.On("Get", mock.Anything, "adrian.yu@gmail.com").Return("123456", nil)
		f.users.On("Activate", mock.Anything, "adrian.yu@gmail.com").Return(nil)
		f.codeStore.On("Delete", mock.Anything, "adrian.yu@gmail.com").Return(nil)
		f.mailer.On("SendWelcome", mock.Anything, "adrian.yu@gmail.com").Return(nil)

		_, err := f.svc.VerifyEmail(ctx, &authpb.VerifyEmailRequest{Email: "adrian.yu@gmail.com", Code: " 123456 "})

		require.NoError(t, err)
		f.users.AssertExpectations(t)
		f.codeStore.AssertExpectations(t)
		f.mailer.AssertExpectations(t)
	})

	t.Run("expired code", func(t *testing.T) {
		f := newFixture()
		f.codeStore.On("Get", mock.Anything, "adrian.yu@gmail.com").Return("", nil)

		_, err := f.svc.VerifyEmail(ctx, &authpb.VerifyEmailRequest{Email: "adrian.yu@gmail.com", Code: "123456"})

		requireCode(t, err, codes.FailedPrecondition)
		f.users.AssertNotCalled(t, "Activate", mock.Anything, mock.Anything)
	})

	t.Run("wrong code", func(t *testing.T) {
		f := newFixture()
		f.codeStore.On("Get", mock.Anything, "adrian.yu@gmail.com").Return("123456", nil)

		_, err := f.svc.VerifyEmail(ctx, &authpb.VerifyEmailRequest{Email: "adrian.yu@gmail.com", Code: "000000"})

		requireCode(t, err, codes.InvalidArgument)
		f.users.AssertNotCalled(t, "Activate", mock.Anything, mock.Anything)
	})
}

func TestResendCode(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown email", func(t *testing.T) {
		f := newFixture()
		f.users.On("Status", mock.Anything, "x@gmail.com").Return(false, false, nil)

		_, err := f.svc.ResendCode(ctx, &authpb.ResendCodeRequest{Email: "x@gmail.com"})

		requireCode(t, err, codes.NotFound)
	})

	t.Run("already active", func(t *testing.T) {
		f := newFixture()
		f.users.On("Status", mock.Anything, "x@gmail.com").Return(true, true, nil)

		_, err := f.svc.ResendCode(ctx, &authpb.ResendCodeRequest{Email: "x@gmail.com"})

		requireCode(t, err, codes.FailedPrecondition)
	})

	t.Run("inactive account gets a new code", func(t *testing.T) {
		f := newFixture()
		f.users.On("Status", mock.Anything, "x@gmail.com").Return(true, false, nil)
		f.codeStore.On("Save", mock.Anything, "x@gmail.com", mock.Anything, CodeTTL).Return(nil)
		f.mailer.On("SendVerification", mock.Anything, "x@gmail.com", mock.Anything).Return(nil)

		_, err := f.svc.ResendCode(ctx, &authpb.ResendCodeRequest{Email: "x@gmail.com"})

		require.NoError(t, err)
		f.mailer.AssertExpectations(t)
	})
}

func TestLoginSuccess(t *testing.T) {
	f := newFixture()
	f.users.On("FindCredentials", mock.Anything, "adrian.yu@gmail.com").Return(
		repository.Credentials{ID: "u1", PasswordHash: hash(t, "Abcdef1!"), IsActive: true}, nil)
	f.issuer.On("NewAccess", "u1").Return("access-token", nil)
	f.issuer.On("NewRefresh", "u1").Return("refresh-token", "jti-1", nil)
	f.issuer.On("RefreshTTL").Return(time.Hour)
	f.tokens.On("Save", mock.Anything, "u1", "jti-1", time.Hour).Return(nil)

	res, err := f.svc.Login(context.Background(), &authpb.LoginRequest{Email: "Adrian.Yu@gmail.com", Password: "Abcdef1!"})

	require.NoError(t, err)
	assert.Equal(t, "access-token", res.AccessToken)
	assert.Equal(t, "refresh-token", res.RefreshToken)
	f.tokens.AssertExpectations(t)
}

func TestLoginFailures(t *testing.T) {
	good := hash(t, "Abcdef1!")

	tests := []struct {
		name     string
		cred     repository.Credentials
		findErr  error
		password string
		want     codes.Code
		message  string
	}{
		{"wrong password", repository.Credentials{ID: "u1", PasswordHash: good, IsActive: true}, nil, "Wrong123!", codes.Unauthenticated, "invalid email or password"},
		{"unknown email", repository.Credentials{}, repository.ErrNotFound, "Abcdef1!", codes.Unauthenticated, "invalid email or password"},
		{"not verified", repository.Credentials{ID: "u1", PasswordHash: good}, nil, "Abcdef1!", codes.PermissionDenied, "account is not verified"},
		{"banned", repository.Credentials{ID: "u1", PasswordHash: good, IsActive: true, IsBanned: true}, nil, "Abcdef1!", codes.PermissionDenied, "account is banned"},
		{"deactivated", repository.Credentials{ID: "u1", PasswordHash: good, IsActive: true, IsDeactivated: true}, nil, "Abcdef1!", codes.PermissionDenied, "account is deactivated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			f.users.On("FindCredentials", mock.Anything, "adrian.yu@gmail.com").Return(tt.cred, tt.findErr)

			_, err := f.svc.Login(context.Background(), &authpb.LoginRequest{Email: "adrian.yu@gmail.com", Password: tt.password})

			requireCode(t, err, tt.want)
			assert.Equal(t, tt.message, status.Convert(err).Message())
			f.issuer.AssertNotCalled(t, "NewAccess", mock.Anything)
		})
	}
}

func TestLoginRequiresEmailAndPassword(t *testing.T) {
	f := newFixture()

	_, err := f.svc.Login(context.Background(), &authpb.LoginRequest{})

	requireCode(t, err, codes.InvalidArgument)
}

func TestRefreshTokenRotation(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token is rotated", func(t *testing.T) {
		f := newFixture()
		f.issuer.On("ParseRefresh", "old").Return("u1", "jti-1", nil)
		f.tokens.On("Consume", mock.Anything, "u1", "jti-1").Return(true, nil)
		f.users.On("FindByID", mock.Anything, "u1").Return(repository.Credentials{ID: "u1", IsActive: true}, nil)
		f.issuer.On("NewAccess", "u1").Return("access-2", nil)
		f.issuer.On("NewRefresh", "u1").Return("refresh-2", "jti-2", nil)
		f.issuer.On("RefreshTTL").Return(time.Hour)
		f.tokens.On("Save", mock.Anything, "u1", "jti-2", time.Hour).Return(nil)

		res, err := f.svc.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: "old"})

		require.NoError(t, err)
		assert.Equal(t, "refresh-2", res.RefreshToken)
		f.tokens.AssertExpectations(t)
	})

	t.Run("reused token is rejected", func(t *testing.T) {
		f := newFixture()
		f.issuer.On("ParseRefresh", "old").Return("u1", "jti-1", nil)
		f.tokens.On("Consume", mock.Anything, "u1", "jti-1").Return(false, nil)

		_, err := f.svc.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: "old"})

		requireCode(t, err, codes.Unauthenticated)
		f.issuer.AssertNotCalled(t, "NewAccess", mock.Anything)
	})

	t.Run("banned user cannot refresh", func(t *testing.T) {
		f := newFixture()
		f.issuer.On("ParseRefresh", "old").Return("u1", "jti-1", nil)
		f.tokens.On("Consume", mock.Anything, "u1", "jti-1").Return(true, nil)
		f.users.On("FindByID", mock.Anything, "u1").Return(repository.Credentials{ID: "u1", IsActive: true, IsBanned: true}, nil)

		_, err := f.svc.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: "old"})

		requireCode(t, err, codes.PermissionDenied)
	})

	t.Run("garbage token", func(t *testing.T) {
		f := newFixture()
		f.issuer.On("ParseRefresh", "bad").Return("", "", assert.AnError)

		_, err := f.svc.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: "bad"})

		requireCode(t, err, codes.Unauthenticated)
	})
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	f := newFixture()
	f.issuer.On("ParseRefresh", "tok").Return("u1", "jti-1", nil)
	f.tokens.On("Consume", mock.Anything, "u1", "jti-1").Return(true, nil)

	res, err := f.svc.Logout(context.Background(), &authpb.LogoutRequest{RefreshToken: "tok"})

	require.NoError(t, err)
	assert.Equal(t, "logged out", res.Message)
	f.tokens.AssertExpectations(t)
}