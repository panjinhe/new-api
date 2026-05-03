package model

import (
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

const (
	DefaultPaidUserGroup        = "充值用户"
	DefaultFreeloadingUserGroup = "白嫖怪"
	DefaultPaidAmountThreshold  = 50
)

type UserGroupClassificationOptions struct {
	AmountThreshold float64 `json:"amount_threshold"`
	PaidGroup       string  `json:"paid_group"`
	FreeGroup       string  `json:"free_group"`
}

type UserGroupClassificationResult struct {
	AmountThreshold     float64 `json:"amount_threshold"`
	UsedQuotaThreshold  int     `json:"used_quota_threshold"`
	PaidGroup           string  `json:"paid_group"`
	FreeGroup           string  `json:"free_group"`
	TotalUsers          int64   `json:"total_users"`
	PaidUsers           int64   `json:"paid_users"`
	FreeUsers           int64   `json:"free_users"`
	PaidUsersUpdated    int64   `json:"paid_users_updated"`
	FreeUsersUpdated    int64   `json:"free_users_updated"`
	UpdatedUsers        int64   `json:"updated_users"`
	TopUpQualifiedUsers int64   `json:"topup_qualified_users"`
	UsageQualifiedUsers int64   `json:"usage_qualified_users"`
	SubscriptionUsers   int64   `json:"subscription_users"`
	RedemptionUsers     int64   `json:"redemption_users"`
	QuotaBucketUsers    int64   `json:"quota_bucket_users"`
}

func normalizeUserGroupClassificationOptions(options UserGroupClassificationOptions) (UserGroupClassificationOptions, error) {
	options.PaidGroup = strings.TrimSpace(options.PaidGroup)
	if options.PaidGroup == "" {
		options.PaidGroup = DefaultPaidUserGroup
	}
	options.FreeGroup = strings.TrimSpace(options.FreeGroup)
	if options.FreeGroup == "" {
		options.FreeGroup = DefaultFreeloadingUserGroup
	}
	if options.AmountThreshold <= 0 {
		options.AmountThreshold = DefaultPaidAmountThreshold
	}
	if options.PaidGroup == options.FreeGroup {
		return options, errors.New("充值用户分组和白嫖怪分组不能相同")
	}
	return options, nil
}

func NormalizeManagedUserGroup(group string) string {
	switch strings.TrimSpace(group) {
	case DefaultPaidUserGroup:
		return DefaultPaidUserGroup
	default:
		return DefaultFreeloadingUserGroup
	}
}

func normalizeNewUserGroup(user *User) {
	if user == nil || user.Role != common.RoleCommonUser {
		return
	}
	user.Group = NormalizeManagedUserGroup(user.Group)
}

func displayAmountToQuotaThreshold(amount float64) int {
	if amount <= 0 {
		return 0
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if rate <= 0 {
		rate = 1
	}
	return int(math.Ceil(amount / rate * common.QuotaPerUnit))
}

func commonUserQuery(tx *gorm.DB) *gorm.DB {
	return tx.Model(&User{}).Where("role = ?", common.RoleCommonUser)
}

func successfulTopUpQualifiedUserQuery(tx *gorm.DB, threshold float64) *gorm.DB {
	return tx.Model(&TopUp{}).
		Select("user_id").
		Where("status = ?", common.TopUpStatusSuccess).
		Group("user_id").
		Having("COALESCE(SUM(money), 0) >= ?", threshold)
}

func redemptionQuotaThreshold(amount float64) int {
	if amount <= 0 {
		return 0
	}
	return int(math.Ceil(amount * common.QuotaPerUnit))
}

func successfulRedemptionQualifiedUserQuery(tx *gorm.DB, threshold float64) *gorm.DB {
	return tx.Model(&Redemption{}).
		Select("used_user_id").
		Where("status = ?", common.RedemptionCodeStatusUsed).
		Where("redemption_type = ?", RedemptionTypeQuota).
		Where("used_user_id > 0").
		Where("quota >= ?", redemptionQuotaThreshold(threshold)).
		Group("used_user_id")
}

func subscriptionUserQuery(tx *gorm.DB) *gorm.DB {
	return tx.Model(&UserSubscription{}).Select("DISTINCT user_id")
}

func quotaBucketUserQuery(tx *gorm.DB) *gorm.DB {
	return tx.Model(&QuotaBucket{}).Select("DISTINCT user_id")
}

func classifyChangedUsers(tx *gorm.DB, ids []int, group string) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var changedIDs []int
	err := commonUserQuery(tx).
		Where("id IN ?", ids).
		Where(subscriptionUserGroupColumn()+" <> ?", group).
		Pluck("id", &changedIDs).Error
	return changedIDs, err
}

func updateUsersGroupByIds(tx *gorm.DB, ids []int, group string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := tx.Model(&User{}).
		Where("id IN ?", ids).
		Update("group", group)
	return result.RowsAffected, result.Error
}

func updateUserGroupTx(tx *gorm.DB, userId int, group string) (bool, error) {
	if tx == nil {
		tx = DB
	}
	if userId <= 0 {
		return false, nil
	}
	result := tx.Model(&User{}).
		Where("id = ? AND role = ?", userId, common.RoleCommonUser).
		Where(subscriptionUserGroupColumn()+" <> ?", group).
		Update("group", group)
	return result.RowsAffected > 0, result.Error
}

func promoteUserToPaidGroupTx(tx *gorm.DB, userId int) (bool, error) {
	return updateUserGroupTx(tx, userId, DefaultPaidUserGroup)
}

func invalidateClassifiedUserCaches(userIds []int) {
	for _, userId := range userIds {
		_ = InvalidateUserCache(userId)
		_ = InvalidateUserTokensCache(userId)
	}
}

func promoteUserToPaidGroup(userId int) {
	var changed bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		changed, err = promoteUserToPaidGroupTx(tx, userId)
		return err
	})
	if err != nil {
		common.SysLog("failed to promote user group: " + err.Error())
		return
	}
	if changed {
		invalidateClassifiedUserCaches([]int{userId})
	}
}

func promoteUserToPaidGroupIfTopUpQualified(userId int) {
	if userId <= 0 {
		return
	}
	var changed bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var total float64
		if err := tx.Model(&TopUp{}).
			Select("COALESCE(SUM(money), 0)").
			Where("user_id = ? AND status = ?", userId, common.TopUpStatusSuccess).
			Scan(&total).Error; err != nil {
			return err
		}
		if total < DefaultPaidAmountThreshold {
			return nil
		}
		var err error
		changed, err = promoteUserToPaidGroupTx(tx, userId)
		return err
	})
	if err != nil {
		common.SysLog("failed to classify user after topup: " + err.Error())
		return
	}
	if changed {
		invalidateClassifiedUserCaches([]int{userId})
	}
}

func promoteUserToPaidGroupIfRedemptionQualified(userId int) {
	if userId <= 0 {
		return
	}
	var changed bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&QuotaBucket{}).
			Where("user_id = ?", userId).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Model(&Redemption{}).
				Where("used_user_id = ?", userId).
				Where("status = ?", common.RedemptionCodeStatusUsed).
				Where("redemption_type = ?", RedemptionTypeQuota).
				Where("quota >= ?", redemptionQuotaThreshold(DefaultPaidAmountThreshold)).
				Count(&count).Error; err != nil {
				return err
			}
		}
		if count == 0 {
			return nil
		}
		var err error
		changed, err = promoteUserToPaidGroupTx(tx, userId)
		return err
	})
	if err != nil {
		common.SysLog("failed to classify user after redemption: " + err.Error())
		return
	}
	if changed {
		invalidateClassifiedUserCaches([]int{userId})
	}
}

func promoteUserToPaidGroupIfUsageQualified(userId int) {
	if userId <= 0 {
		return
	}
	threshold := displayAmountToQuotaThreshold(DefaultPaidAmountThreshold)
	if threshold <= 0 {
		return
	}
	var changed bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Model(&User{}).
			Select("id, role, "+subscriptionUserGroupColumn()+", used_quota").
			Where("id = ?", userId).
			First(&user).Error; err != nil {
			return err
		}
		if user.Role != common.RoleCommonUser || user.Group == DefaultPaidUserGroup || user.UsedQuota < threshold {
			return nil
		}
		var err error
		changed, err = promoteUserToPaidGroupTx(tx, userId)
		return err
	})
	if err != nil {
		common.SysLog("failed to classify user after usage update: " + err.Error())
		return
	}
	if changed {
		invalidateClassifiedUserCaches([]int{userId})
	}
}

func ClassifyUsersByPaymentAndUsage(options UserGroupClassificationOptions) (*UserGroupClassificationResult, error) {
	options, err := normalizeUserGroupClassificationOptions(options)
	if err != nil {
		return nil, err
	}

	usedQuotaThreshold := displayAmountToQuotaThreshold(options.AmountThreshold)
	result := &UserGroupClassificationResult{
		AmountThreshold:    options.AmountThreshold,
		UsedQuotaThreshold: usedQuotaThreshold,
		PaidGroup:          options.PaidGroup,
		FreeGroup:          options.FreeGroup,
	}

	var paidIDs []int
	var freeIDs []int
	var paidChangedIDs []int
	var freeChangedIDs []int

	err = DB.Transaction(func(tx *gorm.DB) error {
		paidTopUpUsers := successfulTopUpQualifiedUserQuery(tx, options.AmountThreshold)
		redemptionUsers := successfulRedemptionQualifiedUserQuery(tx, options.AmountThreshold)
		subscriptionUsers := subscriptionUserQuery(tx)
		bucketUsers := quotaBucketUserQuery(tx)

		if err := commonUserQuery(tx).Count(&result.TotalUsers).Error; err != nil {
			return err
		}
		if err := tx.Model(&TopUp{}).
			Select("COUNT(DISTINCT user_id)").
			Where("user_id IN (?)", paidTopUpUsers).
			Scan(&result.TopUpQualifiedUsers).Error; err != nil {
			return err
		}
		if err := commonUserQuery(tx).
			Where("used_quota >= ?", usedQuotaThreshold).
			Count(&result.UsageQualifiedUsers).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserSubscription{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&result.SubscriptionUsers).Error; err != nil {
			return err
		}
		if err := tx.Model(&QuotaBucket{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&result.QuotaBucketUsers).Error; err != nil {
			return err
		}
		if err := tx.Model(&Redemption{}).
			Select("COUNT(DISTINCT used_user_id)").
			Where("status = ?", common.RedemptionCodeStatusUsed).
			Where("redemption_type = ?", RedemptionTypeQuota).
			Where("used_user_id > 0").
			Where("quota >= ?", redemptionQuotaThreshold(options.AmountThreshold)).
			Scan(&result.RedemptionUsers).Error; err != nil {
			return err
		}

		err := commonUserQuery(tx).
			Where("used_quota >= ? OR id IN (?) OR id IN (?) OR id IN (?) OR id IN (?)", usedQuotaThreshold, paidTopUpUsers, redemptionUsers, subscriptionUsers, bucketUsers).
			Pluck("id", &paidIDs).Error
		if err != nil {
			return err
		}
		if len(paidIDs) > 0 {
			err = commonUserQuery(tx).
				Where("id NOT IN ?", paidIDs).
				Pluck("id", &freeIDs).Error
		} else {
			err = commonUserQuery(tx).Pluck("id", &freeIDs).Error
		}
		if err != nil {
			return err
		}

		paidChangedIDs, err = classifyChangedUsers(tx, paidIDs, options.PaidGroup)
		if err != nil {
			return err
		}
		freeChangedIDs, err = classifyChangedUsers(tx, freeIDs, options.FreeGroup)
		if err != nil {
			return err
		}

		result.PaidUsers = int64(len(paidIDs))
		result.FreeUsers = int64(len(freeIDs))

		result.PaidUsersUpdated, err = updateUsersGroupByIds(tx, paidChangedIDs, options.PaidGroup)
		if err != nil {
			return err
		}
		result.FreeUsersUpdated, err = updateUsersGroupByIds(tx, freeChangedIDs, options.FreeGroup)
		if err != nil {
			return err
		}
		result.UpdatedUsers = result.PaidUsersUpdated + result.FreeUsersUpdated
		return nil
	})
	if err != nil {
		return nil, err
	}

	invalidateClassifiedUserCaches(paidChangedIDs)
	invalidateClassifiedUserCaches(freeChangedIDs)
	return result, nil
}
