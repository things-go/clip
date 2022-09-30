package verified

import (
	"strings"
	"time"
)

// OptionSetter option setter
type OptionSetter interface {
	setKeyPrefix(k string)
	setKeyExpires(expires time.Duration)
}

// Option 选项
type Option func(OptionSetter)

// WithKeyPrefix redis存验证码key的前缀, 默认 limit:captcha:
func WithKeyPrefix(k string) Option {
	return func(v OptionSetter) {
		if k != "" {
			if !strings.HasSuffix(k, ":") {
				k += ":"
			}
			v.setKeyPrefix(k)
		}
	}
}

// WithKeyExpires redis存验证码key的过期时间
func WithKeyExpires(t time.Duration) Option {
	return func(v OptionSetter) {
		v.setKeyExpires(t)
	}
}

// GenerateOptionSetter generate option setter
type GenerateOptionSetter interface {
	setKeyExpires(expires time.Duration)
}

// GenerateOption generate option
type GenerateOption func(GenerateOptionSetter)

// WithGenerateKeyExpires redis存验证码key的过期时间
func WithGenerateKeyExpires(t time.Duration) GenerateOption {
	return func(v GenerateOptionSetter) {
		v.setKeyExpires(t)
	}
}

type generateOption struct {
	keyExpires time.Duration
}

func (g *generateOption) setKeyExpires(t time.Duration) { g.keyExpires = t }
