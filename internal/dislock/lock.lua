-- rwlock --

local function write_lock(lock_key)
    local reply = redis.call('EXISTS', lock_key)
    local exists = tonumber(reply)
    if exists ~= nil and exists > 0 then
        return 0
    end
    reply = redis.call('SET', lock_key, -1, 'EX', 60)
    if reply == false then
        return 0
    end
    if reply == true or reply == 'OK' or reply['ok'] == 'OK' then
        return 1
    end
    return redis.error_reply(string.format('ERR Set lock key reply: %s', tostring(reply)))
end

local function write_unlock(lock_key)
    local reply = redis.call('GET', lock_key)
    local num = tonumber(reply)
    if num ~= nil and num == -1 then
        return redis.call('DEL', lock_key)
    end
    return redis.error_reply(string.format('ERR no writer: %s', tostring(reply)))
end

local function read_lock(lock_key)
    local reply = redis.call('GET', lock_key)
    local num = tonumber(reply)
    if num ~= nil and num < 0 then
        return 0
    end
    reply = redis.call('INCR', lock_key)
    num = tonumber(reply)
    if num == nil or num <= 0 then
        return redis.error_reply(string.format('ERR Incr lock key reply: %s', tostring(reply)))
    end
    redis.call('EXPIRE', lock_key, 60)
    return 1
end

local function read_unlock(lock_key)
    local reply = redis.call('GET', lock_key)
    local num = tonumber(reply)
    if num == nil or num <= 0 then
        return redis.error_reply('ERR No reader')
    end
    if num == 1 then
        redis.call('DEL', lock_key)
    else
        redis.call('DECR', lock_key)
    end
    return 1
end

local functions = {read_lock, read_unlock, write_lock, write_unlock}
local lock_key = KEYS[1]
local lock_op = tonumber(ARGV[1])
if lock_op == nil or lock_op <= 0 or lock_op > #functions then
    return redis.error_reply(string.format('ERR Unknown lock operation: %s', ARGV[1]))
end
return functions[lock_op](lock_key)

Sign in to access your highlights
Login / Signup