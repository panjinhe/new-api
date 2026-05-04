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
	modelRequestConcurrencyRedisKeyPrefix = "model_request_concurrency:user:"
	modelRequestConcurrencyRedisTTL       = 10 * time.Minute
	modelRequestConcurrencyRedisRefresh   = time.Minute
)

var (
	modelRequestConcurrencyMu     sync.Mutex
	modelRequestConcurrencyCounts = make(map[int]int)
)

type ModelRequestConcurrencyRelease func()

var ErrModelRequestConcurrencySaturated = errors.New("model request concurrency limit reached")

func TryAcquireModelRequestConcurrency(userID int, limit int) (ModelRequestConcurrencyRelease, error) {
	if userID <= 0 || limit <= 0 {
		return func() {}, nil
	}

	if common.RedisEnabled && common.RDB != nil {
		return tryAcquireModelRequestConcurrencyRedis(userID, limit)
	}

	return tryAcquireModelRequestConcurrencyMemory(userID, limit)
}

func tryAcquireModelRequestConcurrencyRedis(userID int, limit int) (ModelRequestConcurrencyRelease, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", modelRequestConcurrencyRedisKeyPrefix, userID)

	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		common.SysLog(fmt.Sprintf("model request concurrency redis incr failed: user_id=%d, error=%v", userID, err))
		return tryAcquireModelRequestConcurrencyMemory(userID, limit)
	}
	_ = common.RDB.Expire(ctx, key, modelRequestConcurrencyRedisTTL).Err()

	if count > int64(limit) {
		_, decrErr := common.RDB.Decr(ctx, key).Result()
		if decrErr != nil {
			common.SysLog(fmt.Sprintf("model request concurrency redis rollback failed: user_id=%d, error=%v", userID, decrErr))
		}
		return nil, ErrModelRequestConcurrencySaturated
	}

	done := make(chan struct{})
	go refreshModelRequestConcurrencyRedisTTL(key, done)

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			next, decrErr := common.RDB.Decr(context.Background(), key).Result()
			if decrErr != nil {
				common.SysLog(fmt.Sprintf("model request concurrency redis release failed: user_id=%d, error=%v", userID, decrErr))
				return
			}
			if next <= 0 {
				_ = common.RDB.Del(context.Background(), key).Err()
			}
		})
	}, nil
}

func refreshModelRequestConcurrencyRedisTTL(key string, done <-chan struct{}) {
	timer := time.NewTimer(modelRequestConcurrencyRedisRefresh)
	defer timer.Stop()
	for {
		select {
		case <-done:
			return
		case <-timer.C:
			if common.RedisEnabled && common.RDB != nil {
				_ = common.RDB.Expire(context.Background(), key, modelRequestConcurrencyRedisTTL).Err()
			}
			timer.Reset(modelRequestConcurrencyRedisRefresh)
		}
	}
}

func tryAcquireModelRequestConcurrencyMemory(userID int, limit int) (ModelRequestConcurrencyRelease, error) {
	modelRequestConcurrencyMu.Lock()
	if modelRequestConcurrencyCounts[userID] >= limit {
		modelRequestConcurrencyMu.Unlock()
		return nil, ErrModelRequestConcurrencySaturated
	}
	modelRequestConcurrencyCounts[userID]++
	modelRequestConcurrencyMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			modelRequestConcurrencyMu.Lock()
			defer modelRequestConcurrencyMu.Unlock()
			current := modelRequestConcurrencyCounts[userID]
			if current <= 1 {
				delete(modelRequestConcurrencyCounts, userID)
				return
			}
			modelRequestConcurrencyCounts[userID] = current - 1
		})
	}, nil
}
