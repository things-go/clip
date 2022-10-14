package limit_verified

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const DefaultKind = "default"

const (
	limitVerifiedSendCodeScript = `
local keyPrefix = KEYS[1] -- keyPrefix
local kind = KEYS[2] -- kind 
local target = KEYS[3] -- target
local code = ARGV[1] -- 验证码
local maxSendPerDay = tonumber(ARGV[2]) -- 限制一天最大发送次数
local codeMaxSendPerDay = tonumber(ARGV[3]) -- 限制一天最大发送次数
local codeMaxErrorQuota = tonumber(ARGV[4]) -- 验证码最大验证失败次数
local codeAvailWindowSecond = tonumber(ARGV[5]) -- 验证码有效窗口时间, 单位: 秒
local codeResendIntervalSecond = tonumber(ARGV[6]) -- 验证码重发间隔时间
local now = tonumber(ARGV[7]) -- 当前时间, 单位秒
local expires = tonumber(ARGV[8]) -- global key 过期时间, 单位: 秒

local globalKey = keyPrefix .. target
local codeKey = keyPrefix .. target .. ":_entry_:{" .. kind .. "}"

local sendCnt = redis.call("HINCRBY", globalKey, "sendCnt", 1)
local codeCnt = redis.call("HINCRBY", globalKey, "codeCnt", 1)
if sendCnt == 1 then
    redis.call("EXPIRE", globalKey, expires)
end
if sendCnt > maxSendPerDay or codeCnt > codeMaxSendPerDay then
	redis.call("HINCRBY", globalKey, "sendCnt", -1)
	redis.call("HINCRBY", globalKey, "codeCnt", -1)
    return 1 -- 超过每天发送限制次数
end

if (redis.call("EXISTS", codeKey) == 1) then
    local lastedAt = tonumber(redis.call("HGET", codeKey, "lasted"))
    if lastedAt + codeResendIntervalSecond > now then
		redis.call("HINCRBY", globalKey, "sendCnt", -1)
		redis.call("HINCRBY", globalKey, "codeCnt", -1)
        return 2 -- 发送过于频繁, 即还在重发限制窗口
    end
end

redis.call("HMSET", codeKey, "code", code, "quota", codeMaxErrorQuota, "err", 0, "lasted", now)
redis.call("EXPIRE", codeKey, codeAvailWindowSecond)

return 0 -- 成功
`
	limitVerifiedRollbackSendCntAndCodeCntScript = `
local keyPrefix = KEYS[1] -- keyPrefix
local kind = KEYS[2] -- kind 
local target = KEYS[3] -- target
local code = ARGV[1] -- 验证码
local now = tonumber(ARGV[2]) -- 当前时间, 单位秒

local globalKey = keyPrefix .. target
local codeKey = keyPrefix .. target .. ":_entry_:{" .. kind .. "}"

redis.call("HINCRBY", globalKey, "sendCnt", -1)
redis.call("HINCRBY", globalKey, "codeCnt", -1)
if (redis.call("EXISTS", codeKey) == 1) then
	local currentCode = redis.call("HGET", codeKey, "code")
	local lastedAt = tonumber(redis.call("HGET", codeKey, "lasted"))
	if currentCode == code and lastedAt == now then 
        redis.call("DEL", codeKey) -- 的确是你发送的, 删除 code key
	end
end
`
	limitVerifiedVerifyCodeScript = `
local keyPrefix = KEYS[1] -- keyPrefix
local kind = KEYS[2] -- kind 
local target = KEYS[3] -- target
local code = ARGV[1] -- 验证码
local now = tonumber(ARGV[2]) -- 当前时间, 单位秒

local globalKey = keyPrefix .. target
local codeKey = keyPrefix .. target .. ":_entry_:{" .. kind .. "}" 

if redis.call("EXISTS", codeKey) == 0 then
    return 1  -- 未发送短信验证码 或 验证码已过期
end

local errCnt = tonumber(redis.call('HGET', codeKey, "err"))
local codeMaxErrorQuota = tonumber(redis.call('HGET', codeKey, "quota"))
local currentCode = redis.call('HGET', codeKey, "code")
if errCnt >= codeMaxErrorQuota then
    return 2  -- 验证码错误次数超过限制
end
if currentCode == code then
    redis.call("DEL", codeKey) -- 删除 code key
    return 0 -- 成功
else
    redis.call('HINCRBY', codeKey, "err", 1)
    return 3 -- 验证码错误
end
`
	limitVerifiedIncrSendCntScript = `
local keyPrefix = KEYS[1] -- keyPrefix
local target = KEYS[2] -- target
local maxSendPerDay = tonumber(ARGV[1]) -- 限制一天最大发送次数
local expires = tonumber(ARGV[2]) -- global key 过期时间, 单位: 秒

local globalKey = keyPrefix .. target

local sendCnt = redis.call("HINCRBY", globalKey, "sendCnt", 1)
if sendCnt == 1 then
    redis.call("EXPIRE", globalKey, expires)
end
if sendCnt > maxSendPerDay then
	redis.call("HINCRBY", globalKey, "sendCnt", -1)
    return 1 -- 超过每天发送限制次数
end
return 0 -- 成功
`
	limitVerifiedDecrSendCntScript = `
local keyPrefix = KEYS[1] -- keyPrefix
local target = KEYS[2] -- target

local globalKey = keyPrefix .. target

local sendCnt = redis.call("HINCRBY", globalKey, "sendCnt", -1)
if sendCnt < 0 then
    redis.call("DEL", globalKey)
end
return 0
`
)

// error defined for verified
var (
	// ErrUnknownCode is an error that represents unknown status code.
	ErrUnknownCode           = errors.New("limit: unknown status code")
	ErrMaxSendPerDay         = errors.New("limit: reach the maximum send times")
	ErrResendTooFrequently   = errors.New("limit: resend too frequently")
	ErrCodeRequiredOrExpired = errors.New("limit: code is required or expired")
	ErrCodeMaxErrorQuota     = errors.New("limit: over the maximum error quota")
	ErrCodeVerification      = errors.New("limit: code verified failed")
)

type LimitVerifiedState int

const (
	// // LimitVerifiedStsUnknown means not initialized state.
	// LimitVerifiedStsUnknown LimitVerifiedState = -1
	// // LimitVerifiedStsSuccess means success.
	// LimitVerifiedStsSuccess LimitVerifiedState = 0
	//
	// // send code state value
	// // LimitVerifiedStsSendCodeOverMaxSendPerDay means passed the max send times per day.
	// LimitVerifiedStsSendCodeOverMaxSendPerDay LimitVerifiedState = 1
	// // LimitVerifiedStsSendCodeResendTooFrequently means resend to frequently.
	// LimitVerifiedStsSendCodeResendTooFrequently LimitVerifiedState = 2
	//
	// // LimitVerifiedStsVerifyCodeRequired means need code required, it is empty in store.
	// LimitVerifiedStsVerifyCodeRequired LimitVerifiedState = 1
	// // LimitVerifiedStsVerifyCodeExpired means code has expired.
	// LimitVerifiedStsVerifyCodeExpired LimitVerifiedState = 2
	// // LimitVerifiedStsVerifyCodeOverMaxErrorQuota means passed the max error quota.
	// LimitVerifiedStsVerifyCodeOverMaxErrorQuota LimitVerifiedState = 3
	// // LimitVerifiedStsVerifyCodeVerificationFailure means verification failure.
	// LimitVerifiedStsVerifyCodeVerificationFailure LimitVerifiedState = 4

	// inner lua send/verify code statue value
	innerLimitVerifiedSuccess = 0
	// inner lua send code value
	innerLimitVerifiedOfSendCodeReachMaxSendPerDay  = 1
	innerLimitVerifiedOfSendCodeResendTooFrequently = 2
	// inner lua verify code value
	innerLimitVerifiedOfVerifyCodeRequiredOrExpired   = 1
	innerLimitVerifiedOfVerifyCodeReachMaxError       = 2
	innerLimitVerifiedOfVerifyCodeVerificationFailure = 3
)

// LimitVerifiedProvider the provider
type LimitVerifiedProvider interface {
	Name() string
	SendCode(CodeParam) error
}

// LimitVerified limit verified code
type LimitVerified struct {
	p             LimitVerifiedProvider // LimitVerifiedProvider send code
	store         *redis.Client         // store client
	keyPrefix     string                // store 存验证码key的前缀, 默认 limit:verified:
	keyExpires    time.Duration         // store 存验证码key的过期时间, 默认: 24 小时
	maxSendPerDay int                   // 限制一天最大发送次数(全局), 默认: 10
	// 以下只针对验证码进行限制
	codeMaxSendPerDay        int // 验证码限制一天最大发送次数(验证码全局), 默认: 10, codeMaxSendPerDay <= maxSendPerDay
	codeMaxErrorQuota        int // 验证码最大验证失败次数, 默认: 3
	codeAvailWindowSecond    int // 验证码有效窗口时间, 默认180, 单位: 秒
	codeResendIntervalSecond int // 验证码重发间隔时间, 默认60, 单位: 秒
}

// NewVerified  new a limit verified
func NewVerified(p LimitVerifiedProvider, store *redis.Client, opts ...Option) *LimitVerified {
	v := &LimitVerified{
		p,
		store,
		"limit:verified:",
		time.Hour * 24,
		10,
		10,
		3,
		180,
		60,
	}
	for _, opt := range opts {
		opt(v)
	}
	if v.codeMaxSendPerDay > v.maxSendPerDay {
		v.codeMaxSendPerDay = v.maxSendPerDay
	}
	return v
}

// Name the provider name
func (v *LimitVerified) Name() string { return v.p.Name() }

// SendCode send code and store in redis cache.
func (v *LimitVerified) SendCode(c CodeParam, opts ...CodeParamOption) error {
	c.takeCodeParamOption(v, opts...)

	nowSecond := strconv.FormatInt(time.Now().Unix(), 10)
	result, err := v.store.Eval(context.Background(), limitVerifiedSendCodeScript,
		[]string{
			v.keyPrefix,
			c.Kind,
			c.Target,
		},
		[]string{
			c.Code,
			strconv.Itoa(v.maxSendPerDay),
			strconv.Itoa(v.codeMaxSendPerDay),
			strconv.Itoa(c.codeMaxErrorQuota),
			strconv.Itoa(c.codeAvailWindowSecond),
			strconv.Itoa(c.codeResendIntervalSecond),
			nowSecond,
			strconv.FormatInt(int64(v.keyExpires/time.Second), 10),
		},
	).Result()
	if err != nil {
		return err
	}
	sts, ok := result.(int64)
	if !ok {
		return ErrUnknownCode
	}
	switch sts {
	case innerLimitVerifiedSuccess:
		// 发送失败, 回滚发送次数
		defer func() {
			if err != nil && !errors.Is(err, ErrMaxSendPerDay) {
				v.store.Eval(context.Background(), limitVerifiedRollbackSendCntAndCodeCntScript,
					[]string{
						v.keyPrefix,
						c.Kind,
						c.Target,
					},
					[]string{
						c.Code,
						nowSecond,
					},
				)
			}
		}()
		err = v.p.SendCode(c)
	case innerLimitVerifiedOfSendCodeReachMaxSendPerDay:
		err = ErrMaxSendPerDay
	case innerLimitVerifiedOfSendCodeResendTooFrequently:
		err = ErrResendTooFrequently
	default:
		err = ErrUnknownCode
	}
	return err
}

// VerifyCode verify code from redis cache.
func (v *LimitVerified) VerifyCode(c CodeParam) error {
	c.takeCodeParamOption(v)

	result, err := v.store.Eval(
		context.Background(),
		limitVerifiedVerifyCodeScript,
		[]string{
			v.keyPrefix,
			c.Kind,
			c.Target,
		},
		[]string{
			c.Code,
			strconv.FormatInt(time.Now().Unix(), 10),
		},
	).Result()
	if err != nil {
		return err
	}

	sts, ok := result.(int64)
	if !ok {
		return ErrUnknownCode
	}
	switch sts {
	case innerLimitVerifiedSuccess:
		return nil
	case innerLimitVerifiedOfVerifyCodeRequiredOrExpired:
		err = ErrCodeRequiredOrExpired
	case innerLimitVerifiedOfVerifyCodeReachMaxError:
		err = ErrCodeMaxErrorQuota
	case innerLimitVerifiedOfVerifyCodeVerificationFailure:
		err = ErrCodeVerification
	default:
		err = ErrUnknownCode
	}
	return err
}

// Incr send cnt.
func (v *LimitVerified) Incr(target string) error {
	result, err := v.store.Eval(context.Background(), limitVerifiedIncrSendCntScript,
		[]string{
			v.keyPrefix,
			target,
		},
		[]string{
			strconv.Itoa(v.maxSendPerDay),
			strconv.FormatInt(int64(v.keyExpires/time.Second), 10),
		},
	).Result()
	if err != nil {
		return err
	}
	sts, ok := result.(int64)
	if !ok {
		return ErrUnknownCode
	}
	switch sts {
	case innerLimitVerifiedSuccess:
		err = nil
	case innerLimitVerifiedOfSendCodeReachMaxSendPerDay:
		err = ErrMaxSendPerDay
	default:
		err = ErrUnknownCode
	}
	return err
}

// Decr send cnt.
func (v *LimitVerified) Decr(target string) error {
	return v.store.Eval(context.Background(), limitVerifiedDecrSendCntScript,
		[]string{
			v.keyPrefix,
			target,
		},
	).Err()
}
