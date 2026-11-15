package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/validator"
)

const (
	resetTTL        = 10 * time.Minute
	maxAnswerTries  = 5
	attemptWindow   = 15 * time.Minute
)

func (s *AuthService) findRecovery(ctx context.Context, email string) (repository.Recovery, error) {
	rec, err := s.users.FindRecovery(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return repository.Recovery{}, status.Error(codes.NotFound, "email is not registered")
	}
	if err != nil {
		s.log.Error("find recovery failed", "error", err)
		return repository.Recovery{}, status.Error(codes.Internal, "internal error")
	}
	if rec.IsBanned {
		return repository.Recovery{}, status.Error(codes.PermissionDenied, "account is banned")
	}
	return rec, nil
}

func (s *AuthService) GetSecurityQuestion(ctx context.Context, req *authpb.GetSecurityQuestionRequest) (*authpb.GetSecurityQuestionResponse, error) {
	rec, err := s.findRecovery(ctx, normalizeEmail(req.Email))
	if err != nil {
		return nil, err
	}
	return &authpb.GetSecurityQuestionResponse{Question: rec.Question}, nil
}

func (s *AuthService) VerifySecurityAnswer(ctx context.Context, req *authpb.VerifySecurityAnswerRequest) (*authpb.VerifySecurityAnswerResponse, error) {
	email := normalizeEmail(req.Email)

	tries, err := s.tokens.Attempts(ctx, email)
	if err != nil {
		s.log.Error("read attempts failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if tries >= maxAnswerTries {
		return nil, status.Error(codes.ResourceExhausted, "too many attempts, please try again later")
	}

	rec, err := s.findRecovery(ctx, email)
	if err != nil {
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(rec.AnswerHash), []byte(normalizeAnswer(req.Answer))) != nil {
		if _, err := s.tokens.AddAttempt(ctx, email, attemptWindow); err != nil {
			s.log.Error("record attempt failed", "error", err)
		}
		return nil, status.Error(codes.Unauthenticated, "incorrect answer")
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		s.log.Error("generate reset token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	resetToken := hex.EncodeToString(raw)

	if err := s.tokens.SaveReset(ctx, email, resetToken, resetTTL); err != nil {
		s.log.Error("save reset token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := s.tokens.ClearAttempts(ctx, email); err != nil {
		s.log.Warn("clear attempts failed", "error", err)
	}
	return &authpb.VerifySecurityAnswerResponse{ResetToken: resetToken}, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	email := normalizeEmail(req.Email)

	if err := validator.ValidatePassword(req.NewPassword); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	valid, err := s.tokens.CheckReset(ctx, email, req.ResetToken)
	if err != nil {
		s.log.Error("check reset token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !valid {
		return nil, status.Error(codes.Unauthenticated, "reset session expired, please start again")
	}

	rec, err := s.findRecovery(ctx, email)
	if err != nil {
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(req.NewPassword)) == nil {
		return nil, status.Error(codes.InvalidArgument, "new password cannot be the same as the old one")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hash password failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := s.users.UpdatePassword(ctx, rec.ID, string(hash)); err != nil {
		s.log.Error("update password failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := s.tokens.DeleteReset(ctx, email); err != nil {
		s.log.Warn("delete reset token failed", "error", err)
	}

	s.log.Info("password reset", "user_id", rec.ID)
	return &authpb.ResetPasswordResponse{Message: "password has been reset"}, nil
}