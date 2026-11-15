package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

func TestGetSecurityQuestion(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the question", func(t *testing.T) {
		f := newFixture()
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(repository.Recovery{Question: "Q?"}, nil)

		res, err := f.svc.GetSecurityQuestion(ctx, &authpb.GetSecurityQuestionRequest{Email: "a@gmail.com"})

		require.NoError(t, err)
		assert.Equal(t, "Q?", res.Question)
	})

	t.Run("unknown email", func(t *testing.T) {
		f := newFixture()
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(repository.Recovery{}, repository.ErrNotFound)

		_, err := f.svc.GetSecurityQuestion(ctx, &authpb.GetSecurityQuestionRequest{Email: "a@gmail.com"})

		requireCode(t, err, codes.NotFound)
	})

	t.Run("banned account", func(t *testing.T) {
		f := newFixture()
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(repository.Recovery{IsBanned: true}, nil)

		_, err := f.svc.GetSecurityQuestion(ctx, &authpb.GetSecurityQuestionRequest{Email: "a@gmail.com"})

		requireCode(t, err, codes.PermissionDenied)
	})
}

func TestVerifySecurityAnswer(t *testing.T) {
	ctx := context.Background()
	rec := func(t *testing.T) repository.Recovery {
		return repository.Recovery{ID: "u1", AnswerHash: hash(t, "bobby")}
	}

	t.Run("correct answer issues a reset token", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("Attempts", mock.Anything, "a@gmail.com").Return(int64(0), nil)
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(rec(t), nil)
		f.tokens.On("SaveReset", mock.Anything, "a@gmail.com", mock.Anything, resetTTL).Return(nil)
		f.tokens.On("ClearAttempts", mock.Anything, "a@gmail.com").Return(nil)

		res, err := f.svc.VerifySecurityAnswer(ctx, &authpb.VerifySecurityAnswerRequest{Email: "a@gmail.com", Answer: "  BOBBY "})

		require.NoError(t, err)
		assert.Len(t, res.ResetToken, 64)
		f.tokens.AssertExpectations(t)
	})

	t.Run("wrong answer is counted", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("Attempts", mock.Anything, "a@gmail.com").Return(int64(1), nil)
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(rec(t), nil)
		f.tokens.On("AddAttempt", mock.Anything, "a@gmail.com", attemptWindow).Return(int64(2), nil)

		_, err := f.svc.VerifySecurityAnswer(ctx, &authpb.VerifySecurityAnswerRequest{Email: "a@gmail.com", Answer: "rex"})

		requireCode(t, err, codes.Unauthenticated)
		f.tokens.AssertExpectations(t)
		f.tokens.AssertNotCalled(t, "SaveReset", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("locked after too many attempts", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("Attempts", mock.Anything, "a@gmail.com").Return(int64(maxAnswerTries), nil)

		_, err := f.svc.VerifySecurityAnswer(ctx, &authpb.VerifySecurityAnswerRequest{Email: "a@gmail.com", Answer: "bobby"})

		requireCode(t, err, codes.ResourceExhausted)
		f.users.AssertNotCalled(t, "FindRecovery", mock.Anything, mock.Anything)
	})
}

func TestResetPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("weak password", func(t *testing.T) {
		f := newFixture()

		_, err := f.svc.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: "a@gmail.com", ResetToken: "t", NewPassword: "weak"})

		requireCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid reset token", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("CheckReset", mock.Anything, "a@gmail.com", "bad").Return(false, nil)

		_, err := f.svc.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: "a@gmail.com", ResetToken: "bad", NewPassword: "Newpass1!"})

		requireCode(t, err, codes.Unauthenticated)
		f.users.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("same as old password", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("CheckReset", mock.Anything, "a@gmail.com", "tok").Return(true, nil)
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(
			repository.Recovery{ID: "u1", PasswordHash: hash(t, "Abcdef1!")}, nil)

		_, err := f.svc.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: "a@gmail.com", ResetToken: "tok", NewPassword: "Abcdef1!"})

		requireCode(t, err, codes.InvalidArgument)
		assert.Equal(t, "new password cannot be the same as the old one", status.Convert(err).Message())
		f.users.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("success", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("CheckReset", mock.Anything, "a@gmail.com", "tok").Return(true, nil)
		f.users.On("FindRecovery", mock.Anything, "a@gmail.com").Return(
			repository.Recovery{ID: "u1", PasswordHash: hash(t, "Abcdef1!")}, nil)
		f.users.On("UpdatePassword", mock.Anything, "u1", mock.MatchedBy(func(h string) bool {
			return h != "" && h != "Newpass1!"
		})).Return(nil)
		f.tokens.On("DeleteReset", mock.Anything, "a@gmail.com").Return(nil)

		res, err := f.svc.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: "a@gmail.com", ResetToken: "tok", NewPassword: "Newpass1!"})

		require.NoError(t, err)
		assert.Equal(t, "password has been reset", res.Message)
		f.users.AssertExpectations(t)
		f.tokens.AssertExpectations(t)
	})
}