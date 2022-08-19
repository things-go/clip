package signature

import (
	"crypto/subtle"
	"strconv"
	"time"
)

// IatSign 签发获取签发时间和签名
func IatSign(s string) (iat, sign string) {
	return IatSignWith(s, func(iat, s string) string {
		return HmacSha256(iat, iat+s)
	})
}

// VerifyIatSign 验证签发时间是否在有效期内, 并验证签名是否正确
func VerifyIatSign(iat, targetSign, s string, iatTimout time.Duration) bool {
	return VerifyIatSignWith(iat, targetSign, s, iatTimout, func(iat, s string) string {
		return HmacSha256(iat, iat+s)
	})
}

// IatSignWith 签发获取签发时间和签名
func IatSignWith(s string, hash func(iat, s string) string) (iat, sign string) {
	iat = Iat()
	return iat, hash(iat, s)
}

// VerifyIatSignWith 验证签发时间是否在有效期内, 并验证签名是否正确
func VerifyIatSignWith(iat, targetSign, s string, availWindow time.Duration, hash func(iat, s string) string) bool {
	return VerifyIat(iat, availWindow) &&
		subtle.ConstantTimeCompare([]byte(targetSign), []byte(hash(iat, s))) == 1
}

// Iat 签发时间字符串
func Iat() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// VerifyIat 验证签发时间是否在有效期内
func VerifyIat(iat string, availWindow time.Duration) bool {
	ns, err := strconv.ParseInt(iat, 10, 64)
	if err != nil {
		return false
	}
	t := time.Unix(ns/int64(time.Second), ns%int64(time.Second))
	return t.Add(availWindow).After(time.Now())
}
