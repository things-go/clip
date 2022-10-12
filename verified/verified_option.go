package verified

import (
	"strings"
	"time"
)

// OptionSetter option setter
type OptionSetter interface {
	setKeyPrefix(k string)
	setKeyExpires(expires time.Duration)
	setMaxErrQuota(quota int)
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

// SetMaxErrQuota 设置最大错误次数验证, 并禁用一次性验证
// NOTE: 并设置最大错误次数验证, 将禁用一次性验证(默认验证器一次性有效)
func WithMaxErrQuota(quota int) Option {
	return func(v OptionSetter) {
		if quota > 1 {
			v.setMaxErrQuota(quota)
		}
	}
}

// GenerateOptionSetter generate option setter
type GenerateOptionSetter interface {
	setKeyExpires(expires time.Duration)
	setMaxErrQuota(quota int)
}

// GenerateOption generate option
type GenerateOption func(GenerateOptionSetter)

// WithGenerateKeyExpires redis存验证码key的过期时间
func WithGenerateKeyExpires(t time.Duration) GenerateOption {
	return func(v GenerateOptionSetter) {
		v.setKeyExpires(t)
	}
}

// WithGenerateMaxErrQuota 设置最大错误验证次数
// NOTE: 仅禁用一次性验证器时, 该功能有效
func WithGenerateMaxErrQuota(quota int) GenerateOption {
	return func(v GenerateOptionSetter) {
		if quota > 1 {
			v.setMaxErrQuota(quota)
		}
	}
}

type generateOption struct {
	keyExpires  time.Duration
	maxErrQuota int
}

func (g *generateOption) setKeyExpires(t time.Duration) { g.keyExpires = t }
func (g *generateOption) setMaxErrQuota(quota int)      { g.maxErrQuota = quota }
