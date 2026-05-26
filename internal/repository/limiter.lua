local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

local clear_before = now - window

-- 1. Drop stale timestamps that fall outside our current sliding window boundary
redis.call('ZREMRANGEBYSCORE', key, 0, clear_before)

-- 2. Retrieve the total volume of requests executed within this active window
local current_requests = redis.call('ZCARD', key)

-- 3. If they are over the threshold, block the transaction immediately
if current_requests >= limit then
    return 0
else
    -- 4. Otherwise, log this current unique request timestamp into our sorted set
    redis.call('ZADD', key, now, now)
    -- Optimize memory footprint by setting an automatic expiration on the key
    redis.call('EXPIRE', key, window)
    return 1
end