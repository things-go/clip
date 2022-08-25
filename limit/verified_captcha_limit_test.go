package limit

import (
	"math/bits"
	"math/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randString(length int, alphabets []byte) string {
	b := make([]byte, length)
	bn := bits.Len(uint(len(alphabets)))
	mask := int64(1)<<bn - 1
	max := 63 / bn
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63() + rand.Int63()))

	// A rand.Int63() generates 63 random bits, enough for alphabets letters!
	for i, cache, remain := 0, r.Int63(), max; i < length; {
		if remain == 0 {
			cache, remain = r.Int63(), max
		}
		if idx := int(cache & mask); idx < len(alphabets) {
			b[i] = alphabets[idx]
			i++
		}
		cache >>= bn
		remain--
	}
	return string(b)
}

var _ CaptchaProvider = (*TestCaptchaProvider)(nil)

type TestCaptchaProvider struct{}

func (t TestCaptchaProvider) Name() string { return "test_provider" }

func (t TestCaptchaProvider) GenerateQuestionAnswer() (*QuestionAnswer, error) {
	var a = []byte("QWERTYUIOPLKJHGFDSAZXCVBNMabcdefghijklmnopqrstuvwxyz")

	return &QuestionAnswer{
		Id:       randString(6, a),
		Question: "1+1",
		Answer:   "2",
	}, nil
}

func TestVerifiedCaptcha_RedisUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)

	l := NewVerifiedCaptcha(new(TestCaptchaProvider), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	mr.Close()

	_, _, err = l.Generate()
	assert.Error(t, err)
}

func TestVerifiedCaptchaLimit(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	defer mr.Close()

	l := NewVerifiedCaptcha(
		new(TestCaptchaProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithVerifiedCaptchaKeyPrefix("limit:captcha:"),
		WithVerifiedCaptchaKeyExpires(time.Minute*3),
	)
	id, _, err := l.Generate()
	assert.NoError(t, err)

	b := l.Match(id, "2", false)
	require.True(t, b)

	b = l.Verify(id, "3")
	require.False(t, b)

	b = l.Match(id, "2", false)
	require.False(t, b)
}

// TODO: success in redis, but failed in miniredis
// func TestVerifiedCaptcha_Timeout(t *testing.T) {
// 	mr, err := miniredis.Run()
// 	assert.NoError(t, err)
//
// 	defer mr.Close()
//
// 	l := NewVerifiedCaptcha(new(TestCaptchaProvider),
// 		redis.NewClient(&redis.Options{
// 			Addr: mr.Addr(),
// 		}),
// 		// redis.NewClient(&redis.Options{
// 		// 	Addr:     "localhost:6379",
// 		// 	Password: "123456",
// 		// 	DB:       9,
// 		// }),
// 		WithVerifiedCaptchaKeyExpires(time.Second*1),
// 	)
// 	id, _, err := l.Generate()
// 	assert.NoError(t, err)
//
// 	time.Sleep(time.Second * 3)
//
// 	b := l.Match(id, "2", false)
// 	require.False(t, b)
// }
