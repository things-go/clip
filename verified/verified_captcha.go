package verified

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// QuestionAnswer question and answer for driver
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
	p              VerifiedCaptchaProvider // VerifiedCaptchaProvider generate captcha provider
	store          *redis.Client           // store client
	disableOneTime bool                    // 禁用一次性验证
	keyPrefix      string                  // store 存验证码key的前缀, 默认 verified:captcha:
	keyExpires     time.Duration           // store 存验证码key的过期时间, 默认: 3 分种
	maxErrQuota    int                     // store 验证码验证最大错误次数限制, 默认: 1
}

// NewVerifiedCaptcha
func NewVerifiedCaptcha(p VerifiedCaptchaProvider, store *redis.Client, opts ...Option) *VerifiedCaptcha {
	v := &VerifiedCaptcha{
		p,
		store,
		false,
		"verified:captcha:",
		time.Minute * 3,
		1,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *VerifiedCaptcha) setKeyPrefix(k string)         { v.keyPrefix = k }
func (v *VerifiedCaptcha) setKeyExpires(t time.Duration) { v.keyExpires = t }
func (v *VerifiedCaptcha) setMaxErrQuota(quota int) {
	v.disableOneTime = true
	v.maxErrQuota = quota
}

// Name the provider name
func (v *VerifiedCaptcha) Name(kind string) string { return v.p.AcquireDriver(kind).Name() }

// Generate generate id, question.
func (v *VerifiedCaptcha) Generate(kind string, opts ...GenerateOption) (id, question string, err error) {
	genOpt := generateOption{
		keyExpires:  v.keyExpires,
		maxErrQuota: v.maxErrQuota,
	}
	for _, f := range opts {
		f(&genOpt)
	}

	q, err := v.p.AcquireDriver(kind).GenerateQuestionAnswer()
	if err != nil {
		return "", "", err
	}
	if v.disableOneTime {
		err = v.store.Eval(
			context.Background(),
			verifiedGenerateScript,
			[]string{
				v.keyPrefix + kind + ":" + q.Id,
			},
			[]string{
				q.Answer,
				strconv.Itoa(genOpt.maxErrQuota),
				strconv.Itoa(int(genOpt.keyExpires / time.Second)),
			},
		).Err()
	} else {
		err = v.store.Set(
			context.Background(),
			v.keyPrefix+kind+":"+q.Id,
			q.Answer,
			genOpt.keyExpires,
		).Err()
	}
	if err != nil {
		return "", "", err
	}
	return q.Id, q.Question, nil
}

// Verify the answer.
// shortcut Match(id, answer, true)
func (v *VerifiedCaptcha) Verify(kind, id, answer string) bool {
	if v.disableOneTime {
		result, err := v.store.Eval(
			context.Background(),
			verifiedMatchScript,
			[]string{
				v.keyPrefix + kind + ":" + id,
			},
			[]string{
				answer,
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
			v.keyPrefix+kind+":"+id,
		).Result()
		return err == nil && wantAnswer == answer
	}
}

type UnsupportedVerifiedCaptchaDriver struct{}

func (x UnsupportedVerifiedCaptchaDriver) Name() string { return "Unsupported verified captcha driver" }
func (x UnsupportedVerifiedCaptchaDriver) GenerateQuestionAnswer() (*QuestionAnswer, error) {
	return nil, errors.New(x.Name())
}
