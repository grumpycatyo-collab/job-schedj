Use this baseline design.
1.ZSET for schedule: sched:due
* member: job_id
* score: run_at_unix_ms
2. HASH per job: job:{id}
* payload, attempts, max_attempts, retry_backoff, recurring_spec, etc.
3. LIST for ready queue: queue:ready
4. LIST for in-flight: queue:processing
Run a dispatcher loop in Go every 100-500ms:

1. call Lua script move_due.lua(now_ms, batch_size)
2. script atomically:

* get due IDs from sched:due (ZRANGE ... BYSCORE)
* LPUSH queue:ready id
* ZREM sched:due id

return moved IDs count
Worker loop in Go:
1. BRPOPLPUSH queue:ready queue:processing 5
2. load job:{id}, execute handler
3. success: LREM queue:processing 1 id, cleanup or schedule next recurrence
4. fail: LREM queue:processing 1 id, compute backoff+jitter, ZADD sched:due next_ts id

Recurring:
1. store recurrence rule in job:{id} (e.g. cron or interval)
2. after each successful run, compute next fire time in Go
3. ZADD sched:due next_ts id

Minimal Lua (move_due.lua):
```lua 
-- KEYS[1]=sched:due, KEYS[2]=queue:ready
-- ARGV[1]=now_ms, ARGV[2]=limit
local ids = redis.call('ZRANGE', KEYS[1], '-inf', ARGV[1], 'BYSCORE', 'LIMIT', 0, ARGV[2])
for _, id in ipairs(ids) do
redis.call('LPUSH', KEYS[2], id)
redis.call('ZREM', KEYS[1], id)
end
return #ids

```
Go call shape:
```go
n, err := rdb.Eval(ctx, moveDueLua, []string{"sched:due", "queue:ready"}, nowMs, batch).Int()
```
Hard requirements:
1. idempotent handlers (duplicates can still happen)
2. dead-letter queue after max attempts
3. reaper for stuck items in queue:processing
4. metrics: due lag, moved/sec, retries, failures