package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUserEditPersistsConcurrencyLimit(t *testing.T) {
	truncateTables(t)

	user := &User{
		Id:          9901,
		Username:    "edit-concurrency",
		Password:    "password",
		DisplayName: "edit-concurrency",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	require.NoError(t, DB.Create(user).Error)

	editUser := &User{
		Id:               user.Id,
		Username:         user.Username,
		DisplayName:      user.DisplayName,
		Group:            user.Group,
		Remark:           "special user",
		ConcurrencyLimit: 12,
	}
	require.NoError(t, editUser.Edit(false))

	var got User
	require.NoError(t, DB.First(&got, user.Id).Error)
	require.Equal(t, 12, got.ConcurrencyLimit)
}
