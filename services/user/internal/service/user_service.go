package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ridego/services/user/internal/models"
	"github.com/ridego/services/user/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken   = fmt.Errorf("email already registered")
	ErrInvalidCreds = fmt.Errorf("invalid credentials")
	ErrInvalidToken = fmt.Errorf("invalid or expired token")
)

type UserService interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	RefreshTokens(ctx context.Context, rawRefresh string) (*models.AuthResponse, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error)
}

type userService struct {
	repo repository.Repository
	jwt  *JWTService
}

func New(repo repository.Repository, jwt *JWTService) UserService {
	return &userService{repo: repo, jwt: jwt}
}

func (s *userService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Phone:        req.Phone,
		Role:         req.Role,
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueTokenPair(ctx, u)
}

func (s *userService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	u, err := s.repo.GetUserByEmail(ctx, req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCreds
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}
	return s.issueTokenPair(ctx, u)
}

func (s *userService) RefreshTokens(ctx context.Context, raw string) (*models.AuthResponse, error) {
	rt, err := s.repo.GetRefreshToken(ctx, raw)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}

	// Token rotation: delete old, issue new pair
	_ = s.repo.DeleteRefreshToken(ctx, rt.ID)

	u, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, u)
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
	return s.repo.UpdateUser(ctx, id, req)
}

func (s *userService) issueTokenPair(ctx context.Context, u *models.User) (*models.AuthResponse, error) {
	const accessTTL = 15 * time.Minute
	const refreshTTL = 30 * 24 * time.Hour

	access, err := s.jwt.Sign(u, accessTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh := uuid.New().String() // opaque token, not a JWT
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(rawRefresh)))

	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: h,
		ExpiresAt: time.Now().Add(refreshTTL),
		CreatedAt: time.Now(),
	}
	if err := s.repo.SaveRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         u,
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTTL.Seconds()),
	}, nil
}
