package redis

import (
	"context"
	_ "embed"
	"errors"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	red "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stringx"
)

const (
	randomLen       = 16
	tolerance       = 500 // milliseconds
	millisPerSecond = 1000
)

/*
--KEYS[1]: 锁key
--ARGV[1]: 锁value,随机字符串
--ARGV[2]: 过期时间
--判断锁key持有的value是否等于传入的value
--如果key存在且value相等说明是再次获取锁并更新获取时间，防止重入时过期
--这里说明是“可重入锁”


--锁key.value不等于传入的value则说明是第一次获取锁
--SET key value NX PX timeout : 当key不存在时才设置key的值
--设置成功会自动返回“OK”，设置失败返回“NULL Bulk Reply”
--为什么这里要加“NX”呢，因为需要防止把别人的锁给覆盖了


-- 2个携程各自创建key同样的RedisLock,这样2个RedisLock的id值就不一样。
-- A先来，A发现Key不存在，redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2]返回OK,获取锁成功
-- 此时B来了，发现redis.call("GET", KEYS[1]) ！= ARGV[1]，
-- 但是就会调用redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
-- key没有过期或则还没被A释放，那B执行SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2] 返回Nil
--若key过期或者key已经被A释放，那BSET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2] 返回OK,获取锁成功
-- 若A再次获取锁(他没有释放锁)，redis.call("GET", KEYS[1]) == ARGV[1],就让它再次得到锁，锁是可重入的

*/

var (
	//go:embed lockscript.lua
	lockLuaScript string
	lockScript    = NewScript(lockLuaScript)

	//go:embed delscript.lua
	delLuaScript string
	delScript    = NewScript(delLuaScript)
)

// A RedisLock is a redis lock.
type RedisLock struct {
	store   *Redis
	seconds uint32 //默认是0
	key     string
	id      string
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

// NewRedisLock returns a RedisLock.
func NewRedisLock(store *Redis, key string) *RedisLock {
	return &RedisLock{
		store: store,
		key:   key,
		id:    stringx.Randn(randomLen),
	}
}

// Acquire acquires the lock.
func (rl *RedisLock) Acquire() (bool, error) {
	return rl.AcquireCtx(context.Background())
}

// AcquireCtx acquires the lock with the given ctx.
func (rl *RedisLock) AcquireCtx(ctx context.Context) (bool, error) {
	seconds := atomic.LoadUint32(&rl.seconds)
	resp, err := rl.store.ScriptRunCtx(ctx, lockScript, []string{rl.key}, []string{
		rl.id, strconv.Itoa(int(seconds)*millisPerSecond + tolerance),
	})
	//不存在key,才设置.如key存在，lua返回nil
	if errors.Is(err, red.Nil) {
		return false, nil
	} else if err != nil {
		logx.Errorf("Error on acquiring lock for %s, %s", rl.key, err.Error())
		return false, err
	} else if resp == nil {
		return false, nil
	}

	reply, ok := resp.(string)
	if ok && reply == "OK" {
		return true, nil
	}

	logx.Errorf("Unknown reply when acquiring lock for %s: %v", rl.key, resp)
	return false, nil
}

// Release releases the lock.
func (rl *RedisLock) Release() (bool, error) {
	return rl.ReleaseCtx(context.Background())
}

// ReleaseCtx releases the lock with the given ctx.
func (rl *RedisLock) ReleaseCtx(ctx context.Context) (bool, error) {
	resp, err := rl.store.ScriptRunCtx(ctx, delScript, []string{rl.key}, []string{rl.id})
	if err != nil {
		return false, err
	}

	reply, ok := resp.(int64)
	if !ok {
		return false, nil
	}

	return reply == 1, nil
}

// SetExpire sets the expiration.
func (rl *RedisLock) SetExpire(seconds int) {
	atomic.StoreUint32(&rl.seconds, uint32(seconds))
}
