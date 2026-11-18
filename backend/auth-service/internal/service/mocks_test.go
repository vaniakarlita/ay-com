package service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

type MockUserStore struct{ mock.Mock }

func (m *MockUserStore) Create(ctx context.Context, u repository.NewUser) (string, error) {
	args := m.Called(ctx, u)
	return args.String(0), args.Error(1)
}

func (m *MockUserStore) Status(ctx context.Context, email string) (bool, bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Bool(1), args.Error(2)
}

func (m *MockUserStore) Activate(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *MockUserStore) FindCredentials(ctx context.Context, email string) (repository.Credentials, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(repository.Credentials), args.Error(1)
}

func (m *MockUserStore) FindByID(ctx context.Context, id string) (repository.Credentials, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(repository.Credentials), args.Error(1)
}

func (m *MockUserStore) FindRecovery(ctx context.Context, email string) (repository.Recovery, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(repository.Recovery), args.Error(1)
}

func (m *MockUserStore) UpdatePassword(ctx context.Context, id, hash string) error {
	return m.Called(ctx, id, hash).Error(0)
}

type MockCodeStore struct{ mock.Mock }

func (m *MockCodeStore) Save(ctx context.Context, email, code string, ttl time.Duration) error {
	return m.Called(ctx, email, code, ttl).Error(0)
}

func (m *MockCodeStore) Get(ctx context.Context, email string) (string, error) {
	args := m.Called(ctx, email)
	return args.String(0), args.Error(1)
}

func (m *MockCodeStore) Delete(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

type MockMailer struct{ mock.Mock }

func (m *MockMailer) SendVerification(ctx context.Context, to, code string) error {
	return m.Called(ctx, to, code).Error(0)
}

func (m *MockMailer) SendWelcome(ctx context.Context, to string) error {
	return m.Called(ctx, to).Error(0)
}

type MockTokenStore struct{ mock.Mock }

func (m *MockTokenStore) Save(ctx context.Context, userID, jti string, ttl time.Duration) error {
	return m.Called(ctx, userID, jti, ttl).Error(0)
}

func (m *MockTokenStore) Consume(ctx context.Context, userID, jti string) (bool, error) {
	args := m.Called(ctx, userID, jti)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenStore) SaveReset(ctx context.Context, email, token string, ttl time.Duration) error {
	return m.Called(ctx, email, token, ttl).Error(0)
}

func (m *MockTokenStore) CheckReset(ctx context.Context, email, token string) (bool, error) {
	args := m.Called(ctx, email, token)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenStore) DeleteReset(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *MockTokenStore) AddAttempt(ctx context.Context, email string, window time.Duration) (int64, error) {
	args := m.Called(ctx, email, window)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTokenStore) Attempts(ctx context.Context, email string) (int64, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTokenStore) ClearAttempts(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

type MockTokenIssuer struct{ mock.Mock }

func (m *MockTokenIssuer) NewAccess(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenIssuer) NewRefresh(userID string) (string, string, error) {
	args := m.Called(userID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTokenIssuer) ParseRefresh(raw string) (string, string, error) {
	args := m.Called(raw)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTokenIssuer) RefreshTTL() time.Duration {
	return m.Called().Get(0).(time.Duration)
}