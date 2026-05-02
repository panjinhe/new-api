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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Empty,
  Form,
  Modal,
  Progress,
  Space,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Ban, Settings2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import CardPro from '../../common/ui/CardPro';
import CardTable from '../../common/ui/CardTable';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import {
  API,
  renderGroup,
  renderNumber,
  renderQuota,
  showError,
  showSuccess,
} from '../../../helpers';
import {
  createCardProPagination,
  timestamp2string,
} from '../../../helpers/utils';
import { ADMIN_ITEMS_PER_PAGE } from '../../../constants';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { useTableCompactMode } from '../../../hooks/common/useTableCompactMode';
import UserSubscriptionsModal from './modals/UserSubscriptionsModal';

const { Text } = Typography;
const USER_SUBSCRIPTIONS_PAGE_SIZE_STORAGE_KEY =
  'admin-user-subscriptions-page-size';
const QUOTA_BUCKETS_PAGE_SIZE_STORAGE_KEY = 'admin-quota-buckets-page-size';

const formatTs = (timestamp) => {
  const value = Number(timestamp || 0);
  if (value <= 0) return '-';
  return timestamp2string(value);
};

const renderUserCell = (record, t) => {
  const remark = record.remark || '';
  return (
    <div className='min-w-0'>
      <div className='flex items-center gap-2'>
        <Text strong ellipsis={{ showTooltip: true }} style={{ maxWidth: 180 }}>
          {record.username || '-'}
        </Text>
        <Tag color='white' shape='circle' size='small'>
          {t('ID')} {record.id}
        </Tag>
      </div>
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        {record.display_name || record.email || t('无显示信息')}
      </div>
      {remark ? (
        <Tooltip content={remark}>
          <div className='text-xs text-[var(--semi-color-text-2)] truncate max-w-[220px]'>
            {remark}
          </div>
        </Tooltip>
      ) : null}
    </div>
  );
};

const renderPlanCell = (summary, t) => {
  if (!summary?.active_count) {
    return (
      <Tag color='grey' shape='circle'>
        {t('无生效套餐')}
      </Tag>
    );
  }
  return (
    <div className='min-w-0'>
      <div className='flex items-center gap-2'>
        <Tag color='green' shape='circle'>
          {summary.primary_plan_title || t('套餐')}
        </Tag>
        {summary.active_count > 1 ? (
          <Tag color='white' shape='circle' size='small'>
            +{summary.active_count - 1}
          </Tag>
        ) : null}
      </div>
      <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
        #{summary.primary_subscription_id || '-'}
      </div>
    </div>
  );
};

const renderRemainingCell = (summary, t) => {
  if (!summary?.active_count) {
    return <Text type='tertiary'>-</Text>;
  }
  const days = Number(summary.remaining_days || 0);
  const color = days <= 3 ? 'orange' : 'green';
  return (
    <div>
      <Tag color={color} shape='circle'>
        {days <= 0 ? t('今天到期') : `${days} ${t('天')}`}
      </Tag>
      <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
        {formatTs(summary.end_time)}
      </div>
    </div>
  );
};

const renderPeriodCell = (summary, t) => {
  const used = Number(summary?.period_used || 0);
  const total = Number(summary?.period_total || 0);
  const remain = Number(summary?.period_remain || 0);
  if (!summary?.active_count) {
    return <Text type='tertiary'>-</Text>;
  }
  if (total <= 0) {
    return (
      <div>
        <Text>{t('不限')}</Text>
        <div className='text-xs text-[var(--semi-color-text-2)]'>
          {t('已用')}: {renderQuota(used)}
        </div>
      </div>
    );
  }
  const percent = Math.min(
    100,
    Math.max(0, Number(summary.usage_percent || 0)),
  );
  return (
    <div className='min-w-[160px]'>
      <div className='text-xs whitespace-nowrap'>
        {renderQuota(used)} / {renderQuota(total)}
      </div>
      <Progress
        percent={percent}
        aria-label='subscription usage'
        format={() => `${percent.toFixed(0)}%`}
        style={{ marginTop: 2, marginBottom: 0 }}
      />
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        {t('剩余')}: {renderQuota(remain)}
      </div>
    </div>
  );
};

const renderActualUsageCell = (summary, t) => {
  if (!summary?.active_count) {
    return <Text type='tertiary'>-</Text>;
  }
  if (Number(summary?.period_total || 0) <= 0) {
    return <Text type='tertiary'>{t('不限')}</Text>;
  }
  const used = Number(summary?.period_elapsed_used || 0);
  const theoretical = Number(summary?.period_elapsed_quota || 0);
  if (theoretical <= 0) {
    return (
      <div>
        <Text type='tertiary'>-</Text>
        <div className='text-xs text-[var(--semi-color-text-2)]'>
          {t('理论')}: {renderQuota(0)}
        </div>
      </div>
    );
  }
  const percent = Number(summary?.actual_usage_percent || 0);
  const color = percent >= 100 ? 'red' : percent >= 80 ? 'orange' : 'green';
  return (
    <div>
      <Tag color={color} shape='circle'>
        {percent.toFixed(0)}%
      </Tag>
      <div className='mt-1 text-xs text-[var(--semi-color-text-2)] whitespace-nowrap'>
        {renderQuota(used)} / {renderQuota(theoretical)}
      </div>
    </div>
  );
};

const renderBucketStatusTag = (status, t) => {
  if (status === 'active') {
    return (
      <Tag color='green' shape='circle'>
        {t('生效中')}
      </Tag>
    );
  }
  if (status === 'empty') {
    return (
      <Tag color='orange' shape='circle'>
        {t('已用完')}
      </Tag>
    );
  }
  if (status === 'cancelled') {
    return (
      <Tag color='red' shape='circle'>
        {t('已作废')}
      </Tag>
    );
  }
  if (status === 'migrated') {
    return (
      <Tag color='blue' shape='circle'>
        {t('已迁移')}
      </Tag>
    );
  }
  return (
    <Tag color='grey' shape='circle'>
      {t('已过期')}
    </Tag>
  );
};

const renderBucketUserCell = (record, t) => {
  const bucket = record?.bucket || {};
  return (
    <div className='min-w-0'>
      <div className='flex items-center gap-2'>
        <Text strong ellipsis={{ showTooltip: true }} style={{ maxWidth: 180 }}>
          {record.username || '-'}
        </Text>
        <Tag color='white' shape='circle' size='small'>
          {t('ID')} {bucket.user_id || '-'}
        </Tag>
      </div>
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        {record.display_name || record.email || t('无显示信息')}
      </div>
      {record.remark ? (
        <Tooltip content={record.remark}>
          <div className='text-xs text-[var(--semi-color-text-2)] truncate max-w-[220px]'>
            {record.remark}
          </div>
        </Tooltip>
      ) : null}
    </div>
  );
};

const renderBucketInfoCell = (record, t) => {
  const bucket = record?.bucket || {};
  return (
    <div className='min-w-0'>
      <div className='font-medium truncate'>
        {bucket.title || `${t('限时额度包')} #${bucket.id}`}
      </div>
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        #{bucket.id} · {t('兑换码')} #{bucket.source_redemption_id || '-'}
      </div>
      {record.batch_id ? (
        <div className='text-xs text-[var(--semi-color-text-2)] truncate'>
          {t('批次')}: {record.batch_id}
        </div>
      ) : null}
    </div>
  );
};

const renderBucketUsageCell = (record, t) => {
  const bucket = record?.bucket || {};
  const total = Number(bucket.amount_total || 0);
  const used = Number(bucket.amount_used || 0);
  const remaining = Number(record?.remaining_quota || 0);
  const percent =
    total > 0 ? Math.min(100, Math.max(0, (used / total) * 100)) : 0;
  return (
    <div className='min-w-[180px]'>
      <div className='flex justify-between gap-2 text-xs'>
        <span>
          {renderQuota(used)} / {renderQuota(total)}
        </span>
        <Text type='secondary'>{percent.toFixed(0)}%</Text>
      </div>
      <Progress
        percent={percent}
        aria-label='quota bucket usage'
        format={() => ''}
        style={{ marginTop: 2, marginBottom: 0 }}
      />
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        {t('剩余')}: {renderQuota(remaining)}
      </div>
    </div>
  );
};

const buildColumns = ({ t, openSubscriptions }) => [
  {
    title: t('用户'),
    key: 'user',
    width: 240,
    render: (_, record) => renderUserCell(record, t),
  },
  {
    title: t('分组'),
    dataIndex: 'group',
    width: 120,
    render: (group) => renderGroup(group),
  },
  {
    title: t('生效套餐'),
    key: 'plan',
    width: 180,
    render: (_, record) => renderPlanCell(record.subscription_summary, t),
  },
  {
    title: t('剩余天数'),
    key: 'remaining_days',
    width: 160,
    render: (_, record) => renderRemainingCell(record.subscription_summary, t),
  },
  {
    title: t('下次重置'),
    key: 'next_reset_time',
    width: 160,
    render: (_, record) => (
      <Text type='secondary'>
        {formatTs(record.subscription_summary?.next_reset_time)}
      </Text>
    ),
  },
  {
    title: t('今日套餐用量'),
    key: 'today_used',
    width: 150,
    render: (_, record) => {
      const value = Number(record.subscription_summary?.today_used || 0);
      return (
        <div>
          <Text strong={value !== 0}>{renderQuota(value)}</Text>
          <div className='text-xs text-[var(--semi-color-text-2)]'>
            {t('净用量')}
          </div>
        </div>
      );
    },
  },
  {
    title: t('当前周期额度'),
    key: 'period',
    width: 220,
    render: (_, record) => renderPeriodCell(record.subscription_summary, t),
  },
  {
    title: t('实际使用率'),
    key: 'actual_usage_percent',
    width: 150,
    render: (_, record) =>
      renderActualUsageCell(record.subscription_summary, t),
  },
  {
    title: t('累计套餐用量'),
    key: 'lifetime_used',
    width: 150,
    render: (_, record) => (
      <Text strong>
        {renderQuota(record.subscription_summary?.lifetime_used || 0)}
      </Text>
    ),
  },
  {
    title: '',
    key: 'operate',
    width: 130,
    fixed: 'right',
    render: (_, record) => (
      <Button
        size='small'
        type='tertiary'
        icon={<Settings2 size={14} />}
        onClick={() => openSubscriptions(record)}
      >
        {t('权益管理')}
      </Button>
    ),
  },
];

const buildBucketColumns = ({ t, openSubscriptions, invalidateQuotaBucket }) => [
  {
    title: t('用户'),
    key: 'user',
    width: 240,
    render: (_, record) => renderBucketUserCell(record, t),
  },
  {
    title: t('限时额度包'),
    key: 'bucket',
    width: 240,
    render: (_, record) => renderBucketInfoCell(record, t),
  },
  {
    title: t('状态'),
    key: 'status',
    width: 110,
    render: (_, record) => renderBucketStatusTag(record.status, t),
  },
  {
    title: t('使用情况'),
    key: 'usage',
    width: 240,
    render: (_, record) => renderBucketUsageCell(record, t),
  },
  {
    title: t('有效期'),
    key: 'validity',
    width: 220,
    render: (_, record) => (
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        <div>
          {t('开始')}: {formatTs(record.bucket?.start_time)}
        </div>
        <div>
          {t('到期')}: {formatTs(record.bucket?.end_time)}
        </div>
      </div>
    ),
  },
  {
    title: t('来源'),
    key: 'source',
    width: 210,
    render: (_, record) => (
      <div className='text-xs text-[var(--semi-color-text-2)]'>
        <div>{record.redemption_name || record.bucket?.source || '-'}</div>
        <div className='truncate'>
          {t('兑换码')}: {record.redemption_key || '-'}
        </div>
      </div>
    ),
  },
  {
    title: '',
    key: 'operate',
    width: 170,
    fixed: 'right',
    render: (_, record) => {
      const bucket = record?.bucket || {};
      const user = {
        id: bucket.user_id,
        username: record.username,
        display_name: record.display_name,
        email: record.email,
        remark: record.remark,
      };
      return (
        <Space wrap>
          <Button
            size='small'
            type='tertiary'
            icon={<Settings2 size={14} />}
            onClick={() => openSubscriptions(user)}
          >
            {t('权益')}
          </Button>
          <Button
            size='small'
            type='danger'
            theme='light'
            icon={<Ban size={14} />}
            disabled={record.status !== 'active'}
            onClick={() => invalidateQuotaBucket(bucket.id)}
          >
            {t('作废')}
          </Button>
        </Space>
      );
    },
  },
];

const UserSubscriptionFilters = ({
  formInitValues,
  setFormApi,
  loadData,
  resetData,
  pageSize,
  groupOptions,
  planOptions,
  loading,
  t,
}) => {
  const formApiRef = useRef(null);
  const submit = () => {
    const values = formApiRef.current?.getValues?.() || {};
    if (values.keyword?.trim() && values.status === 'active') {
      formApiRef.current?.setValue?.('status', 'all');
    }
    loadData(1, pageSize);
  };
  const reset = () => {
    formApiRef.current?.reset();
    setTimeout(() => resetData(), 100);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={submit}
      allowEmpty
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='w-full'
    >
      <div className='grid grid-cols-1 md:grid-cols-4 xl:grid-cols-8 gap-2 w-full'>
        <Form.Input
          field='keyword'
          prefix={<IconSearch />}
          placeholder={t('搜索用户')}
          showClear
          pure
          size='small'
        />
        <Form.Select
          field='group'
          placeholder={t('选择分组')}
          optionList={groupOptions}
          showClear
          pure
          size='small'
        />
        <Form.Select
          field='plan_id'
          placeholder={t('选择套餐')}
          optionList={planOptions}
          showClear
          filter
          pure
          size='small'
        />
        <Form.Select
          field='status'
          placeholder={t('套餐状态')}
          optionList={[
            { label: t('有生效套餐'), value: 'active' },
            { label: t('全部'), value: 'all' },
            { label: t('无生效套餐'), value: 'none' },
            { label: t('已过期套餐'), value: 'expired' },
            { label: t('即将到期'), value: 'expiring' },
          ]}
          pure
          size='small'
        />
        <Form.Select
          field='expire_days'
          placeholder={t('到期窗口')}
          optionList={[
            { label: `3 ${t('天')}`, value: 3 },
            { label: `7 ${t('天')}`, value: 7 },
            { label: `30 ${t('天')}`, value: 30 },
          ]}
          pure
          size='small'
        />
        <Form.Select
          field='usage_filter'
          placeholder={t('用量筛选')}
          optionList={[
            { label: t('全部'), value: '' },
            { label: t('今日用量大于 0'), value: 'today_gt_zero' },
            { label: t('使用率 80%+'), value: 'usage80' },
            { label: t('使用率 95%+'), value: 'usage95' },
          ]}
          pure
          size='small'
        />
        <Form.Select
          field='sort'
          placeholder={t('排序')}
          optionList={[
            { label: t('用户 ID'), value: 'id' },
            { label: t('剩余天数'), value: 'remaining_days' },
            { label: t('今日套餐用量'), value: 'today_used' },
            { label: t('累计套餐用量'), value: 'lifetime_used' },
            { label: t('周期使用率'), value: 'usage_percent' },
            { label: t('实际使用率'), value: 'actual_usage_percent' },
          ]}
          pure
          size='small'
        />
        <div className='flex gap-2'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading}
            className='flex-1'
            size='small'
          >
            {t('查询')}
          </Button>
          <Button
            type='tertiary'
            onClick={reset}
            className='flex-1'
            size='small'
          >
            {t('重置')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

const QuotaBucketFilters = ({
  formInitValues,
  setFormApi,
  loadData,
  resetData,
  pageSize,
  loading,
  t,
}) => {
  const formApiRef = useRef(null);
  const submit = () => {
    loadData(1, pageSize);
  };
  const reset = () => {
    formApiRef.current?.reset();
    setTimeout(() => resetData(), 100);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={submit}
      allowEmpty
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='w-full'
    >
      <div className='grid grid-cols-1 md:grid-cols-4 xl:grid-cols-6 gap-2 w-full'>
        <Form.Input
          field='keyword'
          prefix={<IconSearch />}
          placeholder={t('搜索用户、兑换码、批次')}
          showClear
          pure
          size='small'
        />
        <Form.Select
          field='status'
          placeholder={t('额度包状态')}
          optionList={[
            { label: t('生效和已过期'), value: 'active_expired' },
            { label: t('生效中'), value: 'active' },
            { label: t('已过期'), value: 'expired' },
            { label: t('已用完'), value: 'empty' },
            { label: t('已作废'), value: 'cancelled' },
            { label: t('全部'), value: 'all' },
          ]}
          pure
          size='small'
        />
        <Form.Select
          field='sort'
          placeholder={t('排序')}
          optionList={[
            { label: t('到期时间'), value: 'end_time' },
            { label: t('创建时间'), value: 'created_at' },
            { label: t('额度包 ID'), value: 'id' },
            { label: t('已用额度'), value: 'amount_used' },
            { label: t('剩余额度'), value: 'remaining_quota' },
            { label: t('使用率'), value: 'usage_percent' },
          ]}
          pure
          size='small'
        />
        <Form.Select
          field='order'
          placeholder={t('顺序')}
          optionList={[
            { label: t('升序'), value: 'asc' },
            { label: t('降序'), value: 'desc' },
          ]}
          pure
          size='small'
        />
        <div className='flex gap-2 md:col-span-2'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading}
            className='flex-1'
            size='small'
          >
            {t('查询')}
          </Button>
          <Button
            type='tertiary'
            onClick={reset}
            className='flex-1'
            size='small'
          >
            {t('重置')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

const UserSubscriptionsOverview = ({ tabsArea }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [compactMode, setCompactMode] =
    useTableCompactMode('user-subscriptions');
  const [activeView, setActiveView] = useState('subscriptions');
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(
    parseInt(localStorage.getItem(USER_SUBSCRIPTIONS_PAGE_SIZE_STORAGE_KEY)) ||
      ADMIN_ITEMS_PER_PAGE,
  );
  const [total, setTotal] = useState(0);
  const [formApi, setFormApi] = useState(null);
  const [bucketRows, setBucketRows] = useState([]);
  const [bucketLoading, setBucketLoading] = useState(false);
  const [bucketActivePage, setBucketActivePage] = useState(1);
  const [bucketPageSize, setBucketPageSize] = useState(
    parseInt(localStorage.getItem(QUOTA_BUCKETS_PAGE_SIZE_STORAGE_KEY)) ||
      ADMIN_ITEMS_PER_PAGE,
  );
  const [bucketTotal, setBucketTotal] = useState(0);
  const [bucketFormApi, setBucketFormApi] = useState(null);
  const [groupOptions, setGroupOptions] = useState([]);
  const [planOptions, setPlanOptions] = useState([]);
  const [modalUser, setModalUser] = useState(null);
  const [showUserSubscriptionsModal, setShowUserSubscriptionsModal] =
    useState(false);

  const formInitValues = {
    keyword: '',
    group: '',
    plan_id: undefined,
    status: 'active',
    expire_days: 7,
    usage_filter: '',
    sort: 'id',
  };
  const bucketFormInitValues = {
    keyword: '',
    status: 'active_expired',
    sort: 'end_time',
    order: 'asc',
  };

  const getFormValues = () => formApi?.getValues?.() || formInitValues;
  const getBucketFormValues = () =>
    bucketFormApi?.getValues?.() || bucketFormInitValues;

  const buildQuery = (page, size) => {
    const values = getFormValues();
    const params = new URLSearchParams();
    params.set('p', page);
    params.set('page_size', size);
    if (values.keyword) params.set('keyword', values.keyword);
    if (values.group) params.set('group', values.group);
    if (values.plan_id) params.set('plan_id', values.plan_id);
    const keyword = values.keyword?.trim();
    const status =
      keyword && values.status === 'active' ? 'all' : values.status;
    params.set('status', status || 'active');
    params.set('expire_days', values.expire_days || 7);
    if (values.usage_filter === 'today_gt_zero') {
      params.set('min_today_used', 1);
    }
    if (values.usage_filter === 'usage80') {
      params.set('min_usage_percent', 80);
    }
    if (values.usage_filter === 'usage95') {
      params.set('min_usage_percent', 95);
    }
    params.set('sort', values.sort || 'id');
    const ascSorts = ['remaining_days', 'end_time'];
    params.set('order', ascSorts.includes(values.sort) ? 'asc' : 'desc');
    return params.toString();
  };

  const loadData = async (page = activePage, size = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/subscription/admin/user-subscription-summaries?${buildQuery(
          page,
          size,
        )}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        const items = data.items || [];
        setRows(items.map((item) => ({ ...item, key: item.id })));
        setActivePage(data.page);
        setTotal(data.total);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  const buildBucketQuery = (page, size) => {
    const values = getBucketFormValues();
    const params = new URLSearchParams();
    params.set('p', page);
    params.set('page_size', size);
    if (values.keyword) params.set('keyword', values.keyword);
    params.set('status', values.status || 'active_expired');
    params.set('sort', values.sort || 'end_time');
    params.set('order', values.order || 'asc');
    return params.toString();
  };

  const loadBucketData = async (
    page = bucketActivePage,
    size = bucketPageSize,
  ) => {
    setBucketLoading(true);
    try {
      const res = await API.get(
        `/api/subscription/admin/quota-buckets?${buildBucketQuery(page, size)}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        const items = data.items || [];
        setBucketRows(
          items.map((item) => ({
            ...item,
            key: item?.bucket?.id,
          })),
        );
        setBucketActivePage(data.page);
        setBucketTotal(data.total);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setBucketLoading(false);
    }
  };

  const resetData = async () => {
    setActivePage(1);
    await loadData(1, pageSize);
  };

  const resetBucketData = async () => {
    setBucketActivePage(1);
    await loadBucketData(1, bucketPageSize);
  };

  const fetchGroups = async () => {
    setGroupOptions([
      { label: t('充值用户'), value: '充值用户' },
      { label: t('白嫖怪'), value: '白嫖怪' },
    ]);
  };

  const fetchPlans = async () => {
    try {
      const res = await API.get('/api/subscription/admin/plans');
      if (res.data?.success) {
        setPlanOptions(
          (res.data.data || []).map((item) => ({
            label: item?.plan?.title || `#${item?.plan?.id}`,
            value: item?.plan?.id,
          })),
        );
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    }
  };

  useEffect(() => {
    loadData(1, pageSize);
    loadBucketData(1, bucketPageSize);
    fetchGroups();
    fetchPlans();
  }, []);

  const handlePageChange = (page) => {
    loadData(page, pageSize);
  };

  const handlePageSizeChange = (size) => {
    localStorage.setItem(USER_SUBSCRIPTIONS_PAGE_SIZE_STORAGE_KEY, `${size}`);
    setPageSize(size);
    loadData(1, size);
  };

  const handleBucketPageChange = (page) => {
    loadBucketData(page, bucketPageSize);
  };

  const handleBucketPageSizeChange = (size) => {
    localStorage.setItem(QUOTA_BUCKETS_PAGE_SIZE_STORAGE_KEY, `${size}`);
    setBucketPageSize(size);
    loadBucketData(1, size);
  };

  const openSubscriptions = (record) => {
    setModalUser(record);
    setShowUserSubscriptionsModal(true);
  };

  const invalidateQuotaBucket = (bucketId) => {
    Modal.confirm({
      title: t('确认作废'),
      content: t('作废后该限时额度包将立即失效，已用量和兑换记录会保留。是否继续？'),
      centered: true,
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await API.post(
            `/api/subscription/admin/quota_buckets/${bucketId}/invalidate`,
          );
          if (res.data?.success) {
            const msg = res.data?.data?.message;
            showSuccess(msg ? msg : t('已作废'));
            await loadBucketData(bucketActivePage, bucketPageSize);
            await loadData(activePage, pageSize);
          } else {
            showError(res.data?.message || t('操作失败'));
          }
        } catch (e) {
          showError(t('请求失败'));
        }
      },
    });
  };

  const columns = useMemo(() => buildColumns({ t, openSubscriptions }), [t]);
  const bucketColumns = useMemo(
    () => buildBucketColumns({ t, openSubscriptions, invalidateQuotaBucket }),
    [t, bucketActivePage, bucketPageSize],
  );

  const tableColumns = useMemo(() => {
    return compactMode
      ? columns.map((col) => {
          if (col.key === 'operate') {
            const { fixed, ...rest } = col;
            return rest;
          }
          return col;
        })
      : columns;
  }, [compactMode, columns]);

  const activeUsers = rows.filter(
    (row) => Number(row.subscription_summary?.active_count || 0) > 0,
  ).length;
  const todayUsed = rows.reduce(
    (sum, row) => sum + Number(row.subscription_summary?.today_used || 0),
    0,
  );
  const activeBucketCount = bucketRows.filter(
    (row) => row.status === 'active',
  ).length;
  const bucketRemaining = bucketRows.reduce(
    (sum, row) => sum + Number(row.remaining_quota || 0),
    0,
  );
  const isBucketView = activeView === 'buckets';

  return (
    <>
      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
            <div className='flex items-center gap-2 text-blue-500'>
              <Text>{t('用户权益')}</Text>
              {isBucketView ? (
                <>
                  <Tag color='white' shape='circle'>
                    {t('本页生效')}: {renderNumber(activeBucketCount)}
                  </Tag>
                  <Tag color='white' shape='circle'>
                    {t('本页剩余')}: {renderQuota(bucketRemaining)}
                  </Tag>
                </>
              ) : (
                <>
                  <Tag color='white' shape='circle'>
                    {t('本页生效')}: {renderNumber(activeUsers)}
                  </Tag>
                  <Tag color='white' shape='circle'>
                    {t('本页今日')}: {renderQuota(todayUsed)}
                  </Tag>
                </>
              )}
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              <Button
                size='small'
                type={activeView === 'subscriptions' ? 'primary' : 'tertiary'}
                theme={activeView === 'subscriptions' ? 'solid' : 'light'}
                onClick={() => setActiveView('subscriptions')}
              >
                {t('月卡套餐')}
              </Button>
              <Button
                size='small'
                type={activeView === 'buckets' ? 'primary' : 'tertiary'}
                theme={activeView === 'buckets' ? 'solid' : 'light'}
                onClick={() => setActiveView('buckets')}
              >
                {t('限时额度包')}
              </Button>
              {!isBucketView ? (
                <CompactModeToggle
                  compactMode={compactMode}
                  setCompactMode={setCompactMode}
                  t={t}
                />
              ) : null}
            </div>
          </div>
        }
        tabsArea={tabsArea}
        actionsArea={
          isBucketView ? (
            <QuotaBucketFilters
              formInitValues={bucketFormInitValues}
              setFormApi={setBucketFormApi}
              loadData={loadBucketData}
              resetData={resetBucketData}
              pageSize={bucketPageSize}
              loading={bucketLoading}
              t={t}
            />
          ) : (
            <UserSubscriptionFilters
              formInitValues={formInitValues}
              setFormApi={setFormApi}
              loadData={loadData}
              resetData={resetData}
              pageSize={pageSize}
              groupOptions={groupOptions}
              planOptions={planOptions}
              loading={loading}
              t={t}
            />
          )
        }
        paginationArea={
          isBucketView
            ? createCardProPagination({
                currentPage: bucketActivePage,
                pageSize: bucketPageSize,
                total: bucketTotal,
                onPageChange: handleBucketPageChange,
                onPageSizeChange: handleBucketPageSizeChange,
                isMobile,
                t,
              })
            : createCardProPagination({
                currentPage: activePage,
                pageSize,
                total,
                onPageChange: handlePageChange,
                onPageSizeChange: handlePageSizeChange,
                isMobile,
                t,
              })
        }
        t={t}
      >
        {isBucketView ? (
          <CardTable
            columns={bucketColumns}
            dataSource={bucketRows}
            scroll={{ x: 'max-content' }}
            pagination={false}
            hidePagination
            loading={bucketLoading}
            empty={
              <Empty
                image={
                  <IllustrationNoResult style={{ width: 150, height: 150 }} />
                }
                darkModeImage={
                  <IllustrationNoResultDark
                    style={{ width: 150, height: 150 }}
                  />
                }
                description={t('暂无限时额度包')}
                style={{ padding: 30 }}
              />
            }
            className='overflow-hidden'
            size='middle'
          />
        ) : (
          <CardTable
            columns={tableColumns}
            dataSource={rows}
            scroll={compactMode ? undefined : { x: 'max-content' }}
            pagination={false}
            hidePagination
            loading={loading}
            empty={
              <Empty
                image={
                  <IllustrationNoResult style={{ width: 150, height: 150 }} />
                }
                darkModeImage={
                  <IllustrationNoResultDark
                    style={{ width: 150, height: 150 }}
                  />
                }
                description={t('搜索无结果')}
                style={{ padding: 30 }}
              />
            }
            className='overflow-hidden'
            size='middle'
          />
        )}
      </CardPro>

      <UserSubscriptionsModal
        visible={showUserSubscriptionsModal}
        onCancel={() => setShowUserSubscriptionsModal(false)}
        user={modalUser}
        t={t}
        onSuccess={() => {
          loadData(activePage, pageSize);
          loadBucketData(bucketActivePage, bucketPageSize);
        }}
      />
    </>
  );
};

export default UserSubscriptionsOverview;
