package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type ModelRequestConcurrencyRelease = service.ModelRequestConcurrencyRelease

func ModelRequestConcurrencyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		release, newAPIError := AcquireModelRequestConcurrency(c)
		if newAPIError != nil {
			abortWithOpenAiMessage(c, newAPIError.StatusCode, newAPIError.Error(), newAPIError.GetErrorCode())
			return
		}
		if release == nil {
			c.Next()
			return
		}
		defer release()

		c.Next()
	}
}

func AcquireModelRequestConcurrency(c *gin.Context) (ModelRequestConcurrencyRelease, *types.NewAPIError) {
	if !setting.ModelRequestConcurrencyLimitEnabled {
		return nil, nil
	}

	userID := c.GetInt("id")
	if userID == 0 {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("用户未认证"),
			types.ErrorCodeAccessDenied,
			http.StatusUnauthorized,
			types.ErrOptionWithSkipRetry(),
		)
	}

	limit := common.GetContextKeyInt(c, constant.ContextKeyUserConcurrencyLimit)
	if limit <= 0 {
		limit = setting.ModelRequestConcurrencyLimit
	}
	if limit <= 0 {
		return nil, nil
	}

	release, err := service.TryAcquireModelRequestConcurrency(userID, limit)
	if err == nil {
		return release, nil
	}
	if errors.Is(err, service.ErrModelRequestConcurrencySaturated) {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("您已达到用户并发请求限制：最多同时进行 %d 个模型请求", limit),
			types.ErrorCodeModelRequestConcurrencySaturated,
			http.StatusTooManyRequests,
			types.ErrOptionWithSkipRetry(),
		)
	}
	return nil, types.NewErrorWithStatusCode(
		err,
		types.ErrorCodeModelRequestConcurrencySaturated,
		http.StatusInternalServerError,
		types.ErrOptionWithSkipRetry(),
	)
}
