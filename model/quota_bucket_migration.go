package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func migrateLegacyWeeklyQuotaPlansToBuckets() error {
	var plans []SubscriptionPlan
	if err := DB.Where("duration_unit = ? AND duration_value = ? AND quota_reset_period = ?", SubscriptionDurationDay, 7, SubscriptionResetNever).
		Find(&plans).Error; err != nil {
		return err
	}
	if len(plans) == 0 {
		return nil
	}
	plansById := make(map[int]SubscriptionPlan, len(plans))
	planIds := make([]int, 0, len(plans))
	for _, plan := range plans {
		if !isLegacyWeeklyQuotaPlan(plan) {
			continue
		}
		plansById[plan.Id] = plan
		planIds = append(planIds, plan.Id)
	}
	if len(planIds) == 0 {
		return nil
	}
	if err := migrateLegacyWeeklyQuotaRedemptions(planIds, plansById); err != nil {
		return err
	}
	return migrateLegacyWeeklyQuotaSubscriptions(planIds, plansById)
}

func isLegacyWeeklyQuotaPlan(plan SubscriptionPlan) bool {
	title := strings.TrimSpace(plan.Title + " " + plan.Subtitle)
	if plan.DurationUnit != SubscriptionDurationDay || plan.DurationValue != 7 {
		return false
	}
	if NormalizeResetPeriod(plan.QuotaResetPeriod) != SubscriptionResetNever {
		return false
	}
	if plan.TotalAmount <= 0 {
		return false
	}
	if plan.SortOrder >= 0 {
		return false
	}
	return strings.Contains(title, "API额度") || strings.Contains(title, "7天")
}

func migrateLegacyWeeklyQuotaRedemptions(planIds []int, plansById map[int]SubscriptionPlan) error {
	var redemptions []Redemption
	if err := DB.Where("redemption_type = ? AND plan_id IN ?", RedemptionTypePlan, planIds).Find(&redemptions).Error; err != nil {
		return err
	}
	for _, redemption := range redemptions {
		plan, ok := plansById[redemption.PlanId]
		if !ok {
			continue
		}
		updates := map[string]interface{}{
			"redemption_type":         RedemptionTypeBucket,
			"quota":                   int(plan.TotalAmount),
			"bucket_duration_seconds": DefaultQuotaBucketDurationSeconds,
		}
		if strings.TrimSpace(redemption.Name) == "" {
			updates["name"] = plan.Title
		}
		if err := DB.Model(&Redemption{}).Where("id = ?", redemption.Id).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateLegacyWeeklyQuotaSubscriptions(planIds []int, plansById map[int]SubscriptionPlan) error {
	var subs []UserSubscription
	if err := DB.Where("plan_id IN ? AND status <> ?", planIds, QuotaBucketStatusMigrated).Find(&subs).Error; err != nil {
		return err
	}
	now := common.GetTimestamp()
	for _, sub := range subs {
		plan, ok := plansById[sub.PlanId]
		if !ok {
			continue
		}
		if sub.Status == "active" && sub.EndTime > now {
			var count int64
			query := DB.Model(&QuotaBucket{}).
				Where("user_id = ? AND source_plan_id = ? AND start_time = ? AND end_time = ?", sub.UserId, sub.PlanId, sub.StartTime, sub.EndTime)
			if sub.SourceRedemptionId > 0 {
				query = query.Where("source_redemption_id = ?", sub.SourceRedemptionId)
			}
			if err := query.Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				bucket := &QuotaBucket{
					UserId:             sub.UserId,
					Title:              plan.Title,
					AmountTotal:        sub.AmountTotal,
					AmountUsed:         sub.AmountUsed,
					AmountUsedTotal:    sub.AmountUsedTotal,
					StartTime:          sub.StartTime,
					EndTime:            sub.EndTime,
					Status:             QuotaBucketStatusActive,
					Source:             QuotaBucketSourceMigration,
					SourceRedemptionId: sub.SourceRedemptionId,
					SourcePlanId:       sub.PlanId,
					CreatedAt:          sub.CreatedAt,
					UpdatedAt:          common.GetTimestamp(),
				}
				if bucket.AmountUsed >= bucket.AmountTotal {
					bucket.Status = QuotaBucketStatusEmpty
				}
				if err := DB.Create(bucket).Error; err != nil {
					return err
				}
			}
		}
		if err := DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Update("status", QuotaBucketStatusMigrated).Error; err != nil {
			return err
		}
	}
	return nil
}
