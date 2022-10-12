package verified

import (
	"context"
	"strconv"
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
	p              VerifiedRefluxProvider // VerifiedCaptchaProvider generate captcha
	store          *redis.Client          // store client
	disableOneTime bool                   // 禁用一次性验证
	keyPrefix      string                 // store 存验证码key的前缀, 默认 verified:reflux:
	keyExpires     time.Duration          // store 存验证码key的过期时间, 默认: 3 分种
	maxErrQuota    int                    // store 验证码验证最大错误次数限制, 默认: 1
}

// NewVerifiedReflux
func NewVerifiedReflux(p VerifiedRefluxProvider, store *redis.Client, opts ...Option) *VerifiedReflux {
	v := &VerifiedReflux{
		p,
		store,
		false,
		"verified:reflux:",
		time.Minute * 3,
		1,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *VerifiedReflux) setKeyPrefix(k string)         { v.keyPrefix = k }
func (v *VerifiedReflux) setKeyExpires(t time.Duration) { v.keyExpires = t }
func (v *VerifiedReflux) setMaxErrQuota(quota int) {
	v.disableOneTime = true
	v.maxErrQuota = quota
}

// Name the provider name
func (v *VerifiedReflux) Name() string { return v.p.Name() }

// Generate generate uniqueId. use GenerateOption overwrite default key expires
func (v *VerifiedReflux) Generate(kind, key string, opts ...GenerateOption) (string, error) {
	var err error

	genOpt := generateOption{
		keyExpires:  v.keyExpires,
		maxErrQuota: v.maxErrQuota,
	}
	for _, f := range opts {
		f(&genOpt)
	}

	uniqueId := v.p.GenerateUniqueId()
	if v.disableOneTime {
		err = v.store.Eval(
			context.Background(),
			verifiedGenerateScript,
			[]string{
				v.keyPrefix + kind + ":" + key,
			},
			[]string{
				uniqueId,
				strconv.Itoa(genOpt.maxErrQuota),
				strconv.Itoa(int(genOpt.keyExpires / time.Second)),
			},
		).Err()
	} else {
		err = v.store.Set(context.Background(),
			v.keyPrefix+kind+":"+key,
			uniqueId,
			genOpt.keyExpires,
		).Err()
	}
	if err != nil {
		return "", err
	}
	return uniqueId, nil
}

// Verify the uniqueId.
// shortcut Match(id, answer, true)
func (v *VerifiedReflux) Verify(kind, key, uniqueId string) bool {
	if v.disableOneTime {
		result, err := v.store.Eval(
			context.Background(),
			verifiedMatchScript,
			[]string{
				v.keyPrefix + kind + ":" + key,
			},
			[]string{
				uniqueId,
			},
		).Result()
		if err != nil {
			return false
		}
		code, ok := result.(int64)
		if !ok {
			return false
		}
		return code == 0
	} else {
		wantAnswer, err := v.store.GetDel(
			context.Background(),
			v.keyPrefix+kind+":"+key,
		).Result()
		return err == nil && wantAnswer == uniqueId
	}
}
