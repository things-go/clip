package verified

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// VerifiedRefluxProvider the provider
type VerifiedRefluxProvider interface {
	Name() string
	GenerateUniqueId() string
}

// VerifiedReflux verified captcha limit
type VerifiedReflux struct {
	p          VerifiedRefluxProvider // VerifiedCaptchaProvider generate captcha
	store      *redis.Client          // store client
	keyPrefix  string                 // store 存验证码key的前缀, 默认 limit:captcha:
	keyExpires time.Duration          // store 存验证码key的过期时间, 默认: 3 分种
}

// NewVerifiedReflux
func NewVerifiedReflux(p VerifiedRefluxProvider, store *redis.Client, opts ...Option) *VerifiedReflux {
	v := &VerifiedReflux{
		p,
		store,
		"verified:reflux:",
		time.Minute * 3,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *VerifiedReflux) setKeyPrefix(k string)         { v.keyPrefix = k }
func (v *VerifiedReflux) setKeyExpires(t time.Duration) { v.keyExpires = t }

// Name the provider name
func (v *VerifiedReflux) Name() string { return v.p.Name() }

// Generate generate uniqueId. use GenerateOption overwrite default key expires
func (v *VerifiedReflux) Generate(kind, key string, opts ...GenerateOption) (string, error) {
	opt := generateOption{
		keyExpires: v.keyExpires,
	}
	for _, f := range opts {
		f(&opt)
	}

	uniqueId := v.p.GenerateUniqueId()
	err := v.store.Set(context.Background(),
		v.keyPrefix+kind+":"+key,
		uniqueId,
		opt.keyExpires,
	).Err()
	if err != nil {
		return "", err
	}
	return uniqueId, nil
}

// Verify the uniqueId.
// shortcut Match(id, answer, true)
func (v *VerifiedReflux) Verify(kind, key, uniqueId string) bool {
	return v.Match(kind, key, uniqueId, true)
}

// Match the uniqueId.
// if clear is true, it will remove from store
func (v *VerifiedReflux) Match(kind, key, uniqueId string, clear bool) bool {
	s := "0"
	if clear {
		s = "1"
	}
	result, err := v.store.Eval(
		context.Background(),
		verifiedScript,
		[]string{v.keyPrefix + kind + ":" + key},
		[]string{uniqueId, s},
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
