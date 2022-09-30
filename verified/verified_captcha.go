package verified

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// QuestionAnswer question and answer for provider
type QuestionAnswer struct {
	Id       string
	Question string
	Answer   string
}

// VerifiedCaptchaDriver the driver
type VerifiedCaptchaDriver interface {
	Name() string
	GenerateQuestionAnswer() (*QuestionAnswer, error)
}

// VerifiedCaptchaProvider the provider
type VerifiedCaptchaProvider interface {
	AcquireDriver(kind string) VerifiedCaptchaDriver
}

// VerifiedCaptcha verified captcha limit
type VerifiedCaptcha struct {
	p          VerifiedCaptchaProvider // VerifiedCaptchaProvider generate captcha provider
	store      *redis.Client           // store client
	keyPrefix  string                  // store 存验证码key的前缀, 默认 verified:captcha:
	keyExpires time.Duration           // store 存验证码key的过期时间, 默认: 3 分种
}

// NewVerifiedCaptcha
func NewVerifiedCaptcha(p VerifiedCaptchaProvider, store *redis.Client, opts ...Option) *VerifiedCaptcha {
	v := &VerifiedCaptcha{
		p,
		store,
		"verified:captcha:",
		time.Minute * 3,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *VerifiedCaptcha) setKeyPrefix(k string)         { v.keyPrefix = k }
func (v *VerifiedCaptcha) setKeyExpires(t time.Duration) { v.keyExpires = t }

// Name the provider name
func (v *VerifiedCaptcha) Name(kind string) string { return v.p.AcquireDriver(kind).Name() }

// Generate generate id, question. use GenerateOption overwrite default key expires
func (v *VerifiedCaptcha) Generate(kind string, opts ...GenerateOption) (id, question string, err error) {
	opt := generateOption{
		keyExpires: v.keyExpires,
	}
	for _, f := range opts {
		f(&opt)
	}

	q, err := v.p.AcquireDriver(kind).GenerateQuestionAnswer()
	if err != nil {
		return "", "", err
	}
	err = v.store.Set(context.Background(),
		v.keyPrefix+kind+":"+q.Id, q.Answer,
		opt.keyExpires,
	).Err()
	if err != nil {
		return "", "", err
	}
	return q.Id, q.Question, nil
}

// Verify the answer.
// shortcut Match(id, answer, true)
func (v *VerifiedCaptcha) Verify(kind, id, answer string) bool {
	return v.Match(kind, id, answer, true)
}

// Match the answer.
// if clear is true, it will remove from store
func (v *VerifiedCaptcha) Match(kind, id, answer string, clear bool) bool {
	s := "0"
	if clear {
		s = "1"
	}
	result, err := v.store.Eval(
		context.Background(),
		verifiedScript,
		[]string{v.keyPrefix + kind + ":" + id},
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

type UnsupportedVerifiedCaptchaDriver struct{}

func (x UnsupportedVerifiedCaptchaDriver) Name() string { return "Unsupported verified captcha driver" }
func (x UnsupportedVerifiedCaptchaDriver) GenerateQuestionAnswer() (*QuestionAnswer, error) {
	return nil, errors.New(x.Name())
}
