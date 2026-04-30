/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button, Popconfirm } from '@douyinfe/semi-ui';
import { IconUserAdd, IconUserGroup } from '@douyinfe/semi-icons';

const UsersActions = ({
  setShowAddUser,
  classifyUsersByPaymentAndUsage,
  classifyingUsers,
  t,
}) => {
  // Add new user
  const handleAddUser = () => {
    setShowAddUser(true);
  };

  return (
    <div className='flex flex-wrap gap-2 w-full md:w-auto order-2 md:order-1'>
      <Button
        className='flex-1 md:flex-initial'
        icon={<IconUserAdd />}
        onClick={handleAddUser}
        size='small'
      >
        {t('添加用户')}
      </Button>
      <Popconfirm
        title={t('确认归类用户分组？')}
        content={t(
          '系统会将累计充值或使用金额达到 50、或历史绑过套餐的普通用户设置为“充值用户”，其余普通用户设置为“白嫖怪”。管理员和已注销用户不会被修改。',
        )}
        okText={t('开始归类')}
        cancelText={t('取消')}
        onConfirm={classifyUsersByPaymentAndUsage}
      >
        <Button
          className='flex-1 md:flex-initial'
          icon={<IconUserGroup />}
          loading={classifyingUsers}
          size='small'
          type='warning'
        >
          {t('归类白嫖怪')}
        </Button>
      </Popconfirm>
    </div>
  );
};

export default UsersActions;
