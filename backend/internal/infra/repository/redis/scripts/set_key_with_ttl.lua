-- KEYS: target Redis keys to set
-- ARGV[1]: matchID
-- ARGV[2]: ttl_ms

for i = 1, #KEYS do
  redis.call("SET", KEYS[i], ARGV[1], "PX", ARGV[2])
end

return 1