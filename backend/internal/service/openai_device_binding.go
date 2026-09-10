package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrOpenAIDeviceIDRequired = errors.New("OpenAI device ID is required")
	ErrOpenAIDeviceLimit      = errors.New("OpenAI account device limit reached")
	ErrOpenAIDeviceBoundAway  = errors.New("OpenAI device is bound to another unavailable account")
)

type openAIDeviceIDContextKey struct{}
type openAIDevicePrincipalContextKey struct{}

type OpenAIDevicePrincipal struct {
	UserID     int64
	APIKeyID   int64
	APIKeyName string
}

// WithOpenAIDevicePrincipal stores the authenticated owner metadata shown in
// the admin device-binding view.
func WithOpenAIDevicePrincipal(ctx context.Context, principal OpenAIDevicePrincipal) context.Context {
	return context.WithValue(ctx, openAIDevicePrincipalContextKey{}, principal)
}

func openAIDevicePrincipalFromContext(ctx context.Context) OpenAIDevicePrincipal {
	if ctx == nil {
		return OpenAIDevicePrincipal{}
	}
	principal, _ := ctx.Value(openAIDevicePrincipalContextKey{}).(OpenAIDevicePrincipal)
	return principal
}

// WithOpenAIDeviceID stores the downstream installation identity used to keep
// one physical client pinned to one upstream OpenAI subscription account.
func WithOpenAIDeviceID(ctx context.Context, deviceID string) context.Context {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ctx
	}
	return context.WithValue(ctx, openAIDeviceIDContextKey{}, deviceID)
}

// OpenAIDeviceIDFromContext returns the normalized downstream installation ID.
func OpenAIDeviceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	deviceID, _ := ctx.Value(openAIDeviceIDContextKey{}).(string)
	return strings.TrimSpace(deviceID)
}

type OpenAIDeviceBindingClaim struct {
	Allowed        bool
	CapacityFull   bool
	BoundAccountID int64
}
type OpenAIDeviceBinding struct {
	DeviceHash     string    `json:"device_hash"`
	AccountID      int64     `json:"account_id"`
	AccountName    string    `json:"account_name"`
	UserID         int64     `json:"user_id"`
	UserEmail      string    `json:"user_email"`
	UserName       string    `json:"user_name"`
	APIKeyID       int64     `json:"api_key_id"`
	APIKeyName     string    `json:"api_key_name"`
	FirstSeenAt    time.Time `json:"first_seen_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	MaxDeviceCount int       `json:"max_device_count"`
}

// OpenAIDeviceBindingCache provides the atomic Redis operation used by the
// scheduler. It is deliberately separate from GatewayCache so older test
// doubles and alternate cache implementations remain source compatible while
// the feature is disabled.
type OpenAIDeviceBindingCache interface {
	ClaimOpenAIDeviceBinding(ctx context.Context, deviceHash string, accountID int64, principal OpenAIDevicePrincipal, maxDevices int, idleTTL time.Duration) (OpenAIDeviceBindingClaim, error)
	ListOpenAIDeviceBindings(ctx context.Context, maxDevices int) ([]OpenAIDeviceBinding, error)
	DeleteOpenAIDeviceBinding(ctx context.Context, deviceHash string) error
}

func hashOpenAIDeviceID(deviceID string) string {
	sum := sha256.Sum256([]byte("sub2api:openai-device-binding:v1:" + deviceID))
	return fmt.Sprintf("%x", sum[:])
}

func (s *OpenAIGatewayService) openAIDeviceBindingEnabled(platform string) bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIDeviceBinding.Enabled && NormalizeOpenAICompatiblePlatform(platform) == PlatformOpenAI
}

func (s *OpenAIGatewayService) claimOpenAIDeviceBinding(ctx context.Context, account *Account) (OpenAIDeviceBindingClaim, error) {
	if account == nil || !account.IsOpenAIOAuthLike() {
		return OpenAIDeviceBindingClaim{Allowed: true}, nil
	}
	deviceID := OpenAIDeviceIDFromContext(ctx)
	if deviceID == "" {
		return OpenAIDeviceBindingClaim{}, ErrOpenAIDeviceIDRequired
	}
	cache, ok := s.cache.(OpenAIDeviceBindingCache)
	if !ok {
		return OpenAIDeviceBindingClaim{}, errors.New("OpenAI device binding cache unavailable")
	}
	cfg := s.cfg.Gateway.OpenAIDeviceBinding
	claim, err := cache.ClaimOpenAIDeviceBinding(
		ctx,
		hashOpenAIDeviceID(deviceID),
		account.ID,
		openAIDevicePrincipalFromContext(ctx),
		cfg.MaxDevicesPerAccount,
		time.Duration(cfg.IdleTTLDays)*24*time.Hour,
	)
	if err != nil {
		return OpenAIDeviceBindingClaim{}, fmt.Errorf("claim OpenAI device binding: %w", err)
	}
	return claim, nil
}

func (s *OpenAIGatewayService) ListOpenAIDeviceBindings(ctx context.Context) ([]OpenAIDeviceBinding, error) {
	if s == nil || s.cfg == nil || !s.cfg.Gateway.OpenAIDeviceBinding.Enabled {
		return []OpenAIDeviceBinding{}, nil
	}
	cache, ok := s.cache.(OpenAIDeviceBindingCache)
	if !ok {
		return nil, errors.New("OpenAI device binding cache unavailable")
	}
	bindings, err := cache.ListOpenAIDeviceBindings(ctx, s.cfg.Gateway.OpenAIDeviceBinding.MaxDevicesPerAccount)
	if err != nil {
		return nil, fmt.Errorf("list OpenAI device bindings: %w", err)
	}
	accountNames := make(map[int64]string)
	userDetails := make(map[int64]*User)
	for i := range bindings {
		binding := &bindings[i]
		if name, found := accountNames[binding.AccountID]; found {
			binding.AccountName = name
		} else if account, loadErr := s.accountRepo.GetByID(ctx, binding.AccountID); loadErr == nil && account != nil {
			accountNames[binding.AccountID] = account.Name
			binding.AccountName = account.Name
		}
		if binding.UserID <= 0 {
			continue
		}
		user, found := userDetails[binding.UserID]
		if !found {
			user, _ = s.userRepo.GetByIDIncludeDeleted(ctx, binding.UserID)
			userDetails[binding.UserID] = user
		}
		if user != nil {
			binding.UserEmail = user.Email
			binding.UserName = user.Username
		}
	}
	return bindings, nil
}

func (s *OpenAIGatewayService) DeleteOpenAIDeviceBinding(ctx context.Context, deviceHash string) error {
	deviceHash = strings.TrimSpace(deviceHash)
	if len(deviceHash) != sha256.Size*2 {
		return errors.New("invalid OpenAI device hash")
	}
	cache, ok := s.cache.(OpenAIDeviceBindingCache)
	if !ok {
		return errors.New("OpenAI device binding cache unavailable")
	}
	if err := cache.DeleteOpenAIDeviceBinding(ctx, deviceHash); err != nil {
		return fmt.Errorf("delete OpenAI device binding: %w", err)
	}
	return nil
}
