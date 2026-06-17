package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridego/services/user/internal/models"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

// A contract(Repository) with promises(methods). it does not care about the db used
// or how to interact with the Db. It just wants the promises fulfilled
type Repository interface {
	CreateUser(ctx context.Context, u *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error)
	SaveRefreshToken(ctx context.Context, rt *models.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
	DeleteExpiredRefreshTokens(ctx context.Context) error
}

// Holds the db connection pool and how to it
type PostgresRepo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) Repository {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateUser(ctx context.Context, u *models.User) error {
	// Implementation to insert user into PostgreSQL
	u.ID = uuid.New()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	u.Rating = 0.0
	u.IsActive = true

	_, err := r.pool.Exec(ctx,
		`INSERT INTO users(id, email, password_hash, role, full_name, phone, rating, is_active, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.FullName, u.Phone, u.Rating, u.IsActive, u.CreatedAt, u.UpdatedAt,
	)

	if err != nil && isDuplicateEmail(err) {
		return ErrEmailTaken
	}

	return err
}

func (r *PostgresRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, full_name, phone, avatar_url, rating, is_active, created_at FROM users WHERE email=$1 and is_active=true`,
		email)

	return scanUser(row)
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, full_name, phone, avatar_url, rating, is_active, created_at FROM users WHERE id=$1`,
		id)

	return scanUser(row)
}

func (r *PostgresRepo) UpdateUser(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE users SET full_name = COALESCE($1, full_name), phone = COALESCE($2, phone), avatar_url = COALESCE($3, avatar_url), updated_at = $4
		 WHERE id = $5 RETURNING id, email, password_hash, role, full_name, phone, avatar_url, rating, is_active, created_at`,
		req.FullName, req.Phone, req.AvatarURL, time.Now().UTC(), id)

	return scanUser(row)
}

func (r *PostgresRepo) SaveRefreshToken(ctx context.Context, rt *models.RefreshToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens(id, user_id, token_hash, expires_at, created_at) VALUES ($1,$2,$3,$4,$5)`,
		rt.ID, rt.UserID, rt.TokenHash, rt.ExpiresAt, rt.CreatedAt,
	)
	return err
}

func (r *PostgresRepo) GetRefreshToken(ctx context.Context, raw string) (*models.RefreshToken, error) {
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
	row := r.pool.QueryRow(ctx,
		`SELECT id,user_id,token_hash,expires_at,created_at FROM refresh_tokens
         WHERE token_hash=$1 AND expires_at > NOW()`, h)
	rt := &models.RefreshToken{}
	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rt, err
}

func (r *PostgresRepo) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE id=$1`, id)
	return err
}

func (r *PostgresRepo) DeleteExpiredRefreshTokens(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
	return err
}

func scanUser(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role,
		&u.FullName, &u.Phone, &u.AvatarURL, &u.Rating, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func isDuplicateEmail(err error) bool {
	return strings.Contains(err.Error(), "unique") &&
		strings.Contains(err.Error(), "email")
}
