package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const userKey = contextKey("user")

func NewContextWithUser(ctx context.Context, claims *AccessTokenClaims) context.Context {
	return context.WithValue(ctx, userKey, claims)
}

func UserFromContext(ctx context.Context) (*AccessTokenClaims, bool) {
	claims, ok := ctx.Value(userKey).(*AccessTokenClaims)
	return claims, ok
}

const (
	BasicPlan int64 = 1 << iota
	ProPlan   int64 = 1 << iota

	// GF_ACCESS_FLAGS_START
	// Model-specific access flags (append new ones below)
	GetSkeletons   int64 = 1 << iota
	CreateSkeleton int64 = 1 << iota
	EditSkeleton   int64 = 1 << iota
	RemoveSkeleton int64 = 1 << iota
	// GF_ACCESS_FLAGS_END

	// Subscription
	CreateCheckout int64 = 1 << iota
	CreatePortal   int64 = 1 << iota
	// Files
	GetFiles     int64 = 1 << iota
	UploadFiles  int64 = 1 << iota
	DownloadFile int64 = 1 << iota
	RemoveFile   int64 = 1 << iota
	// Emails
	GetEmails int64 = 1 << iota
	SendEmail int64 = 1 << iota
)

const UserAccess int64 = CreateCheckout | CreatePortal | GetFiles | UploadFiles | DownloadFile | RemoveFile | GetEmails | SendEmail |
	// GF_USER_ACCESS_START
	GetSkeletons | CreateSkeleton | EditSkeleton | RemoveSkeleton

// GF_USER_ACCESS_END
// Note: BasicPlan and ProPlan are added dynamically based on subscription status

const AdminAccess int64 = -1

func HasAccess(required int64, userAccess int64) bool {
	if userAccess == AdminAccess {
		return true
	}
	if userAccess == 0 {
		return false
	}
	return userAccess&required == required
}

func UpdateAccess(userAccess int64, access int64) int64 {
	return userAccess | access
}

var (
	ErrNoClaims           = errors.New("no claims found in context")
	ErrInsufficientAccess = errors.New("insufficient permissions")
)

func Authorize(ctx context.Context, span trace.Span, requiredAccess int64) (*AccessTokenClaims, error) {
	claims, ok := UserFromContext(ctx)
	if !ok {
		return nil, ErrNoClaims
	}
	if !HasAccess(requiredAccess, claims.Access) {
		return nil, ErrInsufficientAccess
	}
	span.AddEvent("Authorization successful")
	return claims, nil
}

func GenerateTokens(
	refreshTokenID string,
	userID string,
	access int64,
	avatar string,
	email string,
	accessTokenExp time.Duration,
	refreshTokenExp time.Duration,
) (string, string, error) {
	privateKeyParsed, err := loadPrivateKey()
	if err != nil {
		return "", "", err
	}

	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, jwt.MapClaims{
		"id":     userID,
		"access": access,
		"avatar": avatar,
		"email":  email,
		"exp":    time.Now().Add(accessTokenExp).Unix(),
	})
	tokenString, err := token.SignedString(privateKeyParsed)
	if err != nil {
		return "", "", fmt.Errorf("error signing token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, jwt.MapClaims{
		"id":      refreshTokenID,
		"user_id": userID,
		"exp":     time.Now().Add(refreshTokenExp).Unix(),
	})
	refreshTokenString, err := refreshToken.SignedString(privateKeyParsed)
	if err != nil {
		return "", "", fmt.Errorf("error signing refresh token: %w", err)
	}

	return tokenString, refreshTokenString, nil
}

type AccessTokenClaims struct {
	ID     uuid.UUID `json:"id"`
	Access int64     `json:"access"`
	Avatar string    `json:"avatar"`
	Email  string    `json:"email"`
}

func ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims, err := extractTokenClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	accessTokenClaims, err := extractAccessTokenClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	return accessTokenClaims, nil
}

type RefreshTokenClaims struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
}

func ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	claims, err := extractTokenClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	refreshTokenClaims, err := extractRefreshTokenClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	return refreshTokenClaims, nil
}

type SessionTokenClaims struct {
	ID    uuid.UUID `json:"id"`
	Phone string    `json:"phone"`
}

func GenerateSessionToken(userID string, phone string, sessionTokenExp time.Duration) (string, error) {
	privateKeyParsed, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, jwt.MapClaims{
		"id":    userID,
		"phone": phone,
		"exp":   time.Now().Add(sessionTokenExp).Unix(),
	})
	tokenString, err := token.SignedString(privateKeyParsed)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}
	return tokenString, nil
}

func ValidateSessionToken(tokenString string) (*SessionTokenClaims, error) {
	claims, err := extractTokenClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	sessionTokenClaims, err := extractSessionTokenClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("error extracting claims: %w", err)
	}
	return sessionTokenClaims, nil
}

// --- Private helpers ---

func loadPrivateKey() (any, error) {
	privateKeyPath := os.Getenv("PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		privateKeyPath = "/private.pem"
	}
	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}
	if len(privateKey) == 0 {
		return nil, errors.New("error reading private key: empty file")
	}
	privateKeyParsed, err := jwt.ParseEdPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}
	return privateKeyParsed, nil
}

func extractTokenClaims(tokenString string) (jwt.MapClaims, error) {
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = tokenString[7:]
	}

	publicKeyPath := os.Getenv("PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		publicKeyPath = "/public.pem"
	}
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading public key: %w", err)
	}
	if len(publicKey) == 0 {
		return nil, errors.New("error reading public key: empty file")
	}

	publicKeyParsed, err := jwt.ParseEdPublicKeyFromPEM(publicKey)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (any, error) {
		return publicKeyParsed, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("error getting claims")
	}
	return claims, nil
}

var ErrInvalidClaims = errors.New("claims are invalid")

func extractRefreshTokenClaims(claims jwt.MapClaims) (*RefreshTokenClaims, error) {
	id, ok := claims["id"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'id' field: %w", ErrInvalidClaims)
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'user_id' field: %w", ErrInvalidClaims)
	}
	UUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("error parsing user ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("error parsing refresh token ID: %w", err)
	}
	return &RefreshTokenClaims{ID: UUID, UserID: userUUID}, nil
}

func extractAccessTokenClaims(claims jwt.MapClaims) (*AccessTokenClaims, error) {
	id, ok := claims["id"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'id' field: %w", ErrInvalidClaims)
	}
	access, ok := claims["access"].(float64)
	if !ok {
		return nil, fmt.Errorf("claims missing 'access' field: %w", ErrInvalidClaims)
	}
	avatar, ok := claims["avatar"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'avatar' field: %w", ErrInvalidClaims)
	}
	email, ok := claims["email"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'email' field: %w", ErrInvalidClaims)
	}
	UUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("error parsing user ID: %w", err)
	}
	return &AccessTokenClaims{
		ID:     UUID,
		Access: int64(access),
		Avatar: avatar,
		Email:  email,
	}, nil
}

func extractSessionTokenClaims(claims jwt.MapClaims) (*SessionTokenClaims, error) {
	id, ok := claims["id"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'id' field: %w", errors.New("invalid claims"))
	}
	UUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("error parsing user ID: %w", err)
	}
	phone, ok := claims["phone"].(string)
	if !ok {
		return nil, fmt.Errorf("claims missing 'phone' field: %w", errors.New("invalid claims"))
	}
	return &SessionTokenClaims{ID: UUID, Phone: phone}, nil
}
