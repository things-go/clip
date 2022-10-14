package limit_verified

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	target  = "112233"
	code    = "123456"
	badCode = "654321"
)

var _ LimitVerifiedProvider = (*TestProvider)(nil)

type TestProvider struct{}

func (t TestProvider) Name() string { return "test_provider" }

func (t TestProvider) SendCode(c CodeParam) error { return nil }

type TestErrProvider struct{}

func (t TestErrProvider) Name() string { return "test_provider" }

func (t TestErrProvider) SendCode(c CodeParam) error { return errors.New("发送失败") }

func TestName(t *testing.T) {
	l := NewVerified(new(TestProvider), redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}))
	require.Equal(t, "test_provider", l.Name())
}

func TestSendCode_RedisUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)

	l := NewVerified(new(TestProvider), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	mr.Close()

	err = l.SendCode(CodeParam{Target: target, Code: code})
	assert.NotNil(t, err)
}

func TestSendCode_Success(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verification"),
		WithKeyExpires(time.Hour),
	)
	err = l.SendCode(
		CodeParam{Target: target, Code: code},
		WithAvailWindowSecond(3),
	)
	require.NoError(t, err)
}

func TestSendCode_Err_Failure(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestErrProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithKeyPrefix("verification:err"),
		WithKeyExpires(time.Hour),
	)
	err = l.SendCode(CodeParam{Target: target, Code: code})
	require.Error(t, err)
}

func TestSendCode_MaxSendPerDay(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithMaxSendPerDay(1),
		WithCodeMaxSendPerDay(1),
		WithCodeResendIntervalSecond(1),
	)

	err = l.SendCode(CodeParam{Target: target, Code: code})
	require.NoError(t, err)

	time.Sleep(time.Second + time.Millisecond*10)
	err = l.SendCode(CodeParam{Target: target, Code: code})
	require.ErrorIs(t, err, ErrMaxSendPerDay)
}

func TestSendCode_Concurrency_MaxSendPerDay(t *testing.T) {
	var success uint32
	var failed uint32

	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithMaxSendPerDay(1),
	)

	wg := &sync.WaitGroup{}
	wg.Add(15)
	for i := 0; i < 15; i++ {
		go func() {
			defer wg.Done()

			err := l.SendCode(CodeParam{Target: target, Code: code})
			if err != nil {
				require.ErrorIs(t, err, ErrMaxSendPerDay)
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&success, 1)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, uint32(1), success)
	require.Equal(t, uint32(14), failed)
}

func TestSendCode_ResendTooFrequently(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)

	err = l.SendCode(CodeParam{Target: target, Code: code}, WithResendIntervalSecond(1))
	require.NoError(t, err)
	err = l.SendCode(CodeParam{Target: target, Code: code}, WithResendIntervalSecond(1))
	require.ErrorIs(t, err, ErrResendTooFrequently)
}

func TestSendCode_Concurrency_ResendTooFrequently(t *testing.T) {
	var success uint32
	var failed uint32

	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithCodeResendIntervalSecond(3),
	)

	wg := &sync.WaitGroup{}
	wg.Add(15)
	for i := 0; i < 15; i++ {
		go func() {
			defer wg.Done()

			err := l.SendCode(CodeParam{Target: target, Code: code})
			if err != nil {
				require.ErrorIs(t, err, ErrResendTooFrequently)
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&success, 1)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, uint32(1), success)
	require.Equal(t, uint32(14), failed)
}

func TestVerifyCode_Success(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)

	err = l.SendCode(CodeParam{Target: target, Code: code})
	require.Nil(t, err)

	err = l.VerifyCode(CodeParam{Target: target, Code: code})
	assert.NoError(t, err)
}

func TestVerifyCode_CodeRequired(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)

	err = l.VerifyCode(CodeParam{Target: target, Code: code})
	assert.ErrorIs(t, err, ErrCodeRequiredOrExpired)
}

// TODO: mini redis 测试失败, 但redis是成功的
// func TestVerifyCode_CodeExpired(t *testing.T) {
// 	mr, err := miniredis.Run()
// 	require.Nil(t, err)
// 	defer mr.Close()
//
// 	l := NewVerified(new(TestProvider),
// 		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
// 	)
// 	err = l.SendCode(CodeParam{Target: target, Code: code}, WithAvailWindowSecond(1))
// 	require.Nil(t, err)
//
// 	time.Sleep(time.Second * 3)
// 	err = l.VerifyCode(CodeParam{Target: target, Code: code})
// 	assert.ErrorIs(t, err, ErrCodeRequiredOrExpired)
// }

func TestVerifyCode_CodeMaxError(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)
	err = l.SendCode(CodeParam{Target: target, Code: code}, WithMaxErrorQuota(6))
	require.Nil(t, err)

	for i := 0; i < 6; i++ {
		err = l.VerifyCode(CodeParam{Target: target, Code: badCode})
		assert.ErrorIs(t, err, ErrCodeVerification)
	}
	err = l.VerifyCode(CodeParam{Target: target, Code: badCode})
	assert.ErrorIs(t, err, ErrCodeMaxErrorQuota)
}

func TestVerifyCode_Concurrency_CodeMaxError(t *testing.T) {
	var failedMaxError uint32
	var failedVerify uint32

	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithCodeMaxErrorQuota(3),
		WithCodeAvailWindowSecond(180),
	)

	err = l.SendCode(CodeParam{Target: target, Code: code})
	require.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(15)
	for i := 0; i < 15; i++ {
		go func() {
			defer wg.Done()

			err := l.VerifyCode(CodeParam{Target: target, Code: badCode})
			if err != nil {
				if errors.Is(err, ErrCodeMaxErrorQuota) {
					atomic.AddUint32(&failedMaxError, 1)
				} else {
					atomic.AddUint32(&failedVerify, 1)
				}
			}
		}()
	}

	wg.Wait()
	require.Equal(t, uint32(3), failedVerify)
	require.Equal(t, uint32(12), failedMaxError)
}

func Test_INCR_MaxSendPerDay(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		WithMaxSendPerDay(10),
	)
	for i := 0; i < 10; i++ {
		err = l.Incr(target)
		require.Nil(t, err)
	}
	err = l.Incr(target)
	require.ErrorIs(t, err, ErrMaxSendPerDay)
}

func Test_INCR_DECR(t *testing.T) {
	mr, err := miniredis.Run()
	require.Nil(t, err)
	defer mr.Close()

	l := NewVerified(new(TestProvider),
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)
	err = l.Incr(target)
	require.Nil(t, err)

	err = l.Decr(target)
	require.Nil(t, err)
}
