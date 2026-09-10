package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGatewayCacheOpenAIDeviceBindingLimitPinAndExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache, ok := NewGatewayCache(client).(service.OpenAIDeviceBindingCache)
	require.True(t, ok)

	ctx := context.Background()
	const accountID int64 = 17
	const ttl = 30 * 24 * time.Hour
	principal := service.OpenAIDevicePrincipal{UserID: 9, APIKeyID: 12, APIKeyName: "omp-user"}
	for i := 1; i <= 5; i++ {
		claim, err := cache.ClaimOpenAIDeviceBinding(ctx, fmt.Sprintf("device-%d", i), accountID, principal, 5, ttl)
		require.NoError(t, err)
		require.True(t, claim.Allowed)
		require.Equal(t, accountID, claim.BoundAccountID)
	}

	full, err := cache.ClaimOpenAIDeviceBinding(ctx, "device-6", accountID, principal, 5, ttl)
	require.NoError(t, err)
	require.False(t, full.Allowed)
	require.True(t, full.CapacityFull)

	boundElsewhere, err := cache.ClaimOpenAIDeviceBinding(ctx, "device-1", 18, principal, 5, ttl)
	require.NoError(t, err)
	require.False(t, boundElsewhere.Allowed)
	require.Equal(t, accountID, boundElsewhere.BoundAccountID)

	server.FastForward(5 * 24 * time.Hour)
	refreshed, err := cache.ClaimOpenAIDeviceBinding(ctx, "device-1", accountID, principal, 5, ttl)
	require.NoError(t, err)
	require.True(t, refreshed.Allowed)
	require.Greater(t, server.TTL(buildOpenAIDeviceBindingKey("device-1")), 29*24*time.Hour)

	server.FastForward(31 * 24 * time.Hour)
	released, err := cache.ClaimOpenAIDeviceBinding(ctx, "device-6", accountID, principal, 5, ttl)
	require.NoError(t, err)
	require.True(t, released.Allowed)
}

func TestGatewayCacheOpenAIDeviceBindingListAndDelete(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewGatewayCache(client).(service.OpenAIDeviceBindingCache)
	ctx := context.Background()
	principal := service.OpenAIDevicePrincipal{UserID: 7, APIKeyID: 8, APIKeyName: "alice-omp"}

	_, err := cache.ClaimOpenAIDeviceBinding(ctx, "device-hash", 17, principal, 5, 30*24*time.Hour)
	require.NoError(t, err)
	bindings, err := cache.ListOpenAIDeviceBindings(ctx, 5)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, "device-hash", bindings[0].DeviceHash)
	require.Equal(t, int64(7), bindings[0].UserID)
	require.Equal(t, int64(8), bindings[0].APIKeyID)
	require.Equal(t, "alice-omp", bindings[0].APIKeyName)
	require.Equal(t, 5, bindings[0].MaxDeviceCount)

	require.NoError(t, cache.DeleteOpenAIDeviceBinding(ctx, "device-hash"))
	bindings, err = cache.ListOpenAIDeviceBindings(ctx, 5)
	require.NoError(t, err)
	require.Empty(t, bindings)
	require.Equal(t, int64(0), client.ZCard(ctx, buildOpenAIAccountDevicesKey(17)).Val())
}
