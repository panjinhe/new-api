package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertUserForBulkQuotaTest(t *testing.T, id int, username string, quota int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: username,
		Status:   common.UserStatusEnabled,
		Quota:    quota,
		AffCode:  fmt.Sprintf("bulk-quota-aff-%d", id),
	}
	require.NoError(t, DB.Create(user).Error)
}

func getUserQuotaForBulkQuotaTest(t *testing.T, userID int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", userID).First(&user).Error)
	return user.Quota
}

func TestGrantQuotaToAllUsers_IncreasesAllUsers(t *testing.T) {
	truncateTables(t)

	insertUserForBulkQuotaTest(t, 801, "bulk-quota-user-1", 100)
	insertUserForBulkQuotaTest(t, 802, "bulk-quota-user-2", 250)

	affected, err := GrantQuotaToAllUsers(500)
	require.NoError(t, err)

	assert.Equal(t, int64(2), affected)
	assert.Equal(t, 600, getUserQuotaForBulkQuotaTest(t, 801))
	assert.Equal(t, 750, getUserQuotaForBulkQuotaTest(t, 802))
}

func TestGrantQuotaToAllUsers_RejectsNonPositiveQuota(t *testing.T) {
	truncateTables(t)

	affected, err := GrantQuotaToAllUsers(0)
	require.Error(t, err)
	assert.Equal(t, int64(0), affected)
}
