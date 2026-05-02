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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Modal,
  Progress,
  Select,
  SideSheet,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlusCircle } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, renderQuota, showError, showSuccess } from '../../../../helpers';
import { getSubscriptionPriceDisplay } from '../../../../helpers/render';
import { ADMIN_ITEMS_PER_PAGE } from '../../../../constants';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import CardTable from '../../../common/ui/CardTable';

const { Text } = Typography;

function formatTs(ts) {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString();
}

function renderStatusTag(sub, t) {
  const now = Date.now() / 1000;
  const end = sub?.end_time || 0;
  const status = sub?.status || '';

  const isExpiredByTime = end > 0 && end < now;
  const isActive = status === 'active' && !isExpiredByTime;
  if (isActive) {
    return (
      <Tag color='green' shape='circle' size='small'>
        {t('生效')}
      </Tag>
    );
  }
  if (status === 'cancelled') {
    return (
      <Tag color='grey' shape='circle' size='small'>
        {t('已作废')}
      </Tag>
    );
  }
  return (
    <Tag color='grey' shape='circle' size='small'>
      {t('已过期')}
    </Tag>
  );
}

function renderBucketStatusTag(status, t) {
  if (status === 'active') {
    return (
      <Tag color='green' shape='circle' size='small'>
        {t('生效中')}
      </Tag>
    );
  }
  if (status === 'empty') {
    return (
      <Tag color='orange' shape='circle' size='small'>
        {t('已用完')}
      </Tag>
    );
  }
  if (status === 'cancelled') {
    return (
      <Tag color='red' shape='circle' size='small'>
        {t('已作废')}
      </Tag>
    );
  }
  if (status === 'migrated') {
    return (
      <Tag color='blue' shape='circle' size='small'>
        {t('已迁移')}
      </Tag>
    );
  }
  return (
    <Tag color='grey' shape='circle' size='small'>
      {t('已过期')}
    </Tag>
  );
}

function sourceLabel(source, t) {
  if (source === 'redemption') return t('兑换码');
  if (source === 'migration') return t('历史迁移');
  return source || '-';
}

const UserSubscriptionsModal = ({ visible, onCancel, user, t, onSuccess }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [plansLoading, setPlansLoading] = useState(false);

  const [plans, setPlans] = useState([]);
  const [selectedPlanId, setSelectedPlanId] = useState(null);

  const [subs, setSubs] = useState([]);
  const [quotaBuckets, setQuotaBuckets] = useState({
    buckets: [],
    active_buckets: [],
    total_remaining: 0,
    nearest_end_time: 0,
    active_bucket_count: 0,
  });
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = ADMIN_ITEMS_PER_PAGE;

  const planTitleMap = useMemo(() => {
    const map = new Map();
    (plans || []).forEach((p) => {
      const id = p?.plan?.id;
      const title = p?.plan?.title;
      if (id) map.set(id, title || `#${id}`);
    });
    return map;
  }, [plans]);

  const pagedSubs = useMemo(() => {
    const start = Math.max(0, (Number(currentPage || 1) - 1) * pageSize);
    const end = start + pageSize;
    return (subs || []).slice(start, end);
  }, [subs, currentPage]);

  const bucketRows = useMemo(() => {
    const statusOrder = {
      active: 0,
      empty: 1,
      expired: 2,
      migrated: 3,
    };
    return (quotaBuckets.buckets || [])
      .map((summary) => {
        const bucket = summary?.bucket || {};
        const total = Number(bucket.amount_total || 0);
        const used = Number(bucket.amount_used || 0);
        const remaining = Number(summary?.remaining_quota || 0);
        const percent =
          total > 0 ? Math.min(100, Math.max(0, (used / total) * 100)) : 0;
        return {
          ...summary,
          key: bucket.id,
          bucket,
          total,
          used,
          remaining,
          percent,
          status: summary?.status || bucket.status || 'expired',
        };
      })
      .sort((a, b) => {
        const statusDiff =
          (statusOrder[a.status] ?? 99) - (statusOrder[b.status] ?? 99);
        if (statusDiff !== 0) return statusDiff;
        const endDiff =
          Number(a.bucket?.end_time || 0) - Number(b.bucket?.end_time || 0);
        if (endDiff !== 0) return endDiff;
        return Number(a.bucket?.id || 0) - Number(b.bucket?.id || 0);
      });
  }, [quotaBuckets]);

  const allBucketStats = useMemo(() => {
    return bucketRows.reduce(
      (acc, row) => {
        acc.total += row.total;
        acc.used += row.used;
        acc.remaining += row.remaining;
        return acc;
      },
      { total: 0, used: 0, remaining: 0 },
    );
  }, [bucketRows]);

  const planOptions = useMemo(() => {
    return (plans || []).map((p) => ({
      label: `${p?.plan?.title || ''} (${
        getSubscriptionPriceDisplay(p?.plan).label
      })`,
      value: p?.plan?.id,
    }));
  }, [plans]);

  const loadPlans = async () => {
    setPlansLoading(true);
    try {
      const res = await API.get('/api/subscription/admin/plans');
      if (res.data?.success) {
        setPlans(res.data.data || []);
      } else {
        showError(res.data?.message || t('加载失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setPlansLoading(false);
    }
  };

  const loadUserSubscriptions = async () => {
    if (!user?.id) return;
    setLoading(true);
    try {
      const res = await API.get(
        `/api/subscription/admin/users/${user.id}/subscriptions`,
      );
      if (res.data?.success) {
        const next = res.data.data || [];
        setSubs(next);
        setCurrentPage(1);
      } else {
        showError(res.data?.message || t('加载失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadUserQuotaBuckets = async () => {
    if (!user?.id) return;
    try {
      const res = await API.get(
        `/api/subscription/admin/users/${user.id}/quota_buckets`,
      );
      if (res.data?.success) {
        setQuotaBuckets(
          res.data.data || {
            buckets: [],
            active_buckets: [],
            total_remaining: 0,
            nearest_end_time: 0,
            active_bucket_count: 0,
          },
        );
      } else {
        showError(res.data?.message || t('加载失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    }
  };

  useEffect(() => {
    if (!visible) return;
    setSelectedPlanId(null);
    setCurrentPage(1);
    loadPlans();
    loadUserSubscriptions();
    loadUserQuotaBuckets();
  }, [visible]);

  const handlePageChange = (page) => {
    setCurrentPage(page);
  };

  const createSubscription = async () => {
    if (!user?.id) {
      showError(t('用户信息缺失'));
      return;
    }
    if (!selectedPlanId) {
      showError(t('请选择订阅套餐'));
      return;
    }
    setCreating(true);
    try {
      const res = await API.post(
        `/api/subscription/admin/users/${user.id}/subscriptions`,
        {
          plan_id: selectedPlanId,
        },
      );
      if (res.data?.success) {
        const msg = res.data?.data?.message;
        showSuccess(msg ? msg : t('新增成功'));
        setSelectedPlanId(null);
        await loadUserSubscriptions();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('新增失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setCreating(false);
    }
  };

  const invalidateSubscription = (subId) => {
    Modal.confirm({
      title: t('确认作废'),
      content: t('作废后该订阅将立即失效，历史记录不受影响。是否继续？'),
      centered: true,
      onOk: async () => {
        try {
          const res = await API.post(
            `/api/subscription/admin/user_subscriptions/${subId}/invalidate`,
          );
          if (res.data?.success) {
            const msg = res.data?.data?.message;
            showSuccess(msg ? msg : t('已作废'));
            await loadUserSubscriptions();
            await loadUserQuotaBuckets();
            onSuccess?.();
          } else {
            showError(res.data?.message || t('操作失败'));
          }
        } catch (e) {
          showError(t('请求失败'));
        }
      },
    });
  };

  const resetSubscriptionUsage = (subId) => {
    Modal.confirm({
      title: t('确认重置额度'),
      content: t(
        '将清零该订阅当前周期已用额度，恢复当前周期可用额度；历史累计用量和每日统计不会清除。是否继续？',
      ),
      centered: true,
      onOk: async () => {
        try {
          const res = await API.post(
            `/api/subscription/admin/user_subscriptions/${subId}/reset_usage`,
          );
          if (res.data?.success) {
            const msg = res.data?.data?.message;
            showSuccess(msg ? msg : t('已重置'));
            await loadUserSubscriptions();
            onSuccess?.();
          } else {
            showError(res.data?.message || t('操作失败'));
          }
        } catch (e) {
          showError(t('请求失败'));
        }
      },
    });
  };

  const deleteSubscription = (subId) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('删除会彻底移除该订阅记录（含权益明细）。是否继续？'),
      centered: true,
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await API.delete(
            `/api/subscription/admin/user_subscriptions/${subId}`,
          );
          if (res.data?.success) {
            const msg = res.data?.data?.message;
            showSuccess(msg ? msg : t('已删除'));
            await loadUserSubscriptions();
            onSuccess?.();
          } else {
            showError(res.data?.message || t('删除失败'));
          }
        } catch (e) {
          showError(t('请求失败'));
        }
      },
    });
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
            await loadUserQuotaBuckets();
            onSuccess?.();
          } else {
            showError(res.data?.message || t('操作失败'));
          }
        } catch (e) {
          showError(t('请求失败'));
        }
      },
    });
  };

  const columns = useMemo(() => {
    return [
      {
        title: 'ID',
        dataIndex: ['subscription', 'id'],
        key: 'id',
        width: 70,
      },
      {
        title: t('套餐'),
        key: 'plan',
        width: 180,
        render: (_, record) => {
          const sub = record?.subscription;
          const planId = sub?.plan_id;
          const title =
            planTitleMap.get(planId) || (planId ? `#${planId}` : '-');
          return (
            <div className='min-w-0'>
              <div className='font-medium truncate'>{title}</div>
              <div className='text-xs text-gray-500'>
                {t('来源')}: {sub?.source || '-'}
              </div>
            </div>
          );
        },
      },
      {
        title: t('状态'),
        key: 'status',
        width: 90,
        render: (_, record) => renderStatusTag(record?.subscription, t),
      },
      {
        title: t('有效期'),
        key: 'validity',
        width: 200,
        render: (_, record) => {
          const sub = record?.subscription;
          return (
            <div className='text-xs text-gray-600'>
              <div>
                {t('开始')}: {formatTs(sub?.start_time)}
              </div>
              <div>
                {t('结束')}: {formatTs(sub?.end_time)}
              </div>
            </div>
          );
        },
      },
      {
        title: t('当前周期额度'),
        key: 'total',
        width: 120,
        render: (_, record) => {
          const sub = record?.subscription;
          const total = Number(sub?.amount_total || 0);
          const used = Number(sub?.amount_used || 0);
          return (
            <Text type={total > 0 ? 'secondary' : 'tertiary'}>
              {total > 0
                ? `${renderQuota(used)} / ${renderQuota(total)}`
                : `${t('不限')} · ${renderQuota(used)}`}
            </Text>
          );
        },
      },
      {
        title: t('累计用量'),
        key: 'lifetime_used',
        width: 120,
        render: (_, record) => {
          const used = Number(record?.subscription?.amount_used_total || 0);
          return <Text strong>{renderQuota(used)}</Text>;
        },
      },
      {
        title: '',
        key: 'operate',
        width: 230,
        fixed: 'right',
        render: (_, record) => {
          const sub = record?.subscription;
          const now = Date.now() / 1000;
          const isExpired =
            (sub?.end_time || 0) > 0 && (sub?.end_time || 0) < now;
          const isActive = sub?.status === 'active' && !isExpired;
          const isCancelled = sub?.status === 'cancelled';
          const hasUsed = Number(sub?.amount_used || 0) > 0;
          return (
            <Space wrap>
              <Button
                size='small'
                type='tertiary'
                theme='light'
                disabled={!isActive || isCancelled || !hasUsed}
                onClick={() => resetSubscriptionUsage(sub?.id)}
              >
                {t('重置额度')}
              </Button>
              <Button
                size='small'
                type='warning'
                theme='light'
                disabled={!isActive || isCancelled}
                onClick={() => invalidateSubscription(sub?.id)}
              >
                {t('作废')}
              </Button>
              <Button
                size='small'
                type='danger'
                theme='light'
                onClick={() => deleteSubscription(sub?.id)}
              >
                {t('删除')}
              </Button>
            </Space>
          );
        },
      },
    ];
  }, [t, planTitleMap]);

  const bucketColumns = useMemo(() => {
    return [
      {
        title: 'ID',
        key: 'id',
        width: 72,
        render: (_, record) => (
          <Text type='secondary'>#{record.bucket.id}</Text>
        ),
      },
      {
        title: t('限时额度包'),
        key: 'bucket',
        width: 220,
        render: (_, record) => (
          <div className='min-w-0'>
            <div className='font-medium truncate'>
              {record.bucket.title || `${t('限时额度包')} #${record.bucket.id}`}
            </div>
            <div className='text-xs text-[var(--semi-color-text-2)]'>
              {t('创建时间')}: {formatTs(record.bucket.created_at)}
            </div>
          </div>
        ),
      },
      {
        title: t('状态'),
        key: 'status',
        width: 92,
        render: (_, record) => renderBucketStatusTag(record.status, t),
      },
      {
        title: t('使用情况'),
        key: 'usage',
        width: 240,
        render: (_, record) => (
          <div className='min-w-[180px]'>
            <div className='flex justify-between gap-2 text-xs'>
              <span>
                {renderQuota(record.used)} / {renderQuota(record.total)}
              </span>
              <Text type='secondary'>{record.percent.toFixed(0)}%</Text>
            </div>
            <Progress
              percent={record.percent}
              aria-label='quota bucket usage'
              format={() => ''}
              style={{ marginTop: 2, marginBottom: 0 }}
            />
            <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
              {t('剩余')}: {renderQuota(record.remaining)}
            </div>
          </div>
        ),
      },
      {
        title: t('有效期'),
        key: 'validity',
        width: 220,
        render: (_, record) => (
          <div className='text-xs text-gray-600'>
            <div>
              {t('开始')}: {formatTs(record.bucket.start_time)}
            </div>
            <div>
              {t('到期')}: {formatTs(record.bucket.end_time)}
            </div>
          </div>
        ),
      },
      {
        title: t('来源'),
        key: 'source',
        width: 190,
        render: (_, record) => {
          const planId = record.bucket.source_plan_id;
          const planTitle = planTitleMap.get(planId);
          return (
            <div className='text-xs text-[var(--semi-color-text-2)]'>
              <div>
                {sourceLabel(record.bucket.source, t)}
                {record.bucket.source_redemption_id
                  ? ` #${record.bucket.source_redemption_id}`
                  : ''}
              </div>
              <div>
                {t('关联套餐')}: {planTitle || (planId ? `#${planId}` : '-')}
              </div>
            </div>
          );
        },
      },
      {
        title: '',
        key: 'operate',
        width: 100,
        fixed: 'right',
        render: (_, record) => {
          const bucket = record?.bucket || {};
          const isActive = record?.status === 'active';
          return (
            <Button
              size='small'
              type='danger'
              theme='light'
              disabled={!isActive}
              onClick={() => invalidateQuotaBucket(bucket.id)}
            >
              {t('作废')}
            </Button>
          );
        },
      },
    ];
  }, [t, planTitleMap]);

  return (
    <SideSheet
      visible={visible}
      placement='right'
      width={isMobile ? '100%' : 920}
      bodyStyle={{ padding: 0 }}
      onCancel={onCancel}
      title={
        <Space>
          <Tag color='blue' shape='circle'>
            {t('管理')}
          </Tag>
          <Typography.Title heading={4} className='m-0'>
            {t('用户权益管理')}
          </Typography.Title>
          <Text type='tertiary' className='ml-2'>
            {user?.username || '-'} {'(ID:'} {user?.id || '-'}
            {')'}
          </Text>
        </Space>
      }
    >
      <div className='p-4'>
        {/* 顶部操作栏：新增订阅 */}
        <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-3 mb-4'>
          <div className='flex gap-2 flex-1'>
            <Select
              placeholder={t('选择订阅套餐')}
              optionList={planOptions}
              value={selectedPlanId}
              onChange={setSelectedPlanId}
              loading={plansLoading}
              filter
              style={{ minWidth: isMobile ? undefined : 300, flex: 1 }}
            />
            <Button
              type='primary'
              theme='solid'
              icon={<IconPlusCircle />}
              loading={creating}
              onClick={createSubscription}
            >
              {t('新增订阅')}
            </Button>
          </div>
        </div>

        {/* 限时额度包 */}
        <div
          className='mb-4 rounded-xl px-4 py-3'
          style={{
            background: 'var(--semi-color-fill-0)',
            border: '1px solid var(--semi-color-border)',
          }}
        >
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div>
              <Text strong>{t('限时额度包管理')}</Text>
              <div className='text-xs text-[var(--semi-color-text-2)]'>
                {t('查看一周畅用包的额度、使用量、来源和到期时间')}
              </div>
            </div>
            <Tag
              color={quotaBuckets.active_bucket_count ? 'green' : 'grey'}
              shape='circle'
            >
              {t('生效')} {quotaBuckets.active_bucket_count || 0} /{' '}
              {bucketRows.length}
            </Tag>
          </div>

          <div className='mt-3 grid grid-cols-2 md:grid-cols-4 gap-2'>
            <div className='rounded-lg px-3 py-2 bg-[var(--semi-color-bg-1)]'>
              <div className='text-xs text-[var(--semi-color-text-2)]'>
                {t('生效剩余')}
              </div>
              <div className='mt-1 font-semibold'>
                {renderQuota(quotaBuckets.total_remaining || 0)}
              </div>
            </div>
            <div className='rounded-lg px-3 py-2 bg-[var(--semi-color-bg-1)]'>
              <div className='text-xs text-[var(--semi-color-text-2)]'>
                {t('生效用量')}
              </div>
              <div className='mt-1 font-semibold'>
                {renderQuota(quotaBuckets.total_used || 0)} /{' '}
                {renderQuota(quotaBuckets.total_amount || 0)}
              </div>
            </div>
            <div className='rounded-lg px-3 py-2 bg-[var(--semi-color-bg-1)]'>
              <div className='text-xs text-[var(--semi-color-text-2)]'>
                {t('全部累计用量')}
              </div>
              <div className='mt-1 font-semibold'>
                {renderQuota(allBucketStats.used)} /{' '}
                {renderQuota(allBucketStats.total)}
              </div>
            </div>
            <div className='rounded-lg px-3 py-2 bg-[var(--semi-color-bg-1)]'>
              <div className='text-xs text-[var(--semi-color-text-2)]'>
                {t('最近到期')}
              </div>
              <div className='mt-1 font-semibold text-sm'>
                {formatTs(quotaBuckets.nearest_end_time)}
              </div>
            </div>
          </div>

          <div className='mt-3'>
            <CardTable
              columns={bucketColumns}
              dataSource={bucketRows}
              rowKey={(row) => row?.bucket?.id}
              pagination={false}
              hidePagination
              scroll={{ x: 'max-content' }}
              empty={
                <Empty
                  image={
                    <IllustrationNoResult style={{ width: 120, height: 120 }} />
                  }
                  darkModeImage={
                    <IllustrationNoResultDark
                      style={{ width: 120, height: 120 }}
                    />
                  }
                  description={t('暂无限时额度包')}
                  style={{ padding: 20 }}
                />
              }
              size='small'
            />
          </div>
        </div>

        <div className='mb-2 flex items-center justify-between'>
          <Text strong>{t('月卡套餐')}</Text>
          <Text type='tertiary' size='small'>
            {t('保留每日额度、重置和到期信息')}
          </Text>
        </div>
        <CardTable
          columns={columns}
          dataSource={pagedSubs}
          rowKey={(row) => row?.subscription?.id}
          loading={loading}
          scroll={{ x: 'max-content' }}
          hidePagination={false}
          pagination={{
            currentPage,
            pageSize,
            total: subs.length,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: false,
            onPageChange: handlePageChange,
          }}
          empty={
            <Empty
              image={
                <IllustrationNoResult style={{ width: 150, height: 150 }} />
              }
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无订阅记录')}
              style={{ padding: 30 }}
            />
          }
          size='middle'
        />
      </div>
    </SideSheet>
  );
};

export default UserSubscriptionsModal;
