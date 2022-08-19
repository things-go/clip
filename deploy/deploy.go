//go:generate stringer -type=Deploy -linecomment
package deploy

import (
	"log"
)

type Deploy int

const (
	None  Deploy = iota // none
	Debug               // debug
	Dev                 // dev
	Uat                 // uat
	Prod                // prod
)

var deploy = None

// Convert convert string to Deploy
func Convert(m string) Deploy {
	switch m {
	case Debug.String():
		return Debug
	case Dev.String():
		return Dev
	case Uat.String():
		return Uat
	case Prod.String():
		return Prod
	default:
		return None
	}
}

// Set 设置布署模式
func Set(m Deploy) {
	deploy = m
}

// Get 获取当前的布署模式
func Get() Deploy { return deploy }

// IsDevelop 是否开发模式
func IsDevelop() bool { return deploy == Dev }

// IsDebug 是否调试模式
func IsDebug() bool { return deploy == Debug }

// IsUat 是否预发布模式
func IsUat() bool { return deploy == Uat }

// IsProduction 是否生产模式
func IsProduction() bool { return deploy == Prod }

// IsTest 测试: 开发或者调试
func IsTest() bool { return IsDevelop() || IsDebug() }

// IsRelease 预发或者生产环境
func IsRelease() bool { return IsUat() || IsProduction() }

// MustSetDeploy 设置布署模式, 不得为 None 模式, 否则panic
func MustSetDeploy(m string) {
	Set(Convert(m))
	CheckMustDeploy()
}

// GetDeploy 获取当前的布署模式
func GetDeploy() string {
	return Get().String()
}

// CheckMustDeploy 校验当前的布署环境必须设置非 unknown 模式, 否则panic
func CheckMustDeploy() {
	if deploy == None {
		log.Fatalf("Please set deploy mode first, must be one of local, dev, debug, uat, prod")
	}
}
