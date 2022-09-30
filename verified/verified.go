package verified

const verifiedScript = `
local key = KEYS[1]    -- key
local answer = ARGV[1] -- 答案
local clear = tonumber(ARGV[2]) -- 是否清除

local wantAnswer = redis.call('GET', key)
if wantAnswer == false then
	return 1  -- 键不存在, 验证失败
end
if clear == 1 then
	redis.call('DEL', key)
end

if wantAnswer == answer then 
	return 0  -- 成功
end
return 1      -- 值不相等, 验证失败
`
