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
import { TabPane, Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import CardPro from '../../common/ui/CardPro';
import UsersTable from './UsersTable';
import UsersActions from './UsersActions';
import UsersFilters from './UsersFilters';
import UsersDescription from './UsersDescription';
import AddUserModal from './modals/AddUserModal';
import EditUserModal from './modals/EditUserModal';
import { useUsersData } from '../../../hooks/users/useUsersData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';
import UserSubscriptionsOverview from './UserSubscriptionsOverview';

const UsersManagementPanel = ({ tabsArea }) => {
  const usersData = useUsersData();
  const isMobile = useIsMobile();

  const {
    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    closeAddUser,
    closeEditUser,
    refresh,
    classifyUsersByPaymentAndUsage,

    // Form state
    formInitValues,
    setFormApi,
    searchUsers,
    loadUsers,
    activePage,
    pageSize,
    groupOptions,
    loading,
    searching,
    classifyingUsers,

    // Description state
    compactMode,
    setCompactMode,

    // Translation
    t,
  } = usersData;

  return (
    <>
      <AddUserModal
        refresh={refresh}
        visible={showAddUser}
        handleClose={closeAddUser}
      />

      <EditUserModal
        refresh={refresh}
        visible={showEditUser}
        handleClose={closeEditUser}
        editingUser={editingUser}
      />

      <CardPro
        type='type3'
        descriptionArea={
          <UsersDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        tabsArea={tabsArea}
        actionsArea={
          <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
            <UsersActions
              setShowAddUser={setShowAddUser}
              classifyUsersByPaymentAndUsage={classifyUsersByPaymentAndUsage}
              classifyingUsers={classifyingUsers}
              t={t}
            />

            <UsersFilters
              formInitValues={formInitValues}
              setFormApi={setFormApi}
              searchUsers={searchUsers}
              loadUsers={loadUsers}
              activePage={activePage}
              pageSize={pageSize}
              groupOptions={groupOptions}
              loading={loading}
              searching={searching}
              t={t}
            />
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: usersData.activePage,
          pageSize: usersData.pageSize,
          total: usersData.userCount,
          onPageChange: usersData.handlePageChange,
          onPageSizeChange: usersData.handlePageSizeChange,
          isMobile: isMobile,
          t: usersData.t,
        })}
        t={usersData.t}
      >
        <UsersTable {...usersData} />
      </CardPro>
    </>
  );
};

const UsersPage = () => {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab =
    searchParams.get('tab') === 'subscriptions' ? 'subscriptions' : 'users';

  const handleTabChange = (key) => {
    const next = new URLSearchParams(searchParams);
    next.set('tab', key);
    setSearchParams(next);
  };

  const tabsArea = (
    <Tabs type='line' activeKey={activeTab} onChange={handleTabChange}>
      <TabPane tab={t('用户管理')} itemKey='users' />
      <TabPane tab={t('用户套餐')} itemKey='subscriptions' />
    </Tabs>
  );

  if (activeTab === 'subscriptions') {
    return <UserSubscriptionsOverview tabsArea={tabsArea} />;
  }

  return <UsersManagementPanel tabsArea={tabsArea} />;
};

export default UsersPage;
