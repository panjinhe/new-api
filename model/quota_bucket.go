package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	QuotaBucketStatusActive   = "active"
	QuotaBucketStatusExpired  = "expired"
	QuotaBucketStatusEmpty    = "empty"
	QuotaBucketStatusMigrated = "migrated"

	QuotaBucketSourceRedemption = "redemption"
	QuotaBucketSourceMigration  = "migration"
)

type QuotaBucket struct {
	Id                 int    `json:"id"`
	UserId             int    `json:"user_id" gorm:"index;index:idx_quota_bucket_active,priority:1"`
	Title              string `json:"title" gorm:"type:varchar(128);default:''"`
	AmountTotal        int64  `json:"amount_total" gorm:"type:bigint;not null;default:0"`
	AmountUsed         int64  `json:"amount_used" gorm:"type:bigint;not null;default:0"`
	AmountUsedTotal    int64  `json:"amount_used_total" gorm:"type:bigint;not null;default:0"`
	StartTime          int64  `json:"start_time" gorm:"type:bigint"`
	EndTime            int64  `json:"end_time" gorm:"type:bigint;index;index:idx_quota_bucket_active,priority:3"`
	Status             string `json:"status" gorm:"type:varchar(32);index;index:idx_quota_bucket_active,priority:2"`
	Source             string `json:"source" gorm:"type:varchar(32);default:'redemption';index"`
	SourceRedemptionId int    `json:"source_redemption_id" gorm:"default:0;index"`
	SourcePlanId       int    `json:"source_plan_id" gorm:"default:0;index"`
	CreatedAt          int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"type:bigint"`
}

func (b *QuotaBucket) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if b.CreatedAt == 0 {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	return nil
}

func (b *QuotaBucket) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = common.GetTimestamp()
	return nil
}

type QuotaBucketPreConsumeRecord struct {
	Id          int    `json:"id"`
	RequestId   string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId      int    `json:"user_id" gorm:"index"`
	PreConsumed int64  `json:"pre_consumed" gorm:"type:bigint;not null;default:0"`
	Status      string `json:"status" gorm:"type:varchar(32);index"`
	CreatedAt   int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"type:bigint;index"`
}

func (r *QuotaBucketPreConsumeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *QuotaBucketPreConsumeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

type QuotaBucketPreConsumeAllocation struct {
	Id        int    `json:"id"`
	RecordId  int    `json:"record_id" gorm:"index"`
	RequestId string `json:"request_id" gorm:"type:varchar(64);index"`
	BucketId  int    `json:"bucket_id" gorm:"index"`
	Amount    int64  `json:"amount" gorm:"type:bigint;not null;default:0"`
	CreatedAt int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"type:bigint"`
}

func (a *QuotaBucketPreConsumeAllocation) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

func (a *QuotaBucketPreConsumeAllocation) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = common.GetTimestamp()
	return nil
}

type QuotaBucketSummary struct {
	Bucket         QuotaBucket `json:"bucket"`
	RemainingQuota int64       `json:"remaining_quota"`
	Status         string      `json:"status"`
}

type QuotaBucketSelfSummary struct {
	Buckets           []QuotaBucketSummary `json:"buckets"`
	ActiveBuckets     []QuotaBucketSummary `json:"active_buckets"`
	TotalAmount       int64                `json:"total_amount"`
	TotalUsed         int64                `json:"total_used"`
	TotalRemaining    int64                `json:"total_remaining"`
	NearestEndTime    int64                `json:"nearest_end_time"`
	ActiveBucketCount int                  `json:"active_bucket_count"`
}

type QuotaBucketPreConsumeResult struct {
	RecordId        int
	PreConsumed     int64
	TotalRemaining  int64
	NearestEndTime  int64
	AllocationCount int
}

func HasActiveQuotaBucket(userId int) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var count int64
	err := DB.Model(&QuotaBucket{}).
		Where("user_id = ? AND status = ? AND end_time > ? AND amount_total > amount_used", userId, QuotaBucketStatusActive, now).
		Count(&count).Error
	return count > 0, err
}

func GetUserQuotaBucketSelf(userId int) (*QuotaBucketSelfSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var buckets []QuotaBucket
	if err := DB.Where("user_id = ?", userId).Order("end_time asc, id asc").Find(&buckets).Error; err != nil {
		return nil, err
	}
	result := &QuotaBucketSelfSummary{
		Buckets:       make([]QuotaBucketSummary, 0, len(buckets)),
		ActiveBuckets: make([]QuotaBucketSummary, 0, len(buckets)),
	}
	for _, bucket := range buckets {
		status := effectiveQuotaBucketStatus(bucket, now)
		remaining := quotaBucketRemaining(bucket)
		summary := QuotaBucketSummary{
			Bucket:         bucket,
			RemainingQuota: remaining,
			Status:         status,
		}
		result.Buckets = append(result.Buckets, summary)
		if status == QuotaBucketStatusActive {
			result.ActiveBuckets = append(result.ActiveBuckets, summary)
			result.ActiveBucketCount++
			result.TotalAmount += bucket.AmountTotal
			result.TotalUsed += bucket.AmountUsed
			result.TotalRemaining += remaining
			if result.NearestEndTime == 0 || bucket.EndTime < result.NearestEndTime {
				result.NearestEndTime = bucket.EndTime
			}
		}
	}
	return result, nil
}

func CreateQuotaBucketFromRedemptionTx(tx *gorm.DB, redemption *Redemption, userId int) (*QuotaBucket, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if redemption == nil {
		return nil, errors.New("redemption is nil")
	}
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	if redemption.Quota <= 0 {
		return nil, errors.New("额度必须大于0")
	}
	duration := redemption.BucketDurationSeconds
	if duration <= 0 {
		duration = DefaultQuotaBucketDurationSeconds
	}
	now := common.GetTimestamp()
	title := strings.TrimSpace(redemption.Name)
	if title == "" {
		title = "一周畅用包"
	}
	bucket := &QuotaBucket{
		UserId:             userId,
		Title:              title,
		AmountTotal:        int64(redemption.Quota),
		AmountUsed:         0,
		AmountUsedTotal:    0,
		StartTime:          now,
		EndTime:            now + duration,
		Status:             QuotaBucketStatusActive,
		Source:             QuotaBucketSourceRedemption,
		SourceRedemptionId: redemption.Id,
		SourcePlanId:       redemption.PlanId,
	}
	if err := tx.Create(bucket).Error; err != nil {
		return nil, err
	}
	return bucket, nil
}

func PreConsumeUserQuotaBuckets(requestId string, userId int, amount int64) (*QuotaBucketPreConsumeResult, error) {
	if strings.TrimSpace(requestId) == "" {
		return nil, errors.New("requestId is empty")
	}
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	var result *QuotaBucketPreConsumeResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		record, err := getOrCreateQuotaBucketPreConsumeRecordTx(tx, requestId, userId)
		if err != nil {
			return err
		}
		if record.Status == "refunded" {
			return errors.New("quota bucket pre-consume already refunded")
		}
		if record.PreConsumed > 0 {
			result = &QuotaBucketPreConsumeResult{RecordId: record.Id, PreConsumed: record.PreConsumed}
			return nil
		}
		if err := consumeQuotaBucketsTx(tx, record, amount); err != nil {
			return err
		}
		result = &QuotaBucketPreConsumeResult{RecordId: record.Id, PreConsumed: amount}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("quota bucket pre-consume result is nil")
	}
	fillQuotaBucketPreConsumeResult(result, userId)
	return result, nil
}

func PostConsumeQuotaBucketDelta(requestId string, delta int64) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	if delta == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record QuotaBucketPreConsumeRecord
		if err := withUpdateLock(tx).Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return errors.New("quota bucket pre-consume already refunded")
		}
		if delta > 0 {
			if err := consumeQuotaBucketsTx(tx, &record, delta); err != nil {
				return err
			}
			return nil
		}
		return refundQuotaBucketAllocationsTx(tx, &record, -delta, false)
	})
}

func RefundQuotaBucketPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record QuotaBucketPreConsumeRecord
		if err := withUpdateLock(tx).Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return nil
		}
		if err := refundQuotaBucketAllocationsTx(tx, &record, record.PreConsumed, true); err != nil {
			return err
		}
		record.Status = "refunded"
		record.PreConsumed = 0
		return tx.Save(&record).Error
	})
}

func CleanupQuotaBucketPreConsumeRecords(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 7 * 24 * 3600
	}
	cutoff := common.GetTimestamp() - olderThanSeconds
	res := DB.Where("updated_at < ? AND status = ?", cutoff, "refunded").Delete(&QuotaBucketPreConsumeRecord{})
	return res.RowsAffected, res.Error
}

func getOrCreateQuotaBucketPreConsumeRecordTx(tx *gorm.DB, requestId string, userId int) (*QuotaBucketPreConsumeRecord, error) {
	var record QuotaBucketPreConsumeRecord
	query := withUpdateLock(tx).Where("request_id = ?", requestId).Limit(1).Find(&record)
	if query.Error != nil {
		return nil, query.Error
	}
	if query.RowsAffected > 0 {
		return &record, nil
	}
	record = QuotaBucketPreConsumeRecord{
		RequestId:   requestId,
		UserId:      userId,
		PreConsumed: 0,
		Status:      "consumed",
	}
	if err := tx.Create(&record).Error; err != nil {
		var existing QuotaBucketPreConsumeRecord
		if err2 := withUpdateLock(tx).Where("request_id = ?", requestId).First(&existing).Error; err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	return &record, nil
}

func consumeQuotaBucketsTx(tx *gorm.DB, record *QuotaBucketPreConsumeRecord, amount int64) error {
	if amount <= 0 {
		return nil
	}
	now := common.GetTimestamp()
	var buckets []QuotaBucket
	if err := withUpdateLock(tx).
		Where("user_id = ? AND status = ? AND end_time > ? AND amount_total > amount_used", record.UserId, QuotaBucketStatusActive, now).
		Order("end_time asc, id asc").
		Find(&buckets).Error; err != nil {
		return err
	}
	remaining := amount
	allocations := make([]QuotaBucketPreConsumeAllocation, 0)
	for _, bucket := range buckets {
		if remaining <= 0 {
			break
		}
		available := bucket.AmountTotal - bucket.AmountUsed
		if available <= 0 {
			continue
		}
		use := available
		if use > remaining {
			use = remaining
		}
		bucket.AmountUsed += use
		bucket.AmountUsedTotal += use
		if bucket.AmountUsed >= bucket.AmountTotal {
			bucket.Status = QuotaBucketStatusEmpty
		}
		if err := tx.Save(&bucket).Error; err != nil {
			return err
		}
		allocations = append(allocations, QuotaBucketPreConsumeAllocation{
			RecordId:  record.Id,
			RequestId: record.RequestId,
			BucketId:  bucket.Id,
			Amount:    use,
		})
		remaining -= use
	}
	if remaining > 0 {
		return fmt.Errorf("quota bucket insufficient, need=%d", amount)
	}
	for _, allocation := range allocations {
		if err := tx.Create(&allocation).Error; err != nil {
			return err
		}
	}
	record.PreConsumed += amount
	record.Status = "consumed"
	return tx.Save(record).Error
}

func refundQuotaBucketAllocationsTx(tx *gorm.DB, record *QuotaBucketPreConsumeRecord, amount int64, all bool) error {
	if amount <= 0 && !all {
		return nil
	}
	var allocations []QuotaBucketPreConsumeAllocation
	if err := withUpdateLock(tx).
		Where("record_id = ? AND amount > 0", record.Id).
		Order("id desc").
		Find(&allocations).Error; err != nil {
		return err
	}
	remaining := amount
	for _, allocation := range allocations {
		if !all && remaining <= 0 {
			break
		}
		refund := allocation.Amount
		if !all && refund > remaining {
			refund = remaining
		}
		var bucket QuotaBucket
		if err := withUpdateLock(tx).Where("id = ?", allocation.BucketId).First(&bucket).Error; err != nil {
			return err
		}
		bucket.AmountUsed -= refund
		if bucket.AmountUsed < 0 {
			bucket.AmountUsed = 0
		}
		if bucket.Status == QuotaBucketStatusEmpty && bucket.EndTime > common.GetTimestamp() {
			bucket.Status = QuotaBucketStatusActive
		}
		if err := tx.Save(&bucket).Error; err != nil {
			return err
		}
		allocation.Amount -= refund
		if err := tx.Save(&allocation).Error; err != nil {
			return err
		}
		record.PreConsumed -= refund
		if record.PreConsumed < 0 {
			record.PreConsumed = 0
		}
		remaining -= refund
	}
	if !all && remaining > 0 {
		return fmt.Errorf("quota bucket refund exceeds pre-consumed amount, need=%d", amount)
	}
	return tx.Save(record).Error
}

func fillQuotaBucketPreConsumeResult(result *QuotaBucketPreConsumeResult, userId int) {
	if result == nil {
		return
	}
	self, err := GetUserQuotaBucketSelf(userId)
	if err != nil || self == nil {
		return
	}
	result.TotalRemaining = self.TotalRemaining
	result.NearestEndTime = self.NearestEndTime
	result.AllocationCount = self.ActiveBucketCount
}

func quotaBucketRemaining(bucket QuotaBucket) int64 {
	remaining := bucket.AmountTotal - bucket.AmountUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

func effectiveQuotaBucketStatus(bucket QuotaBucket, now int64) string {
	if bucket.Status == QuotaBucketStatusMigrated {
		return QuotaBucketStatusMigrated
	}
	if bucket.EndTime > 0 && bucket.EndTime <= now {
		return QuotaBucketStatusExpired
	}
	if bucket.AmountTotal > 0 && bucket.AmountUsed >= bucket.AmountTotal {
		return QuotaBucketStatusEmpty
	}
	if strings.TrimSpace(bucket.Status) == "" {
		return QuotaBucketStatusActive
	}
	return bucket.Status
}
