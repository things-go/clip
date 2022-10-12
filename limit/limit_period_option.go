package limit

import (
	"strings"
	"time"
)

// PeriodLimitOptionSetter period limit interface for PeriodLimit and PeriodFailureLimit
type PeriodLimitOptionSetter interface {
	align()
	setKeyPrefix(k string)
	setPeriod(v time.Duration)
	setQuota(v int)
}

// PeriodLimitOption defines the method to customize a PeriodLimit and PeriodFailureLimit.
type PeriodLimitOption func(l PeriodLimitOptionSetter)

// Align returns a func to customize a PeriodLimit and PeriodFailureLimit with alignment.
// For example, if we want to limit end users with 5 sms verification messages every day,
// we need to align with the local timezone and the start of the day.
func Align() PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.align()
	}
}

// KeyPrefix set key prefix
func KeyPrefix(k string) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		if !strings.HasSuffix(k, ":") {
			k += ":"
		}
		l.setKeyPrefix(k)
	}
}

// Period a period of time, must greater than a second
func Period(v time.Duration) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.setPeriod(v)
	}
}

// Quota limit quota requests during a period seconds of time.
func Quota(v int) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.setQuota(v)
	}
}

// GenerateOptionSetter generate option setter
type PeriodLimitParamSetter interface {
	setPeriod(expires time.Duration)
	setQuota(v int)
}

// PeriodLimitParamOption period limit param option
type PeriodLimitParamOption func(PeriodLimitParamSetter)

// WithPeriodLimitParamPeriod a period of time, must greater than a second
func WithPeriodLimitParamPeriod(t time.Duration) PeriodLimitParamOption {
	return func(p PeriodLimitParamSetter) {
		p.setPeriod(t)
	}
}

// WithPeriodLimitParamQuota limit quota requests during a period seconds of time.
func WithPeriodLimitParamQuota(v int) PeriodLimitParamOption {
	return func(p PeriodLimitParamSetter) {
		p.setQuota(v)
	}
}

type periodLimitParamOption struct {
	period int
	quota  int
}

func (p *periodLimitParamOption) calcExpireSeconds(isAlign bool) int {
	if isAlign {
		now := time.Now()
		_, offset := now.Zone()
		unix := now.Unix() + int64(offset)
		return p.period - int(unix%int64(p.period))
	}
	return p.period
}

func (p *periodLimitParamOption) setQuota(v int) { p.quota = v }

func (p *periodLimitParamOption) setPeriod(v time.Duration) {
	if vv := int(v / time.Second); vv > 0 {
		p.period = int(v / time.Second)
	}
}
