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

// WithAlign returns a func to customize a PeriodLimit and PeriodFailureLimit with alignment.
// For example, if we want to limit end users with 5 sms verification messages every day,
// we need to align with the local timezone and the start of the day.
func WithAlign() PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.align()
	}
}

// WithKeyPrefix set key prefix
func WithKeyPrefix(k string) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		if !strings.HasSuffix(k, ":") {
			k += ":"
		}
		l.setKeyPrefix(k)
	}
}

// WithPeriod a period of time, must greater than a second
func WithPeriod(v time.Duration) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.setPeriod(v)
	}
}

// WithQuota limit quota requests during a period seconds of time.
func WithQuota(v int) PeriodLimitOption {
	return func(l PeriodLimitOptionSetter) {
		l.setQuota(v)
	}
}
