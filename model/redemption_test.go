package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRedemptionTestTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM redemptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM welfare_redemption_daily_claims").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_subscriptions").Error)
	require.NoError(t, DB.Exec("DELETE FROM quota_bucket_pre_consume_allocations").Error)
	require.NoError(t, DB.Exec("DELETE FROM quota_bucket_pre_consume_records").Error)
	require.NoError(t, DB.Exec("DELETE FROM quota_buckets").Error)
	require.NoError(t, DB.Exec("DELETE FROM subscription_plans").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
}

func seedRedemptionUser(t *testing.T, id int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		Username: fmt.Sprintf("redeem_user_%d", id),
		Email:    fmt.Sprintf("redeem_user_%d@example%d.com", id, id),
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  fmt.Sprintf("redeem_aff_%d", id),
	}).Error)
}

func seedRedemptionUserWithEmail(t *testing.T, id int, email string) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:       id,
		Username: fmt.Sprintf("redeem_user_%d", id),
		Email:    email,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  fmt.Sprintf("redeem_aff_%d", id),
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

func seedBucketRedemptionCode(t *testing.T, key string, quota int, durationSeconds int64) *Redemption {
	t.Helper()
	redemption := &Redemption{
		UserId:                1,
		Name:                  key,
		Key:                   key,
		Status:                common.RedemptionCodeStatusEnabled,
		Quota:                 quota,
		RedemptionType:        RedemptionTypeBucket,
		BucketDurationSeconds: durationSeconds,
		CreatedTime:           common.GetTimestamp(),
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

func getRedemptionUserQuota(t *testing.T, id int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", id).First(&user).Error)
	return user.Quota
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

func TestRedeemOneTimeWelfareCodeOnlyOncePerUser(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 105)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-first", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "welfare-code-second", RedemptionTypeQuota, welfareQuota, 0)

	result, err := Redeem(first.Key, 105)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, welfareQuota, result.Quota)

	result, err = Redeem(second.Key, 105)
	require.Error(t, err)
	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareAlreadyRedeemed))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 105))
}

func TestRedeemOneTimeWelfareQuotaHistoryBlocksBucket(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 112)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-quota-before-bucket", RedemptionTypeQuota, welfareQuota, 0)
	second := seedBucketRedemptionCode(t, "welfare-bucket-after-quota", welfareQuota, int64(2*24*3600))

	_, err := Redeem(first.Key, 112)
	require.NoError(t, err)
	result, err := Redeem(second.Key, 112)
	require.Error(t, err)

	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareAlreadyRedeemed))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 112))

	var bucketCount int64
	require.NoError(t, DB.Model(&QuotaBucket{}).Where("user_id = ?", 112).Count(&bucketCount).Error)
	assert.Zero(t, bucketCount)
}

func TestRedeemOneTimeWelfareBucketHistoryBlocksQuota(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 113)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedBucketRedemptionCode(t, "welfare-bucket-before-quota", welfareQuota, int64(2*24*3600))
	second := seedRedemptionCode(t, "welfare-quota-after-bucket", RedemptionTypeQuota, welfareQuota, 0)

	result, err := Redeem(first.Key, 113)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Bucket)
	assert.Equal(t, int64(welfareQuota), result.Bucket.Bucket.AmountTotal)

	result, err = Redeem(second.Key, 113)
	require.Error(t, err)
	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareAlreadyRedeemed))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
	assert.Equal(t, 0, getRedemptionUserQuota(t, 113))
}

func TestRedeemOneTimeWelfareBucketDurationDoesNotAffectLimit(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 114)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedBucketRedemptionCode(t, "welfare-bucket-two-days", welfareQuota, int64(2*24*3600))
	second := seedBucketRedemptionCode(t, "welfare-bucket-seven-days", welfareQuota, DefaultQuotaBucketDurationSeconds)

	_, err := Redeem(first.Key, 114)
	require.NoError(t, err)
	result, err := Redeem(second.Key, 114)
	require.Error(t, err)

	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareAlreadyRedeemed))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))

	var bucketCount int64
	require.NoError(t, DB.Model(&QuotaBucket{}).Where("user_id = ?", 114).Count(&bucketCount).Error)
	assert.Equal(t, int64(1), bucketCount)
}

func TestRedeemOneTimeWelfareCodeAllowsDifferentUsers(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 106)
	seedRedemptionUser(t, 107)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-user-a", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "welfare-code-user-b", RedemptionTypeQuota, welfareQuota, 0)

	_, err := Redeem(first.Key, 106)
	require.NoError(t, err)
	result, err := Redeem(second.Key, 107)
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, welfareQuota, result.Quota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 106))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 107))
}

func TestRedeemOneTimeWelfareCodeBlocksSameIpDaily(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 115)
	seedRedemptionUser(t, 116)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-ip-a", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "welfare-code-ip-b", RedemptionTypeQuota, welfareQuota, 0)

	_, err := RedeemWithAudit(first.Key, 115, RedemptionAudit{CallerIp: "203.0.113.10"})
	require.NoError(t, err)
	result, err := RedeemWithAudit(second.Key, 116, RedemptionAudit{CallerIp: "203.0.113.10"})
	require.Error(t, err)

	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareDailyLimit))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 115))
	assert.Equal(t, 0, getRedemptionUserQuota(t, 116))
}

func TestRedeemOneTimeWelfareCodeAllowsSamePublicEmailDomainDaily(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUserWithEmail(t, 117, "first@example.com")
	seedRedemptionUserWithEmail(t, 118, "second@example.com")
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-domain-a", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "welfare-code-domain-b", RedemptionTypeQuota, welfareQuota, 0)

	_, err := RedeemWithAudit(first.Key, 117, RedemptionAudit{CallerIp: "203.0.113.11"})
	require.NoError(t, err)
	result, err := RedeemWithAudit(second.Key, 118, RedemptionAudit{CallerIp: "203.0.113.12"})
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, welfareQuota, result.Quota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 117))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 118))
}

func TestRedeemNonWelfareCodeNotLimitedByDailyWelfareClaims(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUserWithEmail(t, 119, "first@example.org")
	seedRedemptionUserWithEmail(t, 120, "second@example.org")
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-daily-history", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "paid-code-same-risk", RedemptionTypeQuota, welfareQuota+1, 0)

	_, err := RedeemWithAudit(first.Key, 119, RedemptionAudit{CallerIp: "203.0.113.13"})
	require.NoError(t, err)
	result, err := RedeemWithAudit(second.Key, 120, RedemptionAudit{CallerIp: "203.0.113.13"})
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, welfareQuota+1, result.Quota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, first.Id))
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 119))
	assert.Equal(t, welfareQuota+1, getRedemptionUserQuota(t, 120))
}

func TestRedeemNonWelfareQuotaCodeNotLimitedByWelfareHistory(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 108)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-history", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "non-welfare-code", RedemptionTypeQuota, welfareQuota+1, 0)

	_, err := Redeem(first.Key, 108)
	require.NoError(t, err)
	result, err := Redeem(second.Key, 108)
	require.NoError(t, err)

	require.NotNil(t, result)
	assert.Equal(t, welfareQuota+1, result.Quota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota+welfareQuota+1, getRedemptionUserQuota(t, 108))
}

func TestRedeemOneTimeWelfareCodeCountsSoftDeletedHistory(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 109)
	welfareQuota := oneTimeWelfareRedemptionQuota()
	first := seedRedemptionCode(t, "welfare-code-deleted-history", RedemptionTypeQuota, welfareQuota, 0)
	second := seedRedemptionCode(t, "welfare-code-after-deleted-history", RedemptionTypeQuota, welfareQuota, 0)

	_, err := Redeem(first.Key, 109)
	require.NoError(t, err)
	require.NoError(t, DB.Delete(first).Error)
	result, err := Redeem(second.Key, 109)
	require.Error(t, err)

	require.Nil(t, result)
	assert.True(t, errors.Is(err, ErrRedemptionWelfareAlreadyRedeemed))
	assert.Equal(t, common.RedemptionCodeStatusEnabled, getRedemptionStatus(t, second.Id))
	assert.Equal(t, welfareQuota, getRedemptionUserQuota(t, 109))
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

func TestRedeemBucketCodeCreatesQuotaBucketOnly(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 110)
	code := seedBucketRedemptionCode(t, "bucket-code", 7000, DefaultQuotaBucketDurationSeconds)

	result, err := Redeem(code.Key, 110)
	require.NoError(t, err)

	require.NotNil(t, result)
	require.NotNil(t, result.Bucket)
	assert.Equal(t, RedemptionTypeBucket, result.Type)
	assert.Equal(t, 7000, result.Quota)
	assert.Equal(t, int64(7000), result.Bucket.Bucket.AmountTotal)
	assert.Equal(t, int64(7000), result.Bucket.RemainingQuota)
	assert.Equal(t, common.RedemptionCodeStatusUsed, getRedemptionStatus(t, code.Id))
	assert.Equal(t, 0, getRedemptionUserQuota(t, 110))

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", 110).Count(&subCount).Error)
	assert.Zero(t, subCount)
}

func TestQuotaBucketsConsumeEarliestAndRefund(t *testing.T) {
	truncateTables(t)
	resetRedemptionTestTables(t)
	seedRedemptionUser(t, 111)
	now := common.GetTimestamp()
	require.NoError(t, DB.Create(&QuotaBucket{
		UserId:      111,
		Title:       "later",
		AmountTotal: 500,
		StartTime:   now,
		EndTime:     now + 7200,
		Status:      QuotaBucketStatusActive,
		Source:      QuotaBucketSourceRedemption,
	}).Error)
	require.NoError(t, DB.Create(&QuotaBucket{
		UserId:      111,
		Title:       "earlier",
		AmountTotal: 300,
		StartTime:   now,
		EndTime:     now + 3600,
		Status:      QuotaBucketStatusActive,
		Source:      QuotaBucketSourceRedemption,
	}).Error)

	res, err := PreConsumeUserQuotaBuckets("bucket-request-1", 111, 600)
	require.NoError(t, err)
	assert.Equal(t, int64(600), res.PreConsumed)

	var earlier QuotaBucket
	require.NoError(t, DB.Where("user_id = ? AND title = ?", 111, "earlier").First(&earlier).Error)
	assert.Equal(t, int64(300), earlier.AmountUsed)
	assert.Equal(t, QuotaBucketStatusEmpty, earlier.Status)

	var later QuotaBucket
	require.NoError(t, DB.Where("user_id = ? AND title = ?", 111, "later").First(&later).Error)
	assert.Equal(t, int64(300), later.AmountUsed)
	assert.Equal(t, QuotaBucketStatusActive, later.Status)

	require.NoError(t, PostConsumeQuotaBucketDelta("bucket-request-1", -200))
	require.NoError(t, DB.Where("id = ?", later.Id).First(&later).Error)
	assert.Equal(t, int64(100), later.AmountUsed)
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
