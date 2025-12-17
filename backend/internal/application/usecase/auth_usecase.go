package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/domain/repository"
	"github.com/fintech-multipass/backend/internal/infrastructure/config"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthUseCase casos de uso para autenticación
type AuthUseCase struct {
	userRepo repository.UserRepository
	jwtCfg   config.JWTConfig
	log      *logger.Logger
}

// NewAuthUseCase crea una nueva instancia del caso de uso
func NewAuthUseCase(userRepo repository.UserRepository, jwtCfg config.JWTConfig, log *logger.Logger) *AuthUseCase {
	return &AuthUseCase{
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
		log:      log,
	}
}

// LoginInput datos de entrada para login
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginOutput resultado del login
type LoginOutput struct {
	User         *entity.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// RegisterInput datos de entrada para registro
type RegisterInput struct {
	Email      string          `json:"email" binding:"required,email"`
	Password   string          `json:"password" binding:"required,min=6"`
	FullName   string          `json:"full_name" binding:"required,min=3"`
	Role       entity.UserRole `json:"role,omitempty"`
	CountryIDs []uuid.UUID     `json:"country_ids,omitempty"`
}

// Login autentica a un usuario
func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. Buscar usuario por email
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		uc.log.Warn().Str("email", input.Email).Msg("Login attempt with invalid email")
		return nil, fmt.Errorf("invalid credentials")
	}

	// 2. Verificar si el usuario está activo
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// 3. Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		uc.log.Warn().Str("email", input.Email).Msg("Login attempt with invalid password")
		return nil, fmt.Errorf("invalid credentials")
	}

	// 4. Generar tokens
	accessToken, expiresAt, err := uc.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := uc.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 5. Actualizar último login
	_ = uc.userRepo.UpdateLastLogin(ctx, user.ID)

	// 6. Limpiar datos sensibles
	user.PasswordHash = ""

	uc.log.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("User logged in successfully")

	return &LoginOutput{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Register registra un nuevo usuario
func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*entity.User, error) {
	// 1. Verificar si el email ya existe
	existing, _ := uc.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	// 2. Hash de la contraseña
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 3. Crear usuario
	role := input.Role
	if role == "" {
		role = entity.RoleViewer
	}

	user := &entity.User{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: string(passwordHash),
		FullName:     input.FullName,
		Role:         role,
		CountryIDs:   input.CountryIDs,
		IsActive:     true,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 4. Limpiar datos sensibles
	user.PasswordHash = ""

	uc.log.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("User registered successfully")

	return user, nil
}

// RefreshToken refresca el token de acceso
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*LoginOutput, error) {
	// 1. Validar refresh token
	claims, err := uc.validateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// 2. Obtener usuario
	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// 3. Generar nuevo access token
	accessToken, expiresAt, err := uc.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 4. Limpiar datos sensibles
	user.PasswordHash = ""

	return &LoginOutput{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Reutilizar el refresh token
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateToken valida un token y retorna los claims
func (uc *AuthUseCase) ValidateToken(tokenString string) (*entity.TokenClaims, error) {
	return uc.validateToken(tokenString)
}

// GetUserByID obtiene un usuario por ID
func (uc *AuthUseCase) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

// ChangePassword cambia la contraseña de un usuario
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// 1. Obtener usuario
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// 2. Verificar contraseña actual
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}

	// 3. Validar nueva contraseña
	if len(newPassword) < 6 {
		return fmt.Errorf("new password must be at least 6 characters")
	}

	// 4. Hash de la nueva contraseña
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 5. Actualizar en base de datos
	if err := uc.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	uc.log.Info().
		Str("user_id", userID.String()).
		Msg("Password changed successfully")

	return nil
}

// Métodos privados

type jwtClaims struct {
	UserID     uuid.UUID       `json:"user_id"`
	Email      string          `json:"email"`
	Role       entity.UserRole `json:"role"`
	CountryIDs []uuid.UUID     `json:"country_ids,omitempty"`
	jwt.RegisteredClaims
}

func (uc *AuthUseCase) generateAccessToken(user *entity.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(uc.jwtCfg.AccessExpiry)

	claims := jwtClaims{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       user.Role,
		CountryIDs: user.CountryIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    uc.jwtCfg.Issuer,
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(uc.jwtCfg.Secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (uc *AuthUseCase) generateRefreshToken(user *entity.User) (string, error) {
	expiresAt := time.Now().Add(uc.jwtCfg.RefreshExpiry)

	claims := jwtClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    uc.jwtCfg.Issuer,
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtCfg.Secret))
}

func (uc *AuthUseCase) validateToken(tokenString string) (*entity.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(uc.jwtCfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwtClaims); ok && token.Valid {
		return &entity.TokenClaims{
			UserID:     claims.UserID,
			Email:      claims.Email,
			Role:       claims.Role,
			CountryIDs: claims.CountryIDs,
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}
