package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	channelConcurrencyRedisKeyPrefix = "channel_concurrency:"
	channelConcurrencyRedisTTL       = 10 * time.Minute
	channelConcurrencyRedisRefresh   = time.Minute
)

var (
	channelConcurrencyMu     sync.Mutex
	channelConcurrencyCounts = make(map[int]int)
)

type ChannelConcurrencyRelease func()

var ErrChannelConcurrencySaturated = errors.New("channel concurrency limit reached")

func TryAcquireChannelConcurrency(channelID int, limit int) (ChannelConcurrencyRelease, error) {
	if channelID <= 0 || limit <= 0 {
		return func() {}, nil
	}

	if common.RedisEnabled && common.RDB != nil {
		return tryAcquireChannelConcurrencyRedis(channelID, limit)
	}

	return tryAcquireChannelConcurrencyMemory(channelID, limit)
}

func tryAcquireChannelConcurrencyRedis(channelID int, limit int) (ChannelConcurrencyRelease, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", channelConcurrencyRedisKeyPrefix, channelID)

	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		common.SysLog(fmt.Sprintf("channel concurrency redis incr failed: channel_id=%d, error=%v", channelID, err))
		return tryAcquireChannelConcurrencyMemory(channelID, limit)
	}
	_ = common.RDB.Expire(ctx, key, channelConcurrencyRedisTTL).Err()

	if count > int64(limit) {
		_, decrErr := common.RDB.Decr(ctx, key).Result()
		if decrErr != nil {
			common.SysLog(fmt.Sprintf("channel concurrency redis rollback failed: channel_id=%d, error=%v", channelID, decrErr))
		}
		return nil, ErrChannelConcurrencySaturated
	}

	done := make(chan struct{})
	go refreshChannelConcurrencyRedisTTL(key, done)

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			next, decrErr := common.RDB.Decr(context.Background(), key).Result()
			if decrErr != nil {
				common.SysLog(fmt.Sprintf("channel concurrency redis release failed: channel_id=%d, error=%v", channelID, decrErr))
				return
			}
			if next <= 0 {
				_ = common.RDB.Del(context.Background(), key).Err()
			}
		})
	}, nil
}

func refreshChannelConcurrencyRedisTTL(key string, done <-chan struct{}) {
	timer := time.NewTimer(channelConcurrencyRedisRefresh)
	defer timer.Stop()
	for {
		select {
		case <-done:
			return
		case <-timer.C:
			if common.RedisEnabled && common.RDB != nil {
				_ = common.RDB.Expire(context.Background(), key, channelConcurrencyRedisTTL).Err()
			}
			timer.Reset(channelConcurrencyRedisRefresh)
		}
	}
}

func tryAcquireChannelConcurrencyMemory(channelID int, limit int) (ChannelConcurrencyRelease, error) {
	channelConcurrencyMu.Lock()
	if channelConcurrencyCounts[channelID] >= limit {
		channelConcurrencyMu.Unlock()
		return nil, ErrChannelConcurrencySaturated
	}
	channelConcurrencyCounts[channelID]++
	channelConcurrencyMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			channelConcurrencyMu.Lock()
			defer channelConcurrencyMu.Unlock()
			current := channelConcurrencyCounts[channelID]
			if current <= 1 {
				delete(channelConcurrencyCounts, channelID)
				return
			}
			channelConcurrencyCounts[channelID] = current - 1
		})
	}, nil
}
