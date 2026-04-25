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
import { Card, Space } from '@douyinfe/semi-ui';
import SubscriptionPlanCatalog from './SubscriptionPlanCatalog';
import SubscriptionStatusPanel from './SubscriptionStatusPanel';

const SubscriptionPlansCard = ({
  t,
  loading = false,
  plans = [],
  payMethods = [],
  enableOnlineTopUp = false,
  enableStripeTopUp = false,
  enableCreemTopUp = false,
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  withCard = true,
}) => {
  const content = (
    <Space vertical style={{ width: '100%' }}>
      <SubscriptionStatusPanel
        t={t}
        loading={loading}
        plans={plans}
        billingPreference={billingPreference}
        onChangeBillingPreference={onChangeBillingPreference}
        activeSubscriptions={activeSubscriptions}
        allSubscriptions={allSubscriptions}
        reloadSubscriptionSelf={reloadSubscriptionSelf}
        catalogEnabled={plans.length > 0}
      />
      <SubscriptionPlanCatalog
        t={t}
        loading={loading}
        plans={plans}
        payMethods={payMethods}
        enableOnlineTopUp={enableOnlineTopUp}
        enableStripeTopUp={enableStripeTopUp}
        enableCreemTopUp={enableCreemTopUp}
        allSubscriptions={allSubscriptions}
      />
    </Space>
  );

  return withCard ? (
    <Card className='!rounded-2xl shadow-sm border-0'>{content}</Card>
  ) : (
    content
  );
};

export default SubscriptionPlansCard;
