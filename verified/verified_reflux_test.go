package verified

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ VerifiedRefluxProvider = (*TestVerifiedRefluxProvider)(nil)

type TestVerifiedRefluxProvider struct{}

func (t TestVerifiedRefluxProvider) Name() string { return "test_provider" }

func (t TestVerifiedRefluxProvider) GenerateUniqueId() string {
	return randString(6)
}

func TestVerifiedReflux_RedisUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)

	l := NewVerifiedReflux(new(TestVerifiedRefluxProvider), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	mr.Close()

	randKey := randString(6)
	_, err = l.Generate(defaultKind, randKey)
	assert.Error(t, err)
}

func TestVerifiedReflux_OneTime(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	defer mr.Close()

	l := NewVerifiedReflux(
		new(TestVerifiedRefluxProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verified:reflux:"),
		WithKeyExpires(time.Minute*3),
	)
	randKey := randString(6)

	value, err := l.Generate(defaultKind, randKey, WithGenerateKeyExpires(time.Minute*5))
	assert.NoError(t, err)

	b := l.Verify(defaultKind, randKey, value)
	require.True(t, b)

	b = l.Verify(defaultKind, randKey, value)
	require.False(t, b)
}

func TestVerifiedReflux_in_quota(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	defer mr.Close()

	l := NewVerifiedReflux(
		new(TestVerifiedRefluxProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verified:reflux:"),
		WithKeyExpires(time.Minute*3),
		WithMaxErrQuota(3),
	)
	randKey := randString(6)
	value, err := l.Generate(defaultKind, randKey, WithGenerateKeyExpires(time.Minute*5))
	assert.NoError(t, err)

	badValue := value + "xxx"

	b := l.Verify(defaultKind, randKey, badValue)
	require.False(t, b)
	b = l.Verify(defaultKind, randKey, badValue)
	require.False(t, b)
	b = l.Verify(defaultKind, randKey, value)
	require.True(t, b)
}

func TestVerifiedReflux_over_quota(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	defer mr.Close()

	l := NewVerifiedReflux(
		new(TestVerifiedRefluxProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verified:reflux:"),
		WithKeyExpires(time.Minute*3),
		WithMaxErrQuota(3),
	)
	randKey := randString(6)
	value, err := l.Generate(defaultKind, randKey,
		WithGenerateKeyExpires(time.Minute*5),
		WithGenerateMaxErrQuota(6),
	)
	assert.NoError(t, err)

	badValue := value + "xxx"

	for i := 0; i < 6; i++ {
		b := l.Verify(defaultKind, randKey, badValue)
		require.False(t, b)
	}
	b := l.Verify(defaultKind, randKey, value)
	require.False(t, b)
}

// // TODO: success in redis, but failed in miniredis
// func TestVerifiedReflux_OneTime_Timeout(t *testing.T) {
// 	mr, err := miniredis.Run()
// 	assert.NoError(t, err)
//
// 	defer mr.Close()
//
// 	l := NewVerifiedReflux(new(TestVerifiedRefluxProvider),
// 		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
// 		// redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "123456", DB: 0}),
// 	)
// 	randKey := randString(6)
// 	value, err := l.Generate(defaultKind, randKey, WithGenerateKeyExpires(time.Second*1))
// 	assert.NoError(t, err)
//
// 	time.Sleep(time.Second * 2)
//
// 	b := l.Verify(defaultKind, randKey, value)
// 	require.False(t, b)
// }
