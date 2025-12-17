package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserRepository implementación de repositorio de usuarios
type UserRepository struct {
	db *database.PostgresDB
}

// NewUserRepository crea una nueva instancia del repositorio
func NewUserRepository(db *database.PostgresDB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID obtiene un usuario por ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, country_ids, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user entity.User
	row := r.db.QueryRow(ctx, query, id)
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Role, &user.CountryIDs, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail obtiene un usuario por email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, country_ids, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user entity.User
	row := r.db.QueryRow(ctx, query, email)
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Role, &user.CountryIDs, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Create crea un nuevo usuario
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, email, password_hash, full_name, role, country_ids, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	return r.db.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FullName,
		user.Role, user.CountryIDs, user.IsActive,
		user.CreatedAt, user.UpdatedAt,
	)
}

// Update actualiza un usuario
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = $2, full_name = $3, role = $4, country_ids = $5, is_active = $6, updated_at = $7
		WHERE id = $1
	`

	return r.db.Exec(ctx, query,
		user.ID, user.Email, user.FullName, user.Role,
		user.CountryIDs, user.IsActive, user.UpdatedAt,
	)
}

// UpdatePassword actualiza la contraseña de un usuario
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`
	return r.db.Exec(ctx, query, id, passwordHash)
}

// Delete elimina un usuario
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	return r.db.Exec(ctx, query, id)
}

// List lista usuarios con paginación
func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]entity.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Contar total
	var total int64
	countRow := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users")
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Obtener usuarios
	offset := (page - 1) * pageSize
	query := `
		SELECT id, email, password_hash, full_name, role, country_ids, is_active, last_login_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
			&user.Role, &user.CountryIDs, &user.IsActive,
			&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

// UpdateLastLogin actualiza la fecha de último login
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	return r.db.Exec(ctx, query, id)
}

