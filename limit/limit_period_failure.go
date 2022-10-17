package limit

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	periodFailureLimitFixedScript = `
local key = KEYS[1] -- key
local quota = tonumber(ARGV[1]) -- 限制次数
local window = tonumber(ARGV[2]) -- 限制时间
local success = tonumber(ARGV[3]) -- 是否成功

if success == 1 then
    local current = tonumber(redis.call("GET", key))
    if current == nil then
        return 0 -- 成功
    end
    if current < quota then -- 未超出失败最大次数限制范围, 成功, 并清除限制
        redis.call("DEL", key)
        return 0 -- 成功
    end
    return 2 -- 超过失败最大次数限制
end

local current = redis.call("INCRBY", key, 1)
if current == 1 then 
    redis.call("EXPIRE", key, window)
end 
if current <= quota then
    return 1 -- 还在限制范围, 只提示错误
end
return 2 -- 超过失败最大次数限制
`
	periodFailureLimitFixedSetQuotaFullScript = `
local key = KEYS[1]
local quota = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = tonumber(redis.call("GET", key))
if current == nil then 
	redis.call("SETEX", key, window, quota)
elseif current < quota then 
	redis.call("SET", key, quota)
end
return 0
`
)

// PeriodFailureLimitState period failure limit state.
type PeriodFailureLimitState int

const (
	// PeriodFailureLimitStsUnknown means not initialized state.
	PeriodFailureLimitStsUnknown PeriodFailureLimitState = iota - 1
	// PeriodFailureLimitStsSuccess means success.
	PeriodFailureLimitStsSuccess
	// PeriodFailureLimitStsInQuota means within the quota.
	PeriodFailureLimitStsInQuota
	// PeriodFailureLimitStsOverQuota means over the quota.
	PeriodFailureLimitStsOverQuota

	// inner lua code
	// innerPeriodFailureLimitCodeSuccess means success.
	innerPeriodFailureLimitCodeSuccess = 0
	// innerPeriodFailureLimitCodeInQuota means within the quota.
	innerPeriodFailureLimitCodeInQuota = 1
	// innerPeriodFailureLimitCodeOverQuota means passed the quota.
	innerPeriodFailureLimitCodeOverQuota = 2
)

// IsSuccess means success state.
func (p PeriodFailureLimitState) IsSuccess() bool { return p == PeriodFailureLimitStsSuccess }

// IsWithinQuota means within the quota.
func (p PeriodFailureLimitState) IsWithinQuota() bool { return p == PeriodFailureLimitStsInQuota }

// IsOverQuota means passed the quota.
func (p PeriodFailureLimitState) IsOverQuota() bool { return p == PeriodFailureLimitStsOverQuota }

// A PeriodFailureLimit is used to limit requests when failure during a period of time.
type PeriodFailureLimit struct {
	// a period seconds of time
	period int
	// limit quota requests during a period seconds of time.
	quota int
	// keyPrefix in redis
	keyPrefix string
	store     *redis.Client
	isAlign   bool
}

// NewPeriodFailureLimit returns a PeriodFailureLimit with given parameters.
func NewPeriodFailureLimit(store *redis.Client, opts ...PeriodLimitOption) *PeriodFailureLimit {
	limiter := &PeriodFailureLimit{
		period:    int(24 * time.Hour / time.Second),
		quota:     6,
		keyPrefix: "limit:period:failure:", // limit:period:failure:
		store:     store,
	}
	for _, opt := range opts {
		opt(limiter)
	}
	return limiter
}

func (p *PeriodFailureLimit) align()                { p.isAlign = true }
func (p *PeriodFailureLimit) setKeyPrefix(k string) { p.keyPrefix = k }
func (p *PeriodFailureLimit) setPeriod(v time.Duration) {
	if vv := int(v / time.Second); vv > 0 {
		p.period = int(v / time.Second)
	}
}
func (p *PeriodFailureLimit) setQuota(v int) { p.quota = v }

// CheckErr requests a permit state.
// same as Check
func (p *PeriodFailureLimit) CheckErr(ctx context.Context, key string, err error) (PeriodFailureLimitState, error) {
	return p.Check(ctx, key, err == nil)
}

// Check requests a permit.
func (p *PeriodFailureLimit) Check(ctx context.Context, key string, success bool) (PeriodFailureLimitState, error) {
	s := "0"
	if success {
		s = "1"
	}
	result, err := p.store.Eval(ctx,
		periodFailureLimitFixedScript,
		[]string{p.formatKey(key)},
		[]string{
			strconv.Itoa(p.quota),
			strconv.Itoa(p.calcExpireSeconds()),
			s,
		},
	).Result()
	if err != nil {
		return PeriodFailureLimitStsUnknown, err
	}
	code, ok := result.(int64)
	if !ok {
		return PeriodFailureLimitStsUnknown, ErrUnknownCode
	}
	switch code {
	case innerPeriodFailureLimitCodeSuccess:
		return PeriodFailureLimitStsSuccess, nil
	case innerPeriodFailureLimitCodeInQuota:
		return PeriodFailureLimitStsInQuota, nil
	case innerPeriodFailureLimitCodeOverQuota:
		return PeriodFailureLimitStsOverQuota, nil
	default:
		return PeriodFailureLimitStsUnknown, ErrUnknownCode
	}
}

// SetQuotaFull set a permit over quota.
func (p *PeriodFailureLimit) SetQuotaFull(ctx context.Context, key string) error {
	err := p.store.Eval(ctx,
		periodFailureLimitFixedSetQuotaFullScript,
		[]string{p.formatKey(key)},
		[]string{
			strconv.Itoa(p.quota),
			strconv.Itoa(p.calcExpireSeconds()),
		},
	).Err()
	if err == redis.Nil {
		return nil
	}
	return err
}

// Del delete a permit
func (p *PeriodFailureLimit) Del(ctx context.Context, key string) error {
	return p.store.Del(ctx, p.formatKey(key)).Err()
}

// TTL get key ttl
// if key not exist, time = -1.
// if key exist, but not set expire time, t = -2
func (p *PeriodFailureLimit) TTL(ctx context.Context, key string) (time.Duration, error) {
	return p.store.TTL(ctx, p.formatKey(key)).Result()
}

// GetInt get current failure count
func (p *PeriodFailureLimit) GetInt(ctx context.Context, key string) (int, bool, error) {
	v, err := p.store.Get(ctx, p.formatKey(key)).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, err
	}
	return v, true, nil
}

func (p *PeriodFailureLimit) formatKey(key string) string {
	return p.keyPrefix + key
}

func (p *PeriodFailureLimit) calcExpireSeconds() int {
	if p.isAlign {
		now := time.Now()
		_, offset := now.Zone()
		unix := now.Unix() + int64(offset)
		return p.period - int(unix%int64(p.period))
	}
	return p.period
}
