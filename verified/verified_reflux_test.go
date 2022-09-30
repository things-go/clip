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

func TestVerifiedRefluxLimit(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	defer mr.Close()

	l := NewVerifiedReflux(
		new(TestVerifiedRefluxProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verified:Reflux:"),
		WithKeyExpires(time.Minute*3),
	)
	randKey := randString(6)

	value, err := l.Generate(defaultKind, randKey, WithGenerateKeyExpires(time.Minute*5))
	assert.NoError(t, err)

	b := l.Match(defaultKind, randKey, value, false)
	require.True(t, b)

	b = l.Verify(defaultKind, randKey, "xxx")
	require.False(t, b)

	b = l.Match(defaultKind, randKey, value, false)
	require.False(t, b)
}

// // TODO: success in redis, but failed in miniredis
// func TestVerifiedReflux_Timeout(t *testing.T) {
// 	mr, err := miniredis.Run()
// 	assert.NoError(t, err)
//
// 	defer mr.Close()
//
// 	l := NewVerifiedReflux(new(TestVerifiedRefluxProvider),
// 		redis.NewClient(&redis.Options{
// 			Addr: mr.Addr(),
// 		}),
// 		// redis.NewClient(&redis.Options{
// 		// 	Addr:     "localhost:6379",
// 		// 	Password: "123456",
// 		// 	DB:       9,
// 		// }),
// 		WithKeyExpires(time.Second*1),
// 	)
// 	randKey := randString(6)
// 	value, err := l.Generate(defaultKind, randKey)
// 	assert.NoError(t, err)
//
// 	time.Sleep(time.Second * 3)
//
// 	b := l.Match(defaultKind, randKey, value, false)
// 	require.False(t, b)
// }
