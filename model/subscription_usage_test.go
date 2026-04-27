package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubscriptionUsagePlanAndSub(t *testing.T, userId int) (int, int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       userId,
		Username: "sub_usage_user",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)
	plan := &SubscriptionPlan{
		Title:            "Usage Plan",
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1000,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	require.NoError(t, DB.Create(plan).Error)
	now := common.GetTimestamp()
	sub := &UserSubscription{
		UserId:        userId,
		PlanId:        plan.Id,
		AmountTotal:   1000,
		AmountUsed:    0,
		StartTime:     now - 3600,
		EndTime:       now + 86400*30,
		Status:        "active",
		Source:        "admin",
		LastResetTime: now - 86400,
		NextResetTime: now + 86400,
	}
	require.NoError(t, DB.Create(sub).Error)
	return plan.Id, sub.Id
}

func readSubscriptionUsageState(t *testing.T, subId int) (amountUsed int64, amountUsedTotal int64, dailyQuota int64, dailyCount int) {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", subId).First(&sub).Error)
	var daily SubscriptionUsageDaily
	err := DB.Where("user_subscription_id = ? AND day_start = ?", subId, SubscriptionUsageDayStart(common.GetTimestamp())).First(&daily).Error
	if err != nil {
		return sub.AmountUsed, sub.AmountUsedTotal, 0, 0
	}
	return sub.AmountUsed, sub.AmountUsedTotal, daily.Quota, daily.RequestCount
}

func TestSubscriptionUsageTracksPreconsumeDeltaRefundAndReset(t *testing.T) {
	truncateTables(t)
	_, subId := seedSubscriptionUsagePlanAndSub(t, 9101)

	res, err := PreConsumeUserSubscription("sub-usage-req-1", 9101, "test-model", 0, 100)
	require.NoError(t, err)
	require.Equal(t, subId, res.UserSubscriptionId)

	amountUsed, amountUsedTotal, dailyQuota, dailyCount := readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(100), amountUsed)
	assert.Equal(t, int64(100), amountUsedTotal)
	assert.Equal(t, int64(100), dailyQuota)
	assert.Equal(t, 1, dailyCount)

	require.NoError(t, PostConsumeUserSubscriptionDelta(subId, 25))
	amountUsed, amountUsedTotal, dailyQuota, dailyCount = readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(125), amountUsed)
	assert.Equal(t, int64(125), amountUsedTotal)
	assert.Equal(t, int64(125), dailyQuota)
	assert.Equal(t, 1, dailyCount)

	require.NoError(t, PostConsumeUserSubscriptionDelta(subId, -50))
	amountUsed, amountUsedTotal, dailyQuota, dailyCount = readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(75), amountUsed)
	assert.Equal(t, int64(75), amountUsedTotal)
	assert.Equal(t, int64(75), dailyQuota)
	assert.Equal(t, 1, dailyCount)

	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", subId).Updates(map[string]interface{}{
		"next_reset_time": common.GetTimestamp() - 1,
		"last_reset_time": common.GetTimestamp() - 86400,
	}).Error)
	resetCount, err := ResetDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, resetCount)

	amountUsed, amountUsedTotal, dailyQuota, dailyCount = readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(0), amountUsed)
	assert.Equal(t, int64(75), amountUsedTotal)
	assert.Equal(t, int64(75), dailyQuota)
	assert.Equal(t, 1, dailyCount)
}

func TestAdminResetUserSubscriptionCurrentUsageKeepsUsageStats(t *testing.T) {
	truncateTables(t)
	planId, subId := seedSubscriptionUsagePlanAndSub(t, 9104)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", subId).Updates(map[string]interface{}{
		"amount_used":       800,
		"amount_used_total": 1200,
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionUsageDaily{
		UserId:             9104,
		UserSubscriptionId: subId,
		PlanId:             planId,
		DayStart:           SubscriptionUsageDayStart(common.GetTimestamp()),
		Quota:              800,
		RequestCount:       3,
	}).Error)

	msg, err := AdminResetUserSubscriptionCurrentUsage(subId)
	require.NoError(t, err)
	assert.NotEmpty(t, msg)

	amountUsed, amountUsedTotal, dailyQuota, dailyCount := readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(0), amountUsed)
	assert.Equal(t, int64(1200), amountUsedTotal)
	assert.Equal(t, int64(800), dailyQuota)
	assert.Equal(t, 3, dailyCount)
}

func TestSubscriptionUsageRefundPreconsumeIsIdempotent(t *testing.T) {
	truncateTables(t)
	_, subId := seedSubscriptionUsagePlanAndSub(t, 9102)

	_, err := PreConsumeUserSubscription("sub-usage-refund-req", 9102, "test-model", 0, 100)
	require.NoError(t, err)
	require.NoError(t, RefundSubscriptionPreConsume("sub-usage-refund-req"))
	require.NoError(t, RefundSubscriptionPreConsume("sub-usage-refund-req"))

	amountUsed, amountUsedTotal, dailyQuota, dailyCount := readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(0), amountUsed)
	assert.Equal(t, int64(0), amountUsedTotal)
	assert.Equal(t, int64(0), dailyQuota)
	assert.Equal(t, 1, dailyCount)
}

func TestSubscriptionActualUsageWindowForSub(t *testing.T) {
	now := int64(1700864000)
	startDay := SubscriptionUsageDayStart(now) - 2*86400
	sub := UserSubscription{
		AmountTotal:   3000,
		AmountUsed:    1200,
		StartTime:     startDay,
		EndTime:       startDay + 30*86400,
		LastResetTime: startDay,
		NextResetTime: startDay + 30*86400,
	}

	window := subscriptionActualUsageWindowForSub(sub, now)

	assert.Equal(t, startDay, window.StartDay)
	assert.Equal(t, SubscriptionUsageDayStart(now), window.EndDay)
	assert.Equal(t, int64(300), window.TheoreticalQuota)
}

func TestSubscriptionActualUsageWindowCountsTodayAsFullDay(t *testing.T) {
	dayStart := SubscriptionUsageDayStart(1700864000)
	now := dayStart + 13*3600
	sub := UserSubscription{
		AmountTotal:   10000,
		AmountUsed:    9984,
		StartTime:     dayStart,
		EndTime:       dayStart + 86400,
		LastResetTime: dayStart,
		NextResetTime: dayStart + 86400,
	}

	window := subscriptionActualUsageWindowForSub(sub, now)

	assert.Equal(t, dayStart, window.StartDay)
	assert.Equal(t, dayStart, window.EndDay)
	assert.Equal(t, int64(10000), window.TheoreticalQuota)
}

func TestBackfillSubscriptionUsageFromLogsIsIdempotent(t *testing.T) {
	truncateTables(t)
	_, subId := seedSubscriptionUsagePlanAndSub(t, 9103)
	now := time.Now().Unix()

	consumeOther, err := common.Marshal(map[string]interface{}{
		"billing_source":        "subscription",
		"subscription_id":       subId,
		"subscription_consumed": 120,
	})
	require.NoError(t, err)
	refundOther, err := common.Marshal(map[string]interface{}{
		"billing_source":        "subscription",
		"subscription_id":       subId,
		"subscription_consumed": 20,
	})
	require.NoError(t, err)
	missingOther, err := common.Marshal(map[string]interface{}{
		"billing_source": "subscription",
	})
	require.NoError(t, err)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    9103,
		CreatedAt: now,
		Type:      LogTypeConsume,
		Quota:     120,
		Other:     string(consumeOther),
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    9103,
		CreatedAt: now,
		Type:      LogTypeRefund,
		Quota:     20,
		Other:     string(refundOther),
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    9103,
		CreatedAt: now,
		Type:      LogTypeConsume,
		Quota:     50,
		Other:     string(missingOther),
	}).Error)

	stats, err := BackfillSubscriptionUsageFromLogs(0, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.ScannedLogs)
	assert.Equal(t, int64(2), stats.AppliedLogs)
	assert.Equal(t, int64(1), stats.SkippedMissingFields)

	_, amountUsedTotal, dailyQuota, dailyCount := readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(100), amountUsedTotal)
	assert.Equal(t, int64(100), dailyQuota)
	assert.Equal(t, 1, dailyCount)

	stats, err = BackfillSubscriptionUsageFromLogs(0, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.AppliedLogs)
	_, amountUsedTotal, dailyQuota, dailyCount = readSubscriptionUsageState(t, subId)
	assert.Equal(t, int64(100), amountUsedTotal)
	assert.Equal(t, int64(100), dailyQuota)
	assert.Equal(t, 1, dailyCount)
}
