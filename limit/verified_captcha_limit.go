package limit

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const verifiedLimitCaptchaScript = `
local key = KEYS[1]    -- key
local answer = ARGV[1] -- 答案
local clear = tonumber(ARGV[2]) -- 是否清除

local wantAnswer = redis.call('GET', key)
if wantAnswer == false then
	return 1  -- 键不存在, 验证失败
end
if clear == 1 then
	redis.call('DEL', key)
end

if wantAnswer == answer then 
	return 0  -- 成功
end
return 1      -- 值不相等, 验证失败
`

// QuestionAnswer question and answer for provider
type QuestionAnswer struct {
	Id       string
	Question string
	Answer   string
}

// CaptchaProvider the provider
type CaptchaProvider interface {
	Name() string
	GenerateQuestionAnswer() (*QuestionAnswer, error)
}

// VerifiedCaptchaLimit verified captcha limit
type VerifiedCaptchaLimit struct {
	p          CaptchaProvider // CaptchaProvider generate captcha
	store      *redis.Client   // store client
	keyPrefix  string          // store 存验证码key的前缀, 默认 limit:captcha:
	keyExpires time.Duration   // store 存验证码key的过期时间, 默认: 5 分种
}

// VerifiedCaptchaOption VerifiedCaptchaLimit 选项
type VerifiedCaptchaOption func(*VerifiedCaptchaLimit)

// WithVerifiedCaptchaKeyPrefix redis存验证码key的前缀, 默认 limit:captcha:
func WithVerifiedCaptchaKeyPrefix(k string) VerifiedCaptchaOption {
	return func(v *VerifiedCaptchaLimit) {
		if k != "" {
			if !strings.HasSuffix(k, ":") {
				k += ":"
			}
			v.keyPrefix = k
		}
	}
}

// WithVerifiedCaptchaKeyExpires redis存验证码key的过期时间, 默认 3 分钟
func WithVerifiedCaptchaKeyExpires(expires time.Duration) VerifiedCaptchaOption {
	return func(v *VerifiedCaptchaLimit) {
		v.keyExpires = expires
	}
}

// NewVerifiedCaptcha
// Deprecated: use verified package
func NewVerifiedCaptcha(p CaptchaProvider, store *redis.Client, opts ...VerifiedCaptchaOption) *VerifiedCaptchaLimit {
	v := &VerifiedCaptchaLimit{
		p,
		store,
		"limit:captcha:",
		time.Minute * 3,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Name the provider name
func (v *VerifiedCaptchaLimit) Name() string { return v.p.Name() }

// Generate generate id, question.
func (v *VerifiedCaptchaLimit) Generate() (id, question string, err error) {
	q, err := v.p.GenerateQuestionAnswer()
	if err != nil {
		return "", "", err
	}
	err = v.store.Set(
		context.Background(),
		v.keyPrefix+q.Id, q.Answer,
		v.keyExpires,
	).Err()
	if err != nil {
		return "", "", err
	}
	return q.Id, q.Question, nil
}

// Verify the answer.
// shortcut Match(id, answer, true)
func (v *VerifiedCaptchaLimit) Verify(id, answer string) bool {
	return v.Match(id, answer, true)
}

// Match the answer.
// if clear is true, it will remove from store
func (v *VerifiedCaptchaLimit) Match(id, answer string, clear bool) (matched bool) {
	s := "0"
	if clear {
		s = "1"
	}
	result, err := v.store.Eval(
		context.Background(),
		verifiedLimitCaptchaScript,
		[]string{v.keyPrefix + id},
		[]string{answer, s},
	).Result()
	if err != nil {
		return false
	}
	code, ok := result.(int64)
	if !ok {
		return false
	}
	return code == 0
}
