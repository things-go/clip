package limit

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

var internalErr = errors.New("internal error")

func TestPeriodFailureLimit_Check(t *testing.T) {
	testPeriodFailureLimit(t,
		KeyPrefix("PeriodFailureLimit"),
		Period(seconds),
		Quota(quota),
	)
}

func TestPeriodFailureLimit_CheckWithAlign(t *testing.T) {
	testPeriodFailureLimit(t, Align(),
		KeyPrefix("PeriodFailureLimit"),
		Period(seconds),
		Quota(quota),
	)
}

func TestPeriodFailureLimit_RedisUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		KeyPrefix("PeriodFailureLimit"),
		Period(seconds),
		Quota(quota),
	)
	mr.Close()
	sts, err := l.CheckErr(context.Background(),
		defaultKind,
		"first",
		nil,
		WithPeriodLimitParamPeriod(seconds),
		WithPeriodLimitParamQuota(quota),
	)
	assert.Error(t, err)
	assert.Equal(t, PeriodFailureLimitStsUnknown, sts)
}

func testPeriodFailureLimit(t *testing.T, opts ...PeriodLimitOption) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)

	defer mr.Close()

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}), opts...)
	var inLimitCnt, overFailureTimeCnt int
	for i := 0; i < total; i++ {
		sts, err := l.CheckErr(context.Background(), defaultKind, "first", internalErr)
		assert.NoError(t, err)
		switch sts {
		case PeriodFailureLimitStsInQuota:
			inLimitCnt++
		case PeriodFailureLimitStsOverQuota:
			overFailureTimeCnt++
		default:
			t.Errorf("unknown status, must be on of [%d, %d]", PeriodFailureLimitStsInQuota, PeriodFailureLimitStsOverQuota)
		}
	}
	assert.Equal(t, quota, inLimitCnt)
	assert.Equal(t, total-quota, overFailureTimeCnt)

	sts, err := l.CheckErr(context.Background(), defaultKind, "first", nil)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsOverQuota, sts)
}

func TestPeriodFailureLimit_Check_In_Limit_Failure_Time_Then_Success(t *testing.T) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)

	defer mr.Close()

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		Period(seconds),
		Quota(quota),
	)
	var inLimitCnt, overFailureTimeCnt int
	for i := 0; i < quota-1; i++ {
		sts, err := l.CheckErr(context.Background(), defaultKind, "first", internalErr)
		assert.NoError(t, err)
		switch sts {
		case PeriodFailureLimitStsInQuota:
			inLimitCnt++
		case PeriodFailureLimitStsOverQuota:
			overFailureTimeCnt++
		default:
			t.Errorf("unknown status, must be on of [%d, %d]", PeriodFailureLimitStsInQuota, PeriodFailureLimitStsOverQuota)
		}
	}
	assert.Equal(t, quota-1, inLimitCnt)
	assert.Equal(t, 0, overFailureTimeCnt)

	sts, err := l.CheckErr(context.Background(), defaultKind, "first", nil)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsSuccess, sts)

	v, existed, err := l.GetInt(context.Background(), defaultKind, "first")
	assert.NoError(t, err)
	assert.False(t, existed)
	assert.Zero(t, v)
}

func TestPeriodFailureLimit_Check_Over_Limit_Failure_Time_Then_Success_Always_OverFailureTimeError(t *testing.T) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)

	defer mr.Close()

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		Period(seconds),
		Quota(quota),
	)
	var inLimitCnt, overFailureTimeCnt int
	for i := 0; i < quota+1; i++ {
		sts, err := l.CheckErr(context.Background(), defaultKind, "first", internalErr)
		assert.NoError(t, err)
		switch sts {
		case PeriodFailureLimitStsInQuota:
			inLimitCnt++
		case PeriodFailureLimitStsOverQuota:
			overFailureTimeCnt++
		default:
			t.Errorf("unknown status, must be on of [%d, %d]", PeriodFailureLimitStsInQuota, PeriodFailureLimitStsOverQuota)
		}
	}
	assert.Equal(t, quota, inLimitCnt)
	assert.Equal(t, 1, overFailureTimeCnt)

	sts, err := l.CheckErr(context.Background(), defaultKind, "first", nil)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsOverQuota, sts)

	v, existed, err := l.GetInt(context.Background(), defaultKind, "first")
	assert.NoError(t, err)
	assert.True(t, existed)
	assert.Equal(t, quota+1, v)
}

func TestPeriodFailureLimit_SetQuotaFull(t *testing.T) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)
	defer mr.Close()

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		Period(seconds),
		Quota(quota),
	)

	err = l.SetQuotaFull(context.Background(), defaultKind, "first")
	assert.Nil(t, err)

	sts, err := l.CheckErr(context.Background(), defaultKind, "first", nil)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsOverQuota, sts)
}

func TestPeriodFailureLimit_Del(t *testing.T) {
	mr, err := miniredis.Run()
	assert.Nil(t, err)
	defer mr.Close()

	l := NewPeriodFailureLimit(
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		Period(seconds),
		Quota(quota),
	)

	v, b, err := l.GetInt(context.Background(), defaultKind, "first")
	assert.Nil(t, err)
	assert.False(t, b)
	assert.Equal(t, 0, v)

	// 第一次ttl, 不存在
	tt, err := l.TTL(context.Background(), defaultKind, "first")
	assert.Nil(t, err)
	assert.Equal(t, int(tt), -2)

	err = l.SetQuotaFull(context.Background(), defaultKind, "first")
	assert.Nil(t, err)

	// 第二次ttl, key 存在
	tt, err = l.TTL(context.Background(), defaultKind, "first")
	assert.Nil(t, err)
	assert.LessOrEqual(t, tt, seconds)

	v, b, err = l.GetInt(context.Background(), defaultKind, "first")
	assert.Nil(t, err)
	assert.True(t, b)
	assert.Equal(t, quota, v)

	sts, err := l.CheckErr(context.Background(), defaultKind, "first", internalErr)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsOverQuota, sts)
	assert.True(t, sts.IsOverQuota())

	err = l.Del(context.Background(), defaultKind, "first")
	assert.Nil(t, err)

	sts, err = l.CheckErr(context.Background(), defaultKind, "first", internalErr)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsInQuota, sts)
	assert.True(t, sts.IsWithinQuota())

	sts, err = l.CheckErr(context.Background(), defaultKind, "first", nil)
	assert.NoError(t, err)
	assert.Equal(t, PeriodFailureLimitStsSuccess, sts)
	assert.True(t, sts.IsSuccess())
}
