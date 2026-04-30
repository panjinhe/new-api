package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RedemptionTypeQuota = "quota"
	RedemptionTypePlan  = "plan"

	oneTimeWelfareRedemptionAmountUSD = 20.0
)

type Redemption struct {
	Id             int               `json:"id"`
	UserId         int               `json:"user_id"`
	Key            string            `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status         int               `json:"status" gorm:"default:1"`
	Name           string            `json:"name" gorm:"index"`
	Quota          int               `json:"quota" gorm:"default:100"`
	RedemptionType string            `json:"redemption_type" gorm:"type:varchar(16);default:'quota';index"`
	PlanId         int               `json:"plan_id" gorm:"default:0;index"`
	Plan           *SubscriptionPlan `json:"plan,omitempty" gorm:"foreignKey:PlanId;references:Id;constraint:-"`
	BatchId        string            `json:"batch_id" gorm:"type:varchar(64);default:'';index"`
	Source         string            `json:"source" gorm:"type:varchar(64);default:'manual';index"`
	CreatedTime    int64             `json:"created_time" gorm:"bigint"`
	RedeemedTime   int64             `json:"redeemed_time" gorm:"bigint"`
	Count          int               `json:"count" gorm:"-:all"` // only for api request
	UsedUserId     int               `json:"used_user_id"`
	DeletedAt      gorm.DeletedAt    `gorm:"index"`
	ExpiredTime    int64             `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
}

type RedemptionSubscriptionResult struct {
	Id            int    `json:"id"`
	PlanId        int    `json:"plan_id"`
	PlanTitle     string `json:"plan_title"`
	AmountTotal   int64  `json:"amount_total"`
	AmountUsed    int64  `json:"amount_used"`
	DailyQuota    int64  `json:"daily_quota"`
	TotalQuota    int64  `json:"total_quota"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	Extended      bool   `json:"extended"`
	ResetPeriod   string `json:"reset_period"`
	NextResetTime int64  `json:"next_reset_time"`
	RedemptionId  int    `json:"redemption_id"`
	UpgradeGroup  string `json:"upgrade_group,omitempty"`
	Source        string `json:"source"`
}

type RedemptionResult struct {
	Type         string                        `json:"type"`
	Quota        int                           `json:"quota,omitempty"`
	RedemptionId int                           `json:"redemption_id"`
	Subscription *RedemptionSubscriptionResult `json:"subscription,omitempty"`
}

func NormalizeRedemptionType(redemptionType string) string {
	switch strings.TrimSpace(redemptionType) {
	case RedemptionTypePlan:
		return RedemptionTypePlan
	default:
		return RedemptionTypeQuota
	}
}

func (redemption *Redemption) Normalize() {
	redemption.RedemptionType = NormalizeRedemptionType(redemption.RedemptionType)
	if strings.TrimSpace(redemption.Source) == "" {
		redemption.Source = "manual"
	}
	if redemption.RedemptionType == RedemptionTypeQuota {
		redemption.PlanId = 0
	}
}

func withUpdateLock(tx *gorm.DB) *gorm.DB {
	if tx == nil || common.UsingSQLite {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

func oneTimeWelfareRedemptionQuota() int {
	return int(math.Round(oneTimeWelfareRedemptionAmountUSD * common.QuotaPerUnit))
}

func isOneTimeWelfareRedemption(redemption *Redemption) bool {
	if redemption == nil {
		return false
	}
	return redemption.RedemptionType == RedemptionTypeQuota && redemption.Quota == oneTimeWelfareRedemptionQuota()
}

func lockUserForRedemptionTx(tx *gorm.DB, userId int) error {
	var user User
	return withUpdateLock(tx).Select("id").Where("id = ?", userId).First(&user).Error
}

func userHasRedeemedOneTimeWelfareTx(tx *gorm.DB, userId int) (bool, error) {
	var count int64
	err := tx.Unscoped().Model(&Redemption{}).
		Where("used_user_id = ?", userId).
		Where("status = ?", common.RedemptionCodeStatusUsed).
		Where("redemption_type = ?", RedemptionTypeQuota).
		Where("quota = ?", oneTimeWelfareRedemptionQuota()).
		Count(&count).Error
	return count > 0, err
}

func ensureUserCanRedeemOneTimeWelfareTx(tx *gorm.DB, userId int) error {
	if err := lockUserForRedemptionTx(tx, userId); err != nil {
		return err
	}
	redeemed, err := userHasRedeemedOneTimeWelfareTx(tx, userId)
	if err != nil {
		return err
	}
	if redeemed {
		return ErrRedemptionWelfareAlreadyRedeemed
	}
	return nil
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Preload("Plan").Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build query based on keyword type
	query := tx.Model(&Redemption{})

	// Only try to convert to ID if the string represents a valid integer
	if id, err := strconv.Atoi(keyword); err == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Preload("Plan").Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.Preload("Plan").First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(key string, userId int) (*RedemptionResult, error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	redemption := &Redemption{}
	result := &RedemptionResult{}

	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err := DB.Transaction(func(tx *gorm.DB) error {
		err := withUpdateLock(tx).Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		redemption.Normalize()
		result.Type = redemption.RedemptionType
		result.RedemptionId = redemption.Id
		if redemption.RedemptionType == RedemptionTypePlan {
			subscription, plan, extended, err := redeemSubscriptionTx(tx, redemption, userId)
			if err != nil {
				return err
			}
			result.Subscription = buildRedemptionSubscriptionResult(subscription, plan, redemption.Id, extended)
		} else {
			if isOneTimeWelfareRedemption(redemption) {
				if err := ensureUserCanRedeemOneTimeWelfareTx(tx, userId); err != nil {
					return err
				}
			}
			err = tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
			if err != nil {
				return err
			}
			result.Quota = redemption.Quota
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		err = tx.Save(redemption).Error
		return err
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return nil, err
	}
	if redemption.RedemptionType == RedemptionTypePlan && result.Subscription != nil {
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码开通套餐 %s，兑换码ID %d", result.Subscription.PlanTitle, redemption.Id))
		invalidateClassifiedUserCaches([]int{userId})
	} else {
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
		promoteUserToPaidGroupIfRedemptionQualified(userId)
	}
	return result, nil
}

func redeemSubscriptionTx(tx *gorm.DB, redemption *Redemption, userId int) (*UserSubscription, *SubscriptionPlan, bool, error) {
	if redemption.PlanId <= 0 {
		return nil, nil, false, errors.New("套餐不存在或已下架")
	}
	plan, err := getSubscriptionPlanByIdTx(tx, redemption.PlanId)
	if err != nil || plan == nil || !plan.Enabled {
		return nil, nil, false, errors.New("套餐不存在或已下架")
	}
	subscription, extended, err := RedeemSubscriptionFromPlanTx(tx, userId, plan, redemption.Id)
	if err != nil {
		return nil, nil, false, err
	}
	return subscription, plan, extended, nil
}

func buildRedemptionSubscriptionResult(subscription *UserSubscription, plan *SubscriptionPlan, redemptionId int, extended bool) *RedemptionSubscriptionResult {
	if subscription == nil || plan == nil {
		return nil
	}
	return &RedemptionSubscriptionResult{
		Id:            subscription.Id,
		PlanId:        plan.Id,
		PlanTitle:     plan.Title,
		AmountTotal:   subscription.AmountTotal,
		AmountUsed:    subscription.AmountUsed,
		DailyQuota:    subscriptionDailyQuota(plan),
		TotalQuota:    subscriptionTotalQuota(subscription, plan),
		StartTime:     subscription.StartTime,
		EndTime:       subscription.EndTime,
		Extended:      extended,
		ResetPeriod:   NormalizeResetPeriod(plan.QuotaResetPeriod),
		NextResetTime: subscription.NextResetTime,
		RedemptionId:  redemptionId,
		UpgradeGroup:  subscription.UpgradeGroup,
		Source:        subscription.Source,
	}
}

func subscriptionDailyQuota(plan *SubscriptionPlan) int64 {
	if plan == nil {
		return 0
	}
	if NormalizeResetPeriod(plan.QuotaResetPeriod) == SubscriptionResetDaily {
		return plan.TotalAmount
	}
	return 0
}

func subscriptionTotalQuota(subscription *UserSubscription, plan *SubscriptionPlan) int64 {
	if subscription == nil || plan == nil {
		return 0
	}
	if NormalizeResetPeriod(plan.QuotaResetPeriod) != SubscriptionResetDaily {
		return subscription.AmountTotal
	}
	if subscription.StartTime <= 0 || subscription.EndTime <= subscription.StartTime {
		return plan.TotalAmount
	}
	days := int64((subscription.EndTime - subscription.StartTime + 86399) / 86400)
	if days <= 0 {
		days = 1
	}
	return plan.TotalAmount * days
}

func (redemption *Redemption) Insert() error {
	var err error
	redemption.Normalize()
	if redemption.Status == 0 {
		redemption.Status = common.RedemptionCodeStatusEnabled
	}
	err = DB.Select(
		"UserId",
		"Key",
		"Status",
		"Name",
		"Quota",
		"RedemptionType",
		"PlanId",
		"BatchId",
		"Source",
		"CreatedTime",
		"RedeemedTime",
		"UsedUserId",
		"ExpiredTime",
	).Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	redemption.Normalize()
	err = DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "expired_time", "redemption_type", "plan_id", "batch_id", "source").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
