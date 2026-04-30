package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertUserForClassificationTest(t *testing.T, id int, username string, group string, usedQuota int, role int) {
	t.Helper()
	user := &User{
		Id:          id,
		Username:    username,
		Status:      common.UserStatusEnabled,
		Group:       group,
		UsedQuota:   usedQuota,
		Role:        role,
		AffCode:     fmt.Sprintf("classification-aff-%d", id),
		AccessToken: nil,
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertTopUpForClassificationTest(t *testing.T, userId int, money float64, status string, tradeNo string) {
	t.Helper()
	require.NoError(t, DB.Create(&TopUp{
		UserId:  userId,
		Money:   money,
		TradeNo: tradeNo,
		Status:  status,
	}).Error)
}

func getUserGroupForClassificationTest(t *testing.T, userId int) string {
	t.Helper()
	var group string
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userId).Select(subscriptionUserGroupColumn()).Scan(&group).Error)
	return group
}

func TestClassifyUsersByPaymentAndUsage_AssignsPaidAndFreeGroups(t *testing.T) {
	truncateTables(t)

	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldExchangeRate := operation_setting.USDExchangeRate
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate = 10
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
		operation_setting.USDExchangeRate = oldExchangeRate
	})

	usedQuotaThreshold := displayAmountToQuotaThreshold(50)

	insertUserForClassificationTest(t, 901, "paid-topup", "default", 0, common.RoleCommonUser)
	insertTopUpForClassificationTest(t, 901, 25, common.TopUpStatusSuccess, "paid-topup-1")
	insertTopUpForClassificationTest(t, 901, 25, common.TopUpStatusSuccess, "paid-topup-2")

	insertUserForClassificationTest(t, 902, "paid-usage", "default", usedQuotaThreshold, common.RoleCommonUser)

	insertUserForClassificationTest(t, 903, "paid-subscription", "default", 0, common.RoleCommonUser)
	require.NoError(t, DB.Create(&UserSubscription{UserId: 903, PlanId: 1, Status: "expired"}).Error)

	insertUserForClassificationTest(t, 904, "free-user", "default", 0, common.RoleCommonUser)
	insertTopUpForClassificationTest(t, 904, 100, common.TopUpStatusPending, "pending-topup")

	insertUserForClassificationTest(t, 905, "admin-user", "default", 0, common.RoleAdminUser)

	result, err := ClassifyUsersByPaymentAndUsage(UserGroupClassificationOptions{
		AmountThreshold: 50,
		PaidGroup:       DefaultPaidUserGroup,
		FreeGroup:       DefaultFreeloadingUserGroup,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(4), result.TotalUsers)
	assert.Equal(t, int64(3), result.PaidUsers)
	assert.Equal(t, int64(1), result.FreeUsers)
	assert.Equal(t, int64(4), result.UpdatedUsers)
	assert.Equal(t, usedQuotaThreshold, result.UsedQuotaThreshold)
	assert.Equal(t, DefaultPaidUserGroup, getUserGroupForClassificationTest(t, 901))
	assert.Equal(t, DefaultPaidUserGroup, getUserGroupForClassificationTest(t, 902))
	assert.Equal(t, DefaultPaidUserGroup, getUserGroupForClassificationTest(t, 903))
	assert.Equal(t, DefaultFreeloadingUserGroup, getUserGroupForClassificationTest(t, 904))
	assert.Equal(t, "default", getUserGroupForClassificationTest(t, 905))
}

func TestClassifyUsersByPaymentAndUsage_RejectsSameGroups(t *testing.T) {
	truncateTables(t)

	result, err := ClassifyUsersByPaymentAndUsage(UserGroupClassificationOptions{
		AmountThreshold: 50,
		PaidGroup:       "same",
		FreeGroup:       "same",
	})

	require.Error(t, err)
	assert.Nil(t, result)
}
