package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

const codexQuotaAutoReenableTickInterval = time.Minute

var codexQuotaAutoReenableOnce sync.Once

func StartCodexQuotaAutoReenableTask() {
	if !common.IsMasterNode {
		return
	}

	codexQuotaAutoReenableOnce.Do(func() {
		go func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("codex quota auto-reenable task started: tick=%s", codexQuotaAutoReenableTickInterval))
			ticker := time.NewTicker(codexQuotaAutoReenableTickInterval)
			defer ticker.Stop()

			for {
				if err := runCodexQuotaAutoReenablePass(context.Background(), common.GetTimestamp()); err != nil {
					logger.LogWarn(context.Background(), fmt.Sprintf("codex quota auto-reenable task failed: %v", err))
				}
				<-ticker.C
			}
		}()
	})
}
