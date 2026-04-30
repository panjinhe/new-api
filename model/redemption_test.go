package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRedemptionTestTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM redemptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_subscriptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM subscription_plans").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func seedRedemptionUser(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		Username: "redeem_user",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)
}

func seedRedemptionPlan(t *testing.T, title string, total int64) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Title:            title,
		PriceAmount:      20,
		Currency:         "USD",
		DurationUnit:     SubscriptionDurationDay,
		DurationValue:    30,
		Enabled:          true,
		TotalAmount:      total,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	require.NoError(t, DB.Create(plan).Error)
	InvalidateSubscriptionPlanCache(plan.Id)
	return plan
}

func seedRedemptionCode(t *testing.T, key string, typ string, quota int, planId int) *Redemption {
	t.Helper()
	redemption := &Redemption{
		UserId:         1,
		Name:           key,
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          quota,
		RedemptionType: typ,
		PlanId:         planId,
		CreatedTime:    common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())
	return redemption
}

func getRedemptionStatus(t *testing.T, id int) int {
	t.Helper()
	var redemption Redemption
	require.NoError(t, DB.Select("status").Where("id = ?", id).First(&redemption).Error)
	return redemption.Status
}

func TestRedeemQuotaCodeAddsWalletQuota(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 101)
	code := seedRedemptionCode(t, "quota-code", RedemptionTypeQuota, 5000, 0)

	result, err := Redeem("quota-code", 101)
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, RedemptionTypeQuota, result.Type)
	assert.Equal(t, 5000, result.Quota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, code.Id))
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", 101).First(&user).Error)
	assert.Equal(t, 5000, user.Quota)
}

func TestRedeemPlanCodeCreatesSubscription(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 102)
	plan := seedRedemptionPlan(t, "Forward One", 1000)
	code := seedRedemptionCode(t, "plan-code", RedemptionTypePlan, 0, plan.Id)

	result, err := Redeem("plan-code", 102)
	require.NoError(t, err)

	require.NotNil(t, result)
	require.NotNil(t, result.Subscription)
	assert.Equal(t, RedemptionTypePlan, result.Type)
	assert.Equal(t, plan.Id, result.Subscription.PlanId)
	assert.Equal(t, plan.Title, result.Subscription.PlanTitle)
	assert.Equal(t, int64(1000), result.Subscription.DailyQuota)
	assert.Equal(t, int64(30000), result.Subscription.TotalQuota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, code.Id))

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", 102, plan.Id).First(&sub).Error)
	assert.Equal(t, "active", sub.Status)
	assert.Equal(t, "redemption", sub.Source)
	assert.Equal(t, code.Id, sub.SourceRedemptionId)
}

func TestRedeemSamePlanCodeExtendsSubscription(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 103)
	plan := seedRedemptionPlan(t, "Forward Two", 3000)
	first := seedRedemptionCode(t, "plan-code-first", RedemptionTypePlan, 0, plan.Id)
	second := seedRedemptionCode(t, "plan-code-second", RedemptionTypePlan, 0, plan.Id)

	firstResult, err := Redeem(first.Key, 103)
	require.NoError(t, err)
	firstEnd := firstResult.Subscription.EndTime

	secondResult, err := Redeem(second.Key, 103)
	require.NoError(t, err)

	require.NotNil(t, secondResult.Subscription)
	assert.True(t, secondResult.Subscription.Extended)
	assert.Greater(t, secondResult.Subscription.EndTime, firstEnd)
	assert.Equal(t, int64(180000), secondResult.Subscription.TotalQuota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, second.Id))
}

func TestRedeemDifferentPlanCodeKeepsCodeUnused(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 104)
	planA := seedRedemptionPlan(t, "Forward Three", 5000)
	planB := seedRedemptionPlan(t, "Lightspeed", 20000)
	first := seedRedemptionCode(t, "plan-code-a", RedemptionTypePlan, 0, planA.Id)
	second := seedRedemptionCode(t, "plan-code-b", RedemptionTypePlan, 0, planB.Id)

	_, err := Redeem(first.Key, 104)
	require.NoError(t, err)
	_, err = Redeem(second.Key, 104)
	require.Error(t, err)

	assert.True(t, strings.Contains(err.Error(), "已有生效中的其他套餐"))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
}
