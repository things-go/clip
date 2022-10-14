package verified

const (
	verifiedGenerateScript = `
local key = KEYS[1]    -- key
local value = ARGV[1] -- 值
local quota = tonumber(ARGV[2]) -- 最大错误限制次数
local expires = tonumber(ARGV[3]) -- 过期时间

redis.call("HMSET", key, "value", value, "quota", quota, "err", 0)
redis.call("EXPIRE", key, expires)
return 0    -- 成功
`
	verifiedMatchScript = `
local key = KEYS[1]    -- key
local value = ARGV[1] -- 答案

if redis.call("EXISTS", key) == 0 then
	return 1   -- 键不存在, 验证失败
end

local wantValue = redis.call("HGET", key, "value")
if wantValue == value then
	redis.call("DEL", key)
	return 0  -- 成功
else 
	local quota = tonumber(redis.call("HGET", key, "quota"))
	local errCnt = redis.call("HINCRBY", key, "err", 1)
	if errCnt >= quota then 
		redis.call("DEL", key)
	end
		return 1   -- 值不相等, 验证失败
end
`
)
