package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type subscriptionSummaryAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Total int `json:"total"`
		Items []struct {
			Id                  int    `json:"id"`
			Username            string `json:"username"`
			SubscriptionSummary struct {
				ActiveCount      int     `json:"active_count"`
				PrimaryPlanTitle string  `json:"primary_plan_title"`
				TodayUsed        int64   `json:"today_used"`
				LifetimeUsed     int64   `json:"lifetime_used"`
				UsagePercent     float64 `json:"usage_percent"`
				ActualPercent    float64 `json:"actual_usage_percent"`
				ElapsedUsed      int64   `json:"period_elapsed_used"`
				ElapsedQuota     int64   `json:"period_elapsed_quota"`
			} `json:"subscription_summary"`
		} `json:"items"`
	} `json:"data"`
}

func setupSubscriptionSummaryControllerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.SubscriptionUsageDaily{},
	))
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
}

func TestAdminListUserSubscriptionSummariesFiltersAndSorts(t *testing.T) {
	setupSubscriptionSummaryControllerTestDB(t)

	now := common.GetTimestamp()
	startDay := model.SubscriptionUsageDayStart(now) - int64(2*24*time.Hour/time.Second)
	require.NoError(t, model.DB.Create(&model.User{Id: 1, Username: "alpha", Group: "default", Status: common.UserStatusEnabled, AffCode: "summary-alpha"}).Error)
	require.NoError(t, model.DB.Create(&model.User{Id: 2, Username: "beta", Group: "default", Status: common.UserStatusEnabled, AffCode: "summary-beta"}).Error)
	plan := &model.SubscriptionPlan{Title: "Pro", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 3000}
	require.NoError(t, model.DB.Create(plan).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id:              11,
		UserId:          1,
		PlanId:          plan.Id,
		AmountTotal:     3000,
		AmountUsed:      900,
		AmountUsedTotal: 1500,
		StartTime:       startDay,
		EndTime:         startDay + int64(30*24*time.Hour/time.Second),
		Status:          "active",
	}).Error)
	for i := 0; i < 3; i++ {
		require.NoError(t, model.DB.Create(&model.SubscriptionUsageDaily{
			UserId:             1,
			UserSubscriptionId: 11,
			PlanId:             plan.Id,
			DayStart:           model.SubscriptionUsageDayStart(now - int64(time.Duration(i)*24*time.Hour/time.Second)),
			Quota:              50,
			RequestCount:       1,
		}).Error)
	}
	require.NoError(t, model.DB.Create(&model.SubscriptionUsageDaily{
		UserId:             1,
		UserSubscriptionId: 11,
		PlanId:             plan.Id,
		DayStart:           model.SubscriptionUsageDayStart(now - int64(20*24*time.Hour/time.Second)),
		Quota:              999,
		RequestCount:       2,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/subscription/admin/user-subscription-summaries?status=active&sort=today_used&order=desc", nil)

	AdminListUserSubscriptionSummaries(ctx)

	var body subscriptionSummaryAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, "alpha", body.Data.Items[0].Username)
	assert.Equal(t, "Pro", body.Data.Items[0].SubscriptionSummary.PrimaryPlanTitle)
	assert.Equal(t, int64(50), body.Data.Items[0].SubscriptionSummary.TodayUsed)
	assert.Equal(t, int64(1500), body.Data.Items[0].SubscriptionSummary.LifetimeUsed)
	assert.Equal(t, 30.0, body.Data.Items[0].SubscriptionSummary.UsagePercent)
	assert.Equal(t, int64(150), body.Data.Items[0].SubscriptionSummary.ElapsedUsed)
	assert.InDelta(t, int64(300), body.Data.Items[0].SubscriptionSummary.ElapsedQuota, 2)
	assert.InDelta(t, 50.0, body.Data.Items[0].SubscriptionSummary.ActualPercent, 1)
}
