/*
verified 包括 captcha 和 reflux:
  - captcha: 用于图形验证器或问答验证器
  - reflux: 用于签发验证.

verified 有两种方式验证方式:
  - 一次性验证器(失败或成功立即失败验证器)
  - 可多次失败验证(成功或失败超过最大错误限制次数则失效验证器)

一次性验证器:
	  redis 存储格式:
			captcha:  keyPrefix:{kind}:{id} -- answer
			reflux:  keyPrefix:{kind}:{key} -- unique

可多次失败验证:
	  redis 存储格式:
			captcha:  keyPrefix:{kind}:{id} ----- value -- answer
											  |-- quota  -- quota
			reflux:  keyPrefix:{kind}:{key} ----- value -- unique
											  |-- quota  -- quota
	  value: 验证值
	  quota: 最大错误限制次数
*/

package verified
