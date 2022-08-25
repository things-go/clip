package binding

import (
	"github.com/shopspring/decimal"
)

// IsMobile 验证是否是手机号
func IsMobile(s string) bool {
	return rxPhone.MatchString(s)
}

// IsDecimal 是否是 decimal
func IsDecimal(s string) bool {
	_, err := decimal.NewFromString(s)
	return err == nil
}

// IsDecimalGt 是否是 decimal且大于t
func IsDecimalGt(s, t string) bool {
	d, tt, err := parseString2Decimal(s, t)
	if err != nil {
		return false
	}
	return d.GreaterThan(tt)
}

// IsDecimalGte 是否是 decimal且大于等于t
func IsDecimalGte(s, t string) bool {
	d, tt, err := parseString2Decimal(s, t)
	if err != nil {
		return false
	}
	return d.GreaterThanOrEqual(tt)
}

// IsDecimalLt 是否是 decimal且小于t
func IsDecimalLt(s, t string) bool {
	d, tt, err := parseString2Decimal(s, t)
	if err != nil {
		return false
	}
	return d.LessThan(tt)
}

// IsDecimalLte 是否是 decimal且小于等于t
func IsDecimalLte(s, t string) bool {
	d, tt, err := parseString2Decimal(s, t)
	if err != nil {
		return false
	}
	return d.LessThanOrEqual(tt)
}

// IsNumberGt0 是否是 decimal且大于0
func IsNumberGt0(s string) bool {
	return rxNumberGt0.MatchString(s)
}

// IsNumberGte0 是否是 decimal且大于等于0
func IsNumberGte0(s string) bool {
	return rxNumberGte0.MatchString(s)
}

func parseString2Decimal(s, t string) (d, tt decimal.Decimal, err error) {
	d, err = decimal.NewFromString(s)
	if err != nil {
		return
	}
	tt, err = decimal.NewFromString(t)
	if err != nil {
		panic(err.Error())
	}
	return
}
