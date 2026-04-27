package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SubscriptionUsageDaily stores per-day net subscription usage for operations reporting.
type SubscriptionUsageDaily struct {
	Id                 int   `json:"id"`
	UserId             int   `json:"user_id" gorm:"index"`
	UserSubscriptionId int   `json:"user_subscription_id" gorm:"index;uniqueIndex:idx_subscription_usage_daily,priority:1"`
	PlanId             int   `json:"plan_id" gorm:"index"`
	DayStart           int64 `json:"day_start" gorm:"type:bigint;index;uniqueIndex:idx_subscription_usage_daily,priority:2"`
	Quota              int64 `json:"quota" gorm:"type:bigint;not null;default:0"`
	RequestCount       int   `json:"request_count" gorm:"type:int;not null;default:0"`
	CreatedAt          int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64 `json:"updated_at" gorm:"bigint"`
}

var ensureSubscriptionUsageDailyTableMu sync.Mutex
var ensureSubscriptionUsageDailyTableDone bool

func EnsureSubscriptionUsageDailyTable() error {
	if ensureSubscriptionUsageDailyTableDone {
		return nil
	}
	ensureSubscriptionUsageDailyTableMu.Lock()
	defer ensureSubscriptionUsageDailyTableMu.Unlock()
	if ensureSubscriptionUsageDailyTableDone {
		return nil
	}
	if DB == nil {
		return errors.New("database is not initialized")
	}
	if err := DB.AutoMigrate(&SubscriptionUsageDaily{}); err != nil {
		return err
	}
	ensureSubscriptionUsageDailyTableDone = true
	return nil
}

func (u *SubscriptionUsageDaily) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	u.CreatedAt = now
	u.UpdatedAt = now
	return nil
}

func (u *SubscriptionUsageDaily) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = common.GetTimestamp()
	return nil
}

func SubscriptionUsageDayStart(timestamp int64) int64 {
	if timestamp <= 0 {
		timestamp = common.GetTimestamp()
	}
	t := time.Unix(timestamp, 0)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}

func applySubscriptionUsageDeltaTx(tx *gorm.DB, sub *UserSubscription, delta int64, requestDelta int, timestamp int64) error {
	if tx == nil || sub == nil {
		return errors.New("invalid subscription usage delta args")
	}
	if delta == 0 && requestDelta == 0 {
		return nil
	}
	if timestamp <= 0 {
		timestamp = common.GetTimestamp()
	}
	dayStart := SubscriptionUsageDayStart(timestamp)
	row := &SubscriptionUsageDaily{
		UserId:             sub.UserId,
		UserSubscriptionId: sub.Id,
		PlanId:             sub.PlanId,
		DayStart:           dayStart,
		Quota:              delta,
		RequestCount:       requestDelta,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_subscription_id"},
			{Name: "day_start"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quota":         gorm.Expr(subscriptionUsageDailyColumn("quota")+" + ?", delta),
			"request_count": gorm.Expr(subscriptionUsageDailyColumn("request_count")+" + ?", requestDelta),
			"updated_at":    timestamp,
		}),
	}).Create(row).Error
}

func subscriptionUsageDailyColumn(name string) string {
	if common.UsingPostgreSQL {
		return `"subscription_usage_dailies"."` + name + `"`
	}
	return "`subscription_usage_dailies`.`" + name + "`"
}

type AdminUserSubscriptionSummaryQuery struct {
	Page            int
	PageSize        int
	Keyword         string
	Group           string
	PlanId          int
	Status          string
	ExpireDays      int
	MinTodayUsed    int64
	MinUsagePercent float64
	Sort            string
	Order           string
}

type AdminUserSubscriptionSummary struct {
	ActiveCount           int     `json:"active_count"`
	PrimarySubscriptionId int     `json:"primary_subscription_id"`
	PrimaryPlanId         int     `json:"primary_plan_id"`
	PrimaryPlanTitle      string  `json:"primary_plan_title"`
	RemainingDays         int     `json:"remaining_days"`
	EndTime               int64   `json:"end_time"`
	NextResetTime         int64   `json:"next_reset_time"`
	TodayUsed             int64   `json:"today_used"`
	PeriodUsed            int64   `json:"period_used"`
	PeriodTotal           int64   `json:"period_total"`
	PeriodRemain          int64   `json:"period_remain"`
	UsagePercent          float64 `json:"usage_percent"`
	PeriodElapsedUsed     int64   `json:"period_elapsed_used"`
	PeriodElapsedQuota    int64   `json:"period_elapsed_quota"`
	ActualUsagePercent    float64 `json:"actual_usage_percent"`
	LifetimeUsed          int64   `json:"lifetime_used"`
}

type AdminUserSubscriptionSummaryItem struct {
	Id                  int                          `json:"id"`
	Username            string                       `json:"username"`
	DisplayName         string                       `json:"display_name"`
	Email               string                       `json:"email"`
	Group               string                       `json:"group"`
	Status              int                          `json:"status"`
	Role                int                          `json:"role"`
	Remark              string                       `json:"remark,omitempty"`
	DeletedAt           gorm.DeletedAt               `json:"DeletedAt"`
	SubscriptionSummary AdminUserSubscriptionSummary `json:"subscription_summary"`
}

func normalizeAdminUserSubscriptionSummaryQuery(query AdminUserSubscriptionSummaryQuery) AdminUserSubscriptionSummaryQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = common.ItemsPerPage
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Group = strings.TrimSpace(query.Group)
	query.Status = strings.TrimSpace(query.Status)
	switch query.Status {
	case "active", "none", "expired", "expiring":
	default:
		query.Status = "all"
	}
	if query.ExpireDays <= 0 {
		query.ExpireDays = 7
	}
	query.Sort = strings.TrimSpace(query.Sort)
	switch query.Sort {
	case "remaining_days", "end_time", "today_used", "lifetime_used", "usage_percent", "actual_usage_percent", "period_used":
	default:
		query.Sort = "id"
	}
	query.Order = strings.ToLower(strings.TrimSpace(query.Order))
	if query.Order != "asc" {
		query.Order = "desc"
	}
	return query
}

func ListAdminUserSubscriptionSummaries(query AdminUserSubscriptionSummaryQuery) ([]AdminUserSubscriptionSummaryItem, int64, error) {
	query = normalizeAdminUserSubscriptionSummaryQuery(query)
	if err := EnsureSubscriptionUsageDailyTable(); err != nil {
		return nil, 0, err
	}

	userQuery := DB.Unscoped().Model(&User{})
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		if keywordId, err := strconv.Atoi(query.Keyword); err == nil {
			userQuery = userQuery.Where(
				"id = ? OR username LIKE ? OR email LIKE ? OR display_name LIKE ?",
				keywordId, like, like, like,
			)
		} else {
			userQuery = userQuery.Where(
				"username LIKE ? OR email LIKE ? OR display_name LIKE ?",
				like, like, like,
			)
		}
	}
	if query.Group != "" {
		userQuery = userQuery.Where(subscriptionUserGroupColumn()+" = ?", query.Group)
	}

	var users []User
	if err := userQuery.
		Select("id, username, display_name, email, " + subscriptionUserGroupColumn() + ", status, role, remark, deleted_at").
		Order("id desc").
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return []AdminUserSubscriptionSummaryItem{}, 0, nil
	}

	userIds := make([]int, 0, len(users))
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}

	var subs []UserSubscription
	if err := DB.Where("user_id IN ?", userIds).
		Order("end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return nil, 0, err
	}

	planIds := make([]int, 0)
	seenPlans := map[int]struct{}{}
	subsByUser := make(map[int][]UserSubscription)
	activeSubsByUser := make(map[int][]UserSubscription)
	now := common.GetTimestamp()
	for _, sub := range subs {
		subsByUser[sub.UserId] = append(subsByUser[sub.UserId], sub)
		if sub.PlanId > 0 {
			if _, ok := seenPlans[sub.PlanId]; !ok {
				seenPlans[sub.PlanId] = struct{}{}
				planIds = append(planIds, sub.PlanId)
			}
		}
		if sub.Status == "active" && sub.EndTime > now {
			activeSubsByUser[sub.UserId] = append(activeSubsByUser[sub.UserId], sub)
		}
	}

	planTitleById := make(map[int]string, len(planIds))
	if len(planIds) > 0 {
		var plans []SubscriptionPlan
		if err := DB.Select("id, title").Where("id IN ?", planIds).Find(&plans).Error; err != nil {
			return nil, 0, err
		}
		for _, plan := range plans {
			planTitleById[plan.Id] = plan.Title
		}
	}

	todayUsageByUser := make(map[int]int64)
	dayStart := SubscriptionUsageDayStart(now)
	var todayRows []struct {
		UserId int   `gorm:"column:user_id"`
		Quota  int64 `gorm:"column:quota"`
	}
	if err := DB.Model(&SubscriptionUsageDaily{}).
		Select("user_id, sum(quota) as quota").
		Where("user_id IN ? AND day_start = ?", userIds, dayStart).
		Group("user_id").
		Find(&todayRows).Error; err != nil {
		return nil, 0, err
	}
	for _, row := range todayRows {
		todayUsageByUser[row.UserId] = row.Quota
	}

	primaryActiveSubByUser := make(map[int]UserSubscription)
	for userId, activeSubs := range activeSubsByUser {
		if primary, ok := primaryActiveSubscription(activeSubs); ok {
			primaryActiveSubByUser[userId] = primary
		}
	}
	actualUsageStatsBySub, err := loadSubscriptionActualUsageStats(primaryActiveSubByUser, now)
	if err != nil {
		return nil, 0, err
	}

	items := make([]AdminUserSubscriptionSummaryItem, 0, len(users))
	expireBefore := now + int64(query.ExpireDays)*86400
	for _, user := range users {
		allSubs := subsByUser[user.Id]
		activeSubs := activeSubsByUser[user.Id]
		hasPlan := query.PlanId <= 0
		if query.PlanId > 0 {
			for _, sub := range allSubs {
				if sub.PlanId == query.PlanId {
					hasPlan = true
					break
				}
			}
		}
		if !hasPlan {
			continue
		}

		summary := AdminUserSubscriptionSummary{
			ActiveCount:  len(activeSubs),
			TodayUsed:    todayUsageByUser[user.Id],
			LifetimeUsed: sumSubscriptionLifetimeUsage(allSubs),
		}
		if len(activeSubs) > 0 {
			primary := primaryActiveSubByUser[user.Id]
			summary.PrimarySubscriptionId = primary.Id
			summary.PrimaryPlanId = primary.PlanId
			summary.PrimaryPlanTitle = planTitleById[primary.PlanId]
			if summary.PrimaryPlanTitle == "" && primary.PlanId > 0 {
				summary.PrimaryPlanTitle = fmt.Sprintf("#%d", primary.PlanId)
			}
			summary.EndTime = primary.EndTime
			summary.NextResetTime = primary.NextResetTime
			summary.RemainingDays = remainingDays(now, primary.EndTime)
			summary.PeriodUsed = primary.AmountUsed
			summary.PeriodTotal = primary.AmountTotal
			if primary.AmountTotal > 0 {
				summary.PeriodRemain = primary.AmountTotal - primary.AmountUsed
				if summary.PeriodRemain < 0 {
					summary.PeriodRemain = 0
				}
				summary.UsagePercent = math.Round(float64(primary.AmountUsed)/float64(primary.AmountTotal)*10000) / 100
				if actualStats, ok := actualUsageStatsBySub[primary.Id]; ok {
					summary.PeriodElapsedUsed = actualStats.Used
					summary.PeriodElapsedQuota = actualStats.TheoreticalQuota
					summary.ActualUsagePercent = actualStats.ActualUsagePercent
				}
			}
		}

		if !summaryMatchesSubscriptionStatus(summary, len(allSubs) > 0, query.Status, expireBefore) {
			continue
		}
		if query.MinTodayUsed > 0 && summary.TodayUsed < query.MinTodayUsed {
			continue
		}
		if query.MinUsagePercent > 0 && summary.UsagePercent < query.MinUsagePercent {
			continue
		}

		items = append(items, AdminUserSubscriptionSummaryItem{
			Id:                  user.Id,
			Username:            user.Username,
			DisplayName:         user.DisplayName,
			Email:               user.Email,
			Group:               user.Group,
			Status:              user.Status,
			Role:                user.Role,
			Remark:              user.Remark,
			DeletedAt:           user.DeletedAt,
			SubscriptionSummary: summary,
		})
	}

	sortAdminUserSubscriptionSummaryItems(items, query.Sort, query.Order)
	total := int64(len(items))
	start := (query.Page - 1) * query.PageSize
	if start >= len(items) {
		return []AdminUserSubscriptionSummaryItem{}, total, nil
	}
	end := start + query.PageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

func subscriptionUserGroupColumn() string {
	if commonGroupCol != "" {
		return commonGroupCol
	}
	if common.UsingPostgreSQL {
		return `"group"`
	}
	return "`group`"
}

func remainingDays(now int64, endTime int64) int {
	if endTime <= now {
		return 0
	}
	return int(math.Ceil(float64(endTime-now) / 86400.0))
}

func sumSubscriptionLifetimeUsage(subs []UserSubscription) int64 {
	total := int64(0)
	for _, sub := range subs {
		total += sub.AmountUsedTotal
	}
	if total < 0 {
		return 0
	}
	return total
}

func primaryActiveSubscription(activeSubs []UserSubscription) (UserSubscription, bool) {
	if len(activeSubs) == 0 {
		return UserSubscription{}, false
	}
	sort.SliceStable(activeSubs, func(i, j int) bool {
		if activeSubs[i].EndTime == activeSubs[j].EndTime {
			return activeSubs[i].Id < activeSubs[j].Id
		}
		return activeSubs[i].EndTime < activeSubs[j].EndTime
	})
	return activeSubs[0], true
}

type subscriptionActualUsageStats struct {
	Used               int64
	TheoreticalQuota   int64
	ActualUsagePercent float64
}

type subscriptionActualUsageWindow struct {
	StartDay         int64
	EndDay           int64
	TheoreticalQuota int64
}

func loadSubscriptionActualUsageStats(primarySubs map[int]UserSubscription, now int64) (map[int]subscriptionActualUsageStats, error) {
	stats := make(map[int]subscriptionActualUsageStats, len(primarySubs))
	if len(primarySubs) == 0 {
		return stats, nil
	}

	windows := make(map[int]subscriptionActualUsageWindow, len(primarySubs))
	subIds := make([]int, 0, len(primarySubs))
	minDay := int64(0)
	maxDay := int64(0)
	for _, sub := range primarySubs {
		window := subscriptionActualUsageWindowForSub(sub, now)
		if window.TheoreticalQuota <= 0 {
			stats[sub.Id] = subscriptionActualUsageStats{}
			continue
		}
		windows[sub.Id] = window
		stats[sub.Id] = subscriptionActualUsageStats{
			TheoreticalQuota: window.TheoreticalQuota,
		}
		subIds = append(subIds, sub.Id)
		if minDay == 0 || window.StartDay < minDay {
			minDay = window.StartDay
		}
		if window.EndDay > maxDay {
			maxDay = window.EndDay
		}
	}
	if len(subIds) == 0 {
		return stats, nil
	}

	var rows []struct {
		UserSubscriptionId int   `gorm:"column:user_subscription_id"`
		DayStart           int64 `gorm:"column:day_start"`
		Quota              int64 `gorm:"column:quota"`
	}
	if err := DB.Model(&SubscriptionUsageDaily{}).
		Select("user_subscription_id, day_start, quota").
		Where("user_subscription_id IN ? AND day_start >= ? AND day_start <= ?", subIds, minDay, maxDay).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		window, ok := windows[row.UserSubscriptionId]
		if !ok || row.DayStart < window.StartDay || row.DayStart > window.EndDay {
			continue
		}
		stat := stats[row.UserSubscriptionId]
		stat.Used += row.Quota
		stats[row.UserSubscriptionId] = stat
	}
	for subId, stat := range stats {
		if stat.TheoreticalQuota <= 0 {
			continue
		}
		stat.ActualUsagePercent = math.Round(float64(stat.Used)/float64(stat.TheoreticalQuota)*10000) / 100
		stats[subId] = stat
	}
	return stats, nil
}

func subscriptionActualUsageWindowForSub(sub UserSubscription, now int64) subscriptionActualUsageWindow {
	if sub.AmountTotal <= 0 {
		return subscriptionActualUsageWindow{}
	}
	start := sub.LastResetTime
	if start <= 0 || (sub.StartTime > 0 && start < sub.StartTime) {
		start = sub.StartTime
	}
	end := sub.NextResetTime
	if end <= 0 || (sub.EndTime > 0 && end > sub.EndTime) {
		end = sub.EndTime
	}
	if start <= 0 || end <= start || now <= start {
		return subscriptionActualUsageWindow{}
	}
	period := end - start
	totalDays := int64(math.Ceil(float64(period) / 86400.0))
	if totalDays <= 0 {
		return subscriptionActualUsageWindow{}
	}
	startDay := SubscriptionUsageDayStart(start)
	endAt := now
	if endAt >= end {
		endAt = end - 1
	}
	if endAt < start {
		return subscriptionActualUsageWindow{}
	}
	endDay := SubscriptionUsageDayStart(endAt)
	elapsedDays := (endDay-startDay)/86400 + 1
	if elapsedDays <= 0 {
		return subscriptionActualUsageWindow{}
	}
	if elapsedDays > totalDays {
		elapsedDays = totalDays
	}
	expected := float64(sub.AmountTotal) * float64(elapsedDays) / float64(totalDays)
	if expected <= 0 {
		return subscriptionActualUsageWindow{}
	}
	theoreticalQuota := int64(math.Round(expected))
	if theoreticalQuota > sub.AmountTotal {
		theoreticalQuota = sub.AmountTotal
	}
	return subscriptionActualUsageWindow{
		StartDay:         startDay,
		EndDay:           endDay,
		TheoreticalQuota: theoreticalQuota,
	}
}

func summaryMatchesSubscriptionStatus(summary AdminUserSubscriptionSummary, hasAnySubscription bool, status string, expireBefore int64) bool {
	hasActive := summary.ActiveCount > 0
	switch status {
	case "active":
		return hasActive
	case "none":
		return !hasActive
	case "expired":
		return !hasActive && hasAnySubscription
	case "expiring":
		return hasActive && summary.EndTime > 0 && summary.EndTime <= expireBefore
	default:
		return true
	}
}

func sortAdminUserSubscriptionSummaryItems(items []AdminUserSubscriptionSummaryItem, sortKey string, order string) {
	desc := order != "asc"
	sort.SliceStable(items, func(i, j int) bool {
		left := adminUserSubscriptionSortValue(items[i], sortKey)
		right := adminUserSubscriptionSortValue(items[j], sortKey)
		if left == right {
			if desc {
				return items[i].Id > items[j].Id
			}
			return items[i].Id < items[j].Id
		}
		if desc {
			return left > right
		}
		return left < right
	})
}

func adminUserSubscriptionSortValue(item AdminUserSubscriptionSummaryItem, sortKey string) float64 {
	summary := item.SubscriptionSummary
	switch sortKey {
	case "remaining_days":
		if summary.ActiveCount == 0 {
			return -1
		}
		return float64(summary.RemainingDays)
	case "end_time":
		return float64(summary.EndTime)
	case "today_used":
		return float64(summary.TodayUsed)
	case "lifetime_used":
		return float64(summary.LifetimeUsed)
	case "usage_percent":
		return summary.UsagePercent
	case "actual_usage_percent":
		return summary.ActualUsagePercent
	case "period_used":
		return float64(summary.PeriodUsed)
	default:
		return float64(item.Id)
	}
}

type SubscriptionUsageBackfillStats struct {
	ScannedLogs                int64 `json:"scanned_logs"`
	AppliedLogs                int64 `json:"applied_logs"`
	SkippedMissingFields       int64 `json:"skipped_missing_fields"`
	SkippedInvalidJSON         int64 `json:"skipped_invalid_json"`
	SkippedMissingSubscription int64 `json:"skipped_missing_subscription"`
	AggregatedRows             int64 `json:"aggregated_rows"`
	TotalQuota                 int64 `json:"total_quota"`
}

type subscriptionUsageBackfillAggregate struct {
	UserSubscriptionId int
	DayStart           int64
	Quota              int64
	RequestCount       int
}

func BackfillSubscriptionUsageFromLogs(startTimestamp int64, endTimestamp int64, batchSize int) (*SubscriptionUsageBackfillStats, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	stats := &SubscriptionUsageBackfillStats{}
	aggregates := map[string]*subscriptionUsageBackfillAggregate{}
	subscriptionIds := map[int]struct{}{}
	lastId := 0

	for {
		var logs []Log
		tx := LOG_DB.Model(&Log{}).
			Select("id, user_id, created_at, type, quota, other").
			Where("id > ? AND type IN ?", lastId, []int{LogTypeConsume, LogTypeRefund}).
			Order("id asc").
			Limit(batchSize)
		if startTimestamp > 0 {
			tx = tx.Where("created_at >= ?", startTimestamp)
		}
		if endTimestamp > 0 {
			tx = tx.Where("created_at <= ?", endTimestamp)
		}
		if err := tx.Find(&logs).Error; err != nil {
			return nil, err
		}
		if len(logs) == 0 {
			break
		}
		for _, log := range logs {
			lastId = log.Id
			stats.ScannedLogs++
			subscriptionId, quota, ok, invalidJSON := parseSubscriptionUsageFromLog(log)
			if invalidJSON {
				stats.SkippedInvalidJSON++
				continue
			}
			if !ok {
				stats.SkippedMissingFields++
				continue
			}
			key := fmt.Sprintf("%d:%d", subscriptionId, SubscriptionUsageDayStart(log.CreatedAt))
			item, exists := aggregates[key]
			if !exists {
				item = &subscriptionUsageBackfillAggregate{
					UserSubscriptionId: subscriptionId,
					DayStart:           SubscriptionUsageDayStart(log.CreatedAt),
				}
				aggregates[key] = item
			}
			item.Quota += quota
			if log.Type == LogTypeConsume && quota > 0 {
				item.RequestCount++
			}
			subscriptionIds[subscriptionId] = struct{}{}
			stats.AppliedLogs++
			stats.TotalQuota += quota
		}
	}

	if len(aggregates) == 0 {
		return stats, nil
	}

	subIds := make([]int, 0, len(subscriptionIds))
	for id := range subscriptionIds {
		subIds = append(subIds, id)
	}
	var subs []UserSubscription
	if err := DB.Where("id IN ?", subIds).Find(&subs).Error; err != nil {
		return nil, err
	}
	subById := make(map[int]UserSubscription, len(subs))
	for _, sub := range subs {
		subById[sub.Id] = sub
	}

	rows := make([]SubscriptionUsageDaily, 0, len(aggregates))
	minDay := int64(math.MaxInt64)
	maxDay := int64(0)
	for _, aggregate := range aggregates {
		sub, ok := subById[aggregate.UserSubscriptionId]
		if !ok {
			stats.SkippedMissingSubscription++
			continue
		}
		if aggregate.DayStart < minDay {
			minDay = aggregate.DayStart
		}
		if aggregate.DayStart > maxDay {
			maxDay = aggregate.DayStart
		}
		rows = append(rows, SubscriptionUsageDaily{
			UserId:             sub.UserId,
			UserSubscriptionId: sub.Id,
			PlanId:             sub.PlanId,
			DayStart:           aggregate.DayStart,
			Quota:              aggregate.Quota,
			RequestCount:       aggregate.RequestCount,
		})
	}
	stats.AggregatedRows = int64(len(rows))
	if len(rows) == 0 {
		return stats, nil
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_subscription_id IN ? AND day_start >= ? AND day_start <= ?", subIds, minDay, maxDay).
			Delete(&SubscriptionUsageDaily{}).Error; err != nil {
			return err
		}
		if err := tx.CreateInBatches(rows, 500).Error; err != nil {
			return err
		}
		return refreshSubscriptionAmountUsedTotalsTx(tx, subIds)
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func parseSubscriptionUsageFromLog(log Log) (subscriptionId int, quota int64, ok bool, invalidJSON bool) {
	if strings.TrimSpace(log.Other) == "" {
		return 0, 0, false, false
	}
	var other map[string]interface{}
	if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
		return 0, 0, false, true
	}
	if fmt.Sprintf("%v", other["billing_source"]) != "subscription" {
		return 0, 0, false, false
	}
	subscriptionId = int(numberFromMap(other, "subscription_id"))
	if subscriptionId <= 0 {
		return 0, 0, false, false
	}
	rawQuota, hasQuota := numberFromMapOk(other, "subscription_consumed")
	if !hasQuota {
		return 0, 0, false, false
	}
	quota = rawQuota
	if log.Type == LogTypeRefund && quota > 0 {
		quota = -quota
	}
	return subscriptionId, quota, true, false
}

func numberFromMap(m map[string]interface{}, key string) int64 {
	value, _ := numberFromMapOk(m, key)
	return value
}

func numberFromMapOk(m map[string]interface{}, key string) (int64, bool) {
	value, ok := m[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func refreshSubscriptionAmountUsedTotalsTx(tx *gorm.DB, subscriptionIds []int) error {
	if len(subscriptionIds) == 0 {
		return nil
	}
	if err := tx.Model(&UserSubscription{}).Where("id IN ?", subscriptionIds).Update("amount_used_total", 0).Error; err != nil {
		return err
	}
	var sums []struct {
		UserSubscriptionId int   `gorm:"column:user_subscription_id"`
		Quota              int64 `gorm:"column:quota"`
	}
	if err := tx.Model(&SubscriptionUsageDaily{}).
		Select("user_subscription_id, sum(quota) as quota").
		Where("user_subscription_id IN ?", subscriptionIds).
		Group("user_subscription_id").
		Find(&sums).Error; err != nil {
		return err
	}
	for _, sum := range sums {
		quota := sum.Quota
		if quota < 0 {
			quota = 0
		}
		if err := tx.Model(&UserSubscription{}).Where("id = ?", sum.UserSubscriptionId).
			Update("amount_used_total", quota).Error; err != nil {
			return err
		}
	}
	return nil
}
