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
  Badge,
  Button,
  Card,
  Divider,
  Progress,
  Select,
  Skeleton,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  CalendarClock,
  ChevronDown,
  CircleDollarSign,
  CreditCard,
  Gauge,
  PackageCheck,
  RefreshCw,
  ShieldCheck,
  Sparkles,
} from 'lucide-react';
import { renderQuota } from '../../helpers';
import { useActualTheme } from '../../context/Theme';
import {
  formatSubscriptionDuration,
  formatSubscriptionResetPeriod,
} from '../../helpers/subscriptionFormat';

const { Text } = Typography;

const formatDateTime = (timestamp) => {
  const value = Number(timestamp || 0);
  if (value <= 0) return '-';
  return new Date(value * 1000).toLocaleString();
};

const getRemainingDays = (summary) => {
  const endTime = Number(summary?.subscription?.end_time || 0);
  if (endTime <= 0) return 0;
  const remainingSeconds = endTime - Date.now() / 1000;
  return Math.max(0, Math.ceil(remainingSeconds / 86400));
};

const getUsagePercent = (summary) => {
  const total = Number(summary?.subscription?.amount_total || 0);
  const used = Number(summary?.subscription?.amount_used || 0);
  if (total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((used / total) * 100)));
};

const getPlanTitle = (summary, planMap, t) => {
  const subscription = summary?.subscription;
  const plan =
    summary?.plan ||
    (subscription?.plan_id ? planMap.get(subscription.plan_id) : null);
  return plan?.title || `${t('订阅')} #${subscription?.id || '-'}`;
};

const getPlan = (summary, planMap) => {
  const subscription = summary?.subscription;
  return (
    summary?.plan ||
    (subscription?.plan_id ? planMap.get(subscription.plan_id) : null) ||
    null
  );
};

const isSubscriptionActive = (summary) => {
  const subscription = summary?.subscription;
  if (!subscription) return false;
  return (
    subscription.status === 'active' &&
    Number(subscription.end_time || 0) > Date.now() / 1000
  );
};

const getBucketUsagePercent = (summary) => {
  const bucket = summary?.bucket;
  const total = Number(bucket?.amount_total || 0);
  const used = Number(bucket?.amount_used || 0);
  if (total <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((used / total) * 100)));
};

const getBucketRemainingDays = (summary) => {
  const endTime = Number(summary?.bucket?.end_time || 0);
  if (endTime <= 0) return 0;
  return Math.max(0, Math.ceil((endTime - Date.now() / 1000) / 86400));
};

const BucketStatusTag = ({ t, status }) => {
  const color =
    status === 'active' ? 'green' : status === 'expired' ? 'orange' : 'grey';
  const label =
    status === 'active'
      ? t('生效中')
      : status === 'empty'
        ? t('已用完')
        : status === 'expired'
          ? t('已过期')
          : t('已迁移');
  return (
    <Tag color={color} size='small' shape='circle'>
      {label}
    </Tag>
  );
};

const StatusTag = ({ t, summary }) => {
  const subscription = summary?.subscription;
  const active = isSubscriptionActive(summary);
  const cancelled = subscription?.status === 'cancelled';

  if (active) {
    return (
      <Tag
        color='white'
        size='small'
        shape='circle'
        prefixIcon={<Badge dot type='success' />}
      >
        {t('生效中')}
      </Tag>
    );
  }

  return (
    <Tag color='white' size='small' shape='circle'>
      {cancelled ? t('已作废') : t('已过期')}
    </Tag>
  );
};

const subscriptionPlanVisuals = [
  {
    titleKeyword: 'Pro 50x',
    code: 'PRO-50X',
    label: '黑卡旗舰',
    dark: true,
    accent: 'rgba(245, 199, 108, 1)',
    muted: 'rgba(214, 211, 202, 0.82)',
    border: 'rgba(245, 199, 108, 0.34)',
    background:
      'linear-gradient(132deg, rgba(245, 199, 108, 0.16), transparent 26%, rgba(255, 255, 255, 0.055) 54%, transparent 74%), linear-gradient(145deg, rgba(10, 12, 16, 1), rgba(25, 25, 24, 1) 48%, rgba(8, 10, 14, 1))',
    rail: 'linear-gradient(90deg, rgba(245, 199, 108, 0), rgba(245, 199, 108, 0.96), rgba(255, 244, 214, 0.84), rgba(245, 199, 108, 0))',
    glow: '0 22px 50px rgba(2, 6, 23, 0.24), inset 0 1px 0 rgba(255, 255, 255, 0.09), inset 0 -1px 0 rgba(0, 0, 0, 0.36)',
  },
  {
    titleKeyword: '探测',
    code: 'SCAN-01',
    label: '低轨探测',
    accent: 'rgba(8, 145, 178, 1)',
    border: 'rgba(8, 145, 178, 0.32)',
    background:
      'linear-gradient(135deg, rgba(236, 254, 255, 0.98), rgba(255, 255, 255, 1) 56%, rgba(240, 249, 255, 0.92))',
    rail: 'linear-gradient(90deg, rgba(8, 145, 178, 0.96), rgba(34, 211, 238, 0.72))',
    glow: '0 16px 34px rgba(8, 145, 178, 0.10)',
  },
  {
    titleKeyword: '启航',
    code: 'LAUNCH-02',
    label: '启航窗口',
    accent: 'rgba(37, 99, 235, 1)',
    border: 'rgba(37, 99, 235, 0.32)',
    background:
      'linear-gradient(135deg, rgba(239, 246, 255, 0.98), rgba(255, 255, 255, 1) 54%, rgba(238, 242, 255, 0.92))',
    rail: 'linear-gradient(90deg, rgba(37, 99, 235, 0.96), rgba(96, 165, 250, 0.72))',
    glow: '0 16px 34px rgba(37, 99, 235, 0.10)',
  },
  {
    titleKeyword: '巡航',
    code: 'CRUISE-03',
    label: '主力巡航',
    accent: 'rgba(5, 150, 105, 1)',
    border: 'rgba(5, 150, 105, 0.34)',
    background:
      'linear-gradient(135deg, rgba(236, 253, 245, 0.98), rgba(255, 255, 255, 1) 50%, rgba(239, 246, 255, 0.92))',
    rail: 'linear-gradient(90deg, rgba(5, 150, 105, 0.96), rgba(59, 130, 246, 0.72))',
    glow: '0 18px 38px rgba(5, 150, 105, 0.13)',
  },
  {
    titleKeyword: '加速',
    code: 'BOOST-04',
    label: '高能加速',
    accent: 'rgba(217, 119, 6, 1)',
    border: 'rgba(217, 119, 6, 0.34)',
    background:
      'linear-gradient(135deg, rgba(255, 251, 235, 0.98), rgba(255, 255, 255, 1) 52%, rgba(255, 247, 237, 0.92))',
    rail: 'linear-gradient(90deg, rgba(217, 119, 6, 0.96), rgba(251, 191, 36, 0.72))',
    glow: '0 18px 38px rgba(217, 119, 6, 0.12)',
  },
  {
    titleKeyword: '光速跃迁',
    code: 'WARP-05',
    label: '跃迁通道',
    accent: 'rgba(124, 58, 237, 1)',
    border: 'rgba(124, 58, 237, 0.36)',
    background:
      'linear-gradient(135deg, rgba(245, 243, 255, 0.98), rgba(255, 255, 255, 1) 52%, rgba(253, 244, 255, 0.92))',
    rail: 'linear-gradient(90deg, rgba(124, 58, 237, 0.96), rgba(236, 72, 153, 0.66))',
    glow: '0 18px 38px rgba(124, 58, 237, 0.13)',
  },
];

const getSubscriptionPlanVisual = (plan) => {
  const text = `${plan?.title || ''} ${plan?.subtitle || ''}`;
  if (/pro\s*50x|pro50x|黑卡/i.test(text)) {
    return subscriptionPlanVisuals[0];
  }
  return (
    subscriptionPlanVisuals.find((item) => text.includes(item.titleKeyword)) ||
    subscriptionPlanVisuals[2]
  );
};

const isPlanVisualDark = (visual) => visual?.dark || visual?.themeDark;

const getThemeAwarePlanVisual = (visual, isDarkMode) => {
  if (!isDarkMode || visual?.dark) return visual;

  return {
    ...visual,
    themeDark: true,
    muted: 'rgba(203, 213, 225, 0.78)',
    background:
      'linear-gradient(135deg, rgba(15, 23, 42, 0.96), rgba(24, 24, 27, 0.98) 54%, rgba(30, 41, 59, 0.92))',
    glow: '0 18px 42px rgba(0, 0, 0, 0.28), inset 0 1px 0 rgba(255, 255, 255, 0.055)',
  };
};

const getPlanVisualTexture = (visual) =>
  visual?.dark
    ? 'linear-gradient(112deg, transparent 0%, rgba(255, 255, 255, 0.08) 44%, transparent 62%), repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.035) 0 1px, transparent 1px 26px)'
    : visual?.themeDark
      ? 'linear-gradient(112deg, transparent 0%, rgba(255, 255, 255, 0.055) 44%, transparent 62%), repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.028) 0 1px, transparent 1px 26px)'
      : 'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.36) 48%, transparent 68%), repeating-linear-gradient(90deg, rgba(255,255,255,0.22) 0 1px, transparent 1px 24px)';

const getPlanVisualPanel = (visual) =>
  visual?.dark
    ? 'linear-gradient(180deg, rgba(255, 255, 255, 0.105), rgba(255, 255, 255, 0.055))'
    : visual?.themeDark
      ? 'linear-gradient(180deg, rgba(15, 23, 42, 0.72), rgba(30, 41, 59, 0.54))'
      : 'rgba(255,255,255,0.80)';

const getPlanVisualSurface = (visual) =>
  visual?.dark
    ? 'linear-gradient(180deg, rgba(245, 199, 108, 0.16), rgba(255, 255, 255, 0.07))'
    : visual?.themeDark
      ? 'rgba(15, 23, 42, 0.72)'
      : 'rgba(255, 255, 255, 0.68)';

const getPlanVisualText = (visual) =>
  isPlanVisualDark(visual)
    ? 'rgba(248, 250, 252, 0.96)'
    : 'var(--semi-color-text-0)';

const getPlanVisualMutedText = (visual) =>
  isPlanVisualDark(visual)
    ? visual.muted || 'rgba(203, 213, 225, 0.78)'
    : 'var(--semi-color-text-2)';

const getPlanComparisonLabel = (plan) => {
  const text = `${plan?.title || ''} ${plan?.subtitle || ''}`;
  if (/pro\s*50x|pro50x|黑卡/i.test(text)) return '等于 2.5 个 Pro 20x';
  if (text.includes('光速跃迁')) return '约等于 1.5个 Pro 20x';
  if (text.includes('加速')) return '约等于3个 Pro 5x';
  if (text.includes('巡航')) return '约等于1.5个 Pro 5x';
  if (text.includes('启航')) return '约等于 4 个 Plus';
  if (text.includes('探测')) return '约等于 1.4 个 Plus';
  return '';
};

const DetailItem = ({ icon, label, value, tooltip }) => {
  const content = (
    <div
      className='min-w-[140px] flex-1 rounded-xl px-3 py-2'
      style={{
        background: 'var(--semi-color-fill-0)',
        border: '1px solid var(--semi-color-border)',
      }}
    >
      <div className='flex items-center gap-2 text-[12px] text-[var(--semi-color-text-2)]'>
        {icon}
        <span>{label}</span>
      </div>
      <div className='mt-1 text-sm font-semibold text-[var(--semi-color-text-0)] break-words'>
        {value}
      </div>
    </div>
  );

  return tooltip ? <Tooltip content={tooltip}>{content}</Tooltip> : content;
};

const SubscriptionStatusPanel = ({
  t,
  loading = false,
  plans = [],
  billingPreference = 'subscription_first',
  onChangeBillingPreference,
  quotaBuckets = {},
  activeSubscriptions = [],
  reloadSubscriptionSelf,
}) => {
  const actualTheme = useActualTheme();
  const [refreshing, setRefreshing] = useState(false);
  const [animatedPercent, setAnimatedPercent] = useState(0);
  const [bucketDetailsOpen, setBucketDetailsOpen] = useState(false);
  const isDarkMode = actualTheme === 'dark';

  const planMap = useMemo(() => {
    const map = new Map();
    (plans || []).forEach((item) => {
      const plan = item?.plan;
      if (plan?.id) {
        map.set(plan.id, plan);
      }
    });
    return map;
  }, [plans]);

  const activeList = useMemo(
    () => (activeSubscriptions || []).filter((item) => item?.subscription),
    [activeSubscriptions],
  );
  const primarySubscription = activeList[0] || null;
  const primaryPlan = getPlan(primarySubscription, planMap);
  const hasActiveSubscription = activeList.length > 0;
  const activeBucketList = useMemo(
    () => quotaBuckets?.active_buckets || [],
    [quotaBuckets],
  );
  const allBucketList = useMemo(
    () => quotaBuckets?.buckets || [],
    [quotaBuckets],
  );
  const hasActiveBucket = activeBucketList.length > 0;
  const hasAnyBucket = allBucketList.length > 0;
  const hasActiveEntitlement = hasActiveBucket || hasActiveSubscription;
  const disableSubscriptionPreference = !hasActiveEntitlement;
  const isSubscriptionPreference =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only';
  const displayBillingPreference =
    disableSubscriptionPreference && isSubscriptionPreference
      ? 'wallet_first'
      : billingPreference || 'wallet_first';
  const subscriptionPreferenceLabel =
    billingPreference === 'subscription_only' ? t('仅用权益') : t('优先权益');
  const usagePercent = getUsagePercent(primarySubscription);
  const primarySubscriptionRecord = primarySubscription?.subscription;
  const primaryTotalAmount = Number(
    primarySubscriptionRecord?.amount_total || 0,
  );
  const primaryUsedAmount = Number(primarySubscriptionRecord?.amount_used || 0);
  const primaryRemainAmount = Math.max(
    0,
    primaryTotalAmount - primaryUsedAmount,
  );
  const hasLimitedPrimaryQuota = primaryTotalAmount > 0;
  const primaryVisual = getThemeAwarePlanVisual(
    getSubscriptionPlanVisual(primaryPlan),
    isDarkMode,
  );
  const primaryVisualDark = isPlanVisualDark(primaryVisual);
  const primaryComparisonLabel = getPlanComparisonLabel(primaryPlan);
  const primaryDailyQuota = Number(primaryPlan?.total_amount || 0);
  const primaryResetLabel = primaryPlan
    ? formatSubscriptionResetPeriod(primaryPlan, t)
    : t('按订阅快照');

  useEffect(() => {
    setAnimatedPercent(0);
    const timer = window.setTimeout(() => {
      setAnimatedPercent(usagePercent);
    }, 80);
    return () => window.clearTimeout(timer);
  }, [primarySubscription?.subscription?.id, usagePercent]);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await reloadSubscriptionSelf?.();
    } finally {
      setRefreshing(false);
    }
  };

  const renderPreferenceSelect = () => (
    <Select
      value={displayBillingPreference}
      onChange={onChangeBillingPreference}
      size='small'
      style={{ minWidth: 128 }}
      optionList={[
        {
          value: 'subscription_first',
          label: disableSubscriptionPreference
            ? `${t('优先权益')} (${t('无生效')})`
            : t('优先权益'),
          disabled: disableSubscriptionPreference,
        },
        { value: 'wallet_first', label: t('优先钱包') },
        {
          value: 'subscription_only',
          label: disableSubscriptionPreference
            ? `${t('仅用权益')} (${t('无生效')})`
            : t('仅用权益'),
          disabled: disableSubscriptionPreference,
        },
        { value: 'wallet_only', label: t('仅用钱包') },
      ]}
    />
  );

  const renderBucketSection = () => {
    if (!hasActiveBucket) return null;

    const totalRemaining = Number(quotaBuckets?.total_remaining || 0);
    const totalAmount = Number(quotaBuckets?.total_amount || 0);
    const totalUsed = Number(quotaBuckets?.total_used || 0);
    const nearestEndTime = Number(quotaBuckets?.nearest_end_time || 0);
    const percent =
      totalAmount > 0
        ? Math.min(
            100,
            Math.max(0, Math.round((totalUsed / totalAmount) * 100)),
          )
        : 0;

    return (
      <div
        className='mt-4 rounded-2xl px-4 py-4'
        style={{
          border: '1px solid var(--semi-color-border)',
          background: 'var(--semi-color-fill-0)',
        }}
      >
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0'>
            <div className='flex flex-wrap items-center gap-2'>
              <PackageCheck size={17} color='rgba(5, 150, 105, 1)' />
              <Text strong>{t('一周畅用包')}</Text>
              <Tag
                color={hasActiveBucket ? 'green' : 'grey'}
                shape='circle'
                size='small'
              >
                {hasActiveBucket
                  ? `${activeBucketList.length} ${t('个生效中')}`
                  : t('无生效')}
              </Tag>
            </div>
            <Text type='tertiary' size='small'>
              {t('限时额度包独立展示，默认优先消耗最早到期的包')}
            </Text>
          </div>
          <Button
            size='small'
            theme='light'
            type='tertiary'
            icon={
              <ChevronDown
                size={14}
                className={`transition-transform ${bucketDetailsOpen ? 'rotate-180' : ''}`}
              />
            }
            onClick={() => setBucketDetailsOpen((open) => !open)}
            disabled={!hasAnyBucket}
          >
            {bucketDetailsOpen ? t('收起明细') : t('展开明细')}
          </Button>
        </div>

        {hasActiveBucket ? (
          <>
            <div
              className='mt-4 grid grid-cols-1 gap-3 rounded-xl px-3 py-3 sm:grid-cols-3'
              style={{
                background: 'var(--semi-color-bg-1)',
                border: '1px solid var(--semi-color-border)',
              }}
            >
              <div className='min-w-0'>
                <div className='flex items-center gap-1.5 text-xs text-[var(--semi-color-text-2)]'>
                  <Gauge size={13} />
                  {t('总剩余额度')}
                </div>
                <div className='mt-1 text-lg font-semibold leading-tight text-[var(--semi-color-text-0)]'>
                  {renderQuota(totalRemaining)}
                </div>
              </div>
              <div className='min-w-0'>
                <div className='flex items-center gap-1.5 text-xs text-[var(--semi-color-text-2)]'>
                  <CircleDollarSign size={13} />
                  {t('已用/总量')}
                </div>
                <div className='mt-1 text-sm font-medium leading-tight text-[var(--semi-color-text-0)]'>
                  {renderQuota(totalUsed)} / {renderQuota(totalAmount)}
                </div>
              </div>
              <div className='min-w-0'>
                <div className='flex items-center gap-1.5 text-xs text-[var(--semi-color-text-2)]'>
                  <CalendarClock size={13} />
                  {t('最近到期')}
                </div>
                <div className='mt-1 text-sm font-medium leading-tight text-[var(--semi-color-text-0)]'>
                  {formatDateTime(nearestEndTime)}
                </div>
              </div>
            </div>
            {totalAmount > 0 ? (
              <Progress
                className='mt-3'
                percent={percent}
                showInfo={false}
                stroke='rgba(5, 150, 105, 1)'
                aria-label='quota bucket usage'
              />
            ) : null}
          </>
        ) : (
          <div className='mt-4 text-sm text-[var(--semi-color-text-2)]'>
            {hasAnyBucket
              ? t('历史限时额度包已过期或用完')
              : t('兑换一周畅用包后，这里会展示每个码的额度和到期时间')}
          </div>
        )}

        {bucketDetailsOpen && hasAnyBucket ? (
          <div
            className='mt-4 overflow-hidden rounded-xl'
            style={{
              border: '1px solid var(--semi-color-border)',
              background: 'var(--semi-color-bg-1)',
            }}
          >
            {allBucketList.map((summary) => {
              const bucket = summary.bucket || {};
              const remaining = Number(summary.remaining_quota || 0);
              const used = Number(bucket.amount_used || 0);
              const total = Number(bucket.amount_total || 0);
              const percent = getBucketUsagePercent(summary);
              return (
                <div
                  key={bucket.id}
                  className='grid grid-cols-1 gap-3 px-3 py-3 sm:grid-cols-[minmax(0,1.2fr)_minmax(160px,0.8fr)_auto] sm:items-center'
                  style={{
                    borderBottom:
                      bucket.id ===
                      allBucketList[allBucketList.length - 1]?.bucket?.id
                        ? 'none'
                        : '1px solid var(--semi-color-border)',
                  }}
                >
                  <div className='min-w-0'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <Text strong ellipsis={{ rows: 1, showTooltip: true }}>
                        {bucket.title || t('一周畅用包')}
                      </Text>
                      <BucketStatusTag t={t} status={summary.status} />
                    </div>
                    <div className='mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[var(--semi-color-text-2)]'>
                      <span>
                        {t('兑换')} {formatDateTime(bucket.start_time)}
                      </span>
                      <span>
                        {t('到期')} {formatDateTime(bucket.end_time)}
                      </span>
                      <span>
                        {t('剩余')} {getBucketRemainingDays(summary)} {t('天')}
                      </span>
                    </div>
                  </div>
                  <div className='min-w-0'>
                    <div className='mb-1 flex items-center justify-between gap-2 text-xs text-[var(--semi-color-text-2)]'>
                      <span>{t('使用进度')}</span>
                      <span>
                        {renderQuota(used)} / {renderQuota(total)}
                      </span>
                    </div>
                    {total > 0 ? (
                      <Progress
                        percent={percent}
                        showInfo={false}
                        stroke='rgba(5, 150, 105, 1)'
                        aria-label='bucket item usage'
                      />
                    ) : null}
                  </div>
                  <div className='sm:text-right'>
                    <div className='text-lg font-semibold leading-tight text-[var(--semi-color-text-0)]'>
                      {renderQuota(remaining)}
                    </div>
                    <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                      {t('剩余额度')}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : null}
      </div>
    );
  };

  if (!loading && !hasActiveEntitlement) {
    return null;
  }

  return (
    <Card className='!rounded-2xl shadow-sm h-full' bodyStyle={{ padding: 20 }}>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='flex items-center gap-2'>
            <ShieldCheck size={18} color='var(--semi-color-primary)' />
            <Text strong>{t('权益中心')}</Text>
            {hasActiveEntitlement ? (
              <Tag
                color='white'
                size='small'
                shape='circle'
                prefixIcon={<Badge dot type='success' />}
              >
                {activeBucketList.length + activeList.length} {t('项生效中')}
              </Tag>
            ) : (
              <Tag color='white' size='small' shape='circle'>
                {t('无生效')}
              </Tag>
            )}
          </div>
          <Text type='tertiary' size='small'>
            {t('查看一周畅用包、月卡套餐和扣费顺序')}
          </Text>
        </div>
        <div className='flex flex-wrap items-center justify-end gap-2'>
          {renderPreferenceSelect()}
          <Button
            size='small'
            theme='light'
            type='tertiary'
            icon={
              <RefreshCw
                size={14}
                className={refreshing ? 'animate-spin' : ''}
              />
            }
            onClick={handleRefresh}
            loading={refreshing}
          />
        </div>
      </div>

      {disableSubscriptionPreference && isSubscriptionPreference && (
        <div className='mt-3 text-xs text-[var(--semi-color-text-2)]'>
          {t('已保存偏好为')}
          {subscriptionPreferenceLabel}
          {t('，当前无生效权益，将自动使用钱包')}
        </div>
      )}

      {loading ? (
        <div className='mt-5 space-y-3'>
          <Skeleton.Title active style={{ width: '45%', height: 24 }} />
          <Skeleton.Paragraph active rows={3} />
        </div>
      ) : (
        <>
          {renderBucketSection()}
          {hasActiveBucket && hasActiveSubscription ? (
            <Divider margin={16} />
          ) : null}
          {hasActiveSubscription ? (
            <>
              <div className='flex flex-wrap items-center gap-2'>
                <Sparkles size={17} color='var(--semi-color-primary)' />
                <Text strong>{t('月卡套餐')}</Text>
                <Tag
                  color={hasActiveSubscription ? 'green' : 'grey'}
                  shape='circle'
                  size='small'
                >
                  {hasActiveSubscription
                    ? `${activeList.length} ${t('个生效中')}`
                    : t('无生效')}
                </Tag>
              </div>
              <Text type='tertiary' size='small'>
                {t('月卡保留每日额度、重置规则和到期时间')}
              </Text>
              <div
                className='mt-5 relative overflow-hidden rounded-2xl p-4 transition-all duration-200'
                style={{
                  background: primaryVisual.background,
                  border: `1px solid ${primaryVisual.border}`,
                  boxShadow: primaryVisual.glow,
                }}
              >
                <div
                  className='absolute inset-x-0 top-0 h-1.5'
                  style={{ background: primaryVisual.rail }}
                />
                <div
                  className='pointer-events-none absolute inset-0'
                  style={{
                    backgroundImage: getPlanVisualTexture(primaryVisual),
                    opacity: primaryVisualDark ? 0.72 : 0.58,
                  }}
                />

                <div className='relative flex flex-wrap items-start justify-between gap-3'>
                  <div className='min-w-0'>
                    <div className='flex flex-wrap items-center gap-2 mb-3'>
                      <span
                        className='rounded-full px-2.5 py-1 text-xs font-semibold'
                        style={{
                          background: getPlanVisualSurface(primaryVisual),
                          border: `1px solid ${primaryVisual.border}`,
                          color: primaryVisual.accent,
                        }}
                      >
                        {primaryVisual.code}
                      </span>
                      <span
                        className='inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium'
                        style={{
                          background: primaryVisualDark
                            ? 'rgba(245, 199, 108, 0.14)'
                            : 'rgba(255, 255, 255, 0.64)',
                          border: `1px solid ${primaryVisual.border}`,
                          color: primaryVisualDark
                            ? 'rgba(255, 244, 214, 0.94)'
                            : primaryVisual.accent,
                        }}
                      >
                        <Sparkles size={12} className='mr-1' />
                        {t(primaryVisual.label)}
                      </span>
                    </div>
                    <div className='flex flex-wrap items-center gap-2'>
                      <Typography.Title
                        heading={5}
                        ellipsis={{ rows: 1, showTooltip: true }}
                        style={{
                          margin: 0,
                          maxWidth: 360,
                          color: getPlanVisualText(primaryVisual),
                        }}
                      >
                        {getPlanTitle(primarySubscription, planMap, t)}
                      </Typography.Title>
                      <StatusTag t={t} summary={primarySubscription} />
                      {!primaryPlan && (
                        <Tag color='grey' size='small'>
                          {t('套餐已删除')}
                        </Tag>
                      )}
                    </div>
                    <Text
                      type='tertiary'
                      size='small'
                      ellipsis={{ rows: 1, showTooltip: true }}
                      style={{
                        display: 'block',
                        maxWidth: 520,
                        color: getPlanVisualMutedText(primaryVisual),
                      }}
                    >
                      {primaryPlan?.subtitle ||
                        (primaryPlan
                          ? `${formatSubscriptionDuration(primaryPlan, t)} · ${formatSubscriptionResetPeriod(primaryPlan, t)}`
                          : t('历史套餐信息不可用，按订阅快照展示额度'))}
                    </Text>
                  </div>
                  <div className='text-right'>
                    <div
                      className='text-3xl font-semibold leading-none'
                      style={{ color: primaryVisual.accent }}
                    >
                      {getRemainingDays(primarySubscription)}
                    </div>
                    <Text
                      type='tertiary'
                      size='small'
                      style={{ color: getPlanVisualMutedText(primaryVisual) }}
                    >
                      {t('剩余天数')}
                    </Text>
                  </div>
                </div>

                {primaryComparisonLabel && (
                  <div
                    className='relative mt-4 flex items-center gap-2 rounded-xl px-3 py-2'
                    style={{
                      background: getPlanVisualSurface(primaryVisual),
                      border: `1px solid ${primaryVisual.border}`,
                      color: primaryVisualDark
                        ? 'rgba(255, 244, 214, 0.98)'
                        : primaryVisual.accent,
                    }}
                  >
                    <CircleDollarSign size={17} className='shrink-0' />
                    <span className='min-w-0 break-words text-base font-semibold leading-snug'>
                      {t(primaryComparisonLabel)}
                    </span>
                  </div>
                )}

                <div className='relative mt-4'>
                  <div
                    className='mb-2 flex flex-wrap items-center justify-between gap-2 text-xs'
                    style={{ color: getPlanVisualMutedText(primaryVisual) }}
                  >
                    <span>{t('权益进度')}</span>
                    {hasLimitedPrimaryQuota ? (
                      <Tooltip
                        content={`${t('原生额度')}：${primaryUsedAmount}/${primaryTotalAmount}`}
                      >
                        <span>
                          {renderQuota(primaryRemainAmount)} /{' '}
                          {renderQuota(primaryTotalAmount)}
                        </span>
                      </Tooltip>
                    ) : (
                      <span>{t('不限')}</span>
                    )}
                  </div>
                  {hasLimitedPrimaryQuota ? (
                    <Progress
                      percent={animatedPercent}
                      showInfo={false}
                      stroke={primaryVisual.accent}
                      aria-label='subscription usage'
                    />
                  ) : null}
                </div>

                <div className='relative mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4'>
                  <div
                    className='rounded-xl px-3 py-2'
                    style={{ background: getPlanVisualPanel(primaryVisual) }}
                  >
                    <div
                      className='flex items-center gap-1.5 text-xs'
                      style={{ color: getPlanVisualMutedText(primaryVisual) }}
                    >
                      <Gauge size={13} />
                      {t('每日额度')}
                    </div>
                    <div
                      className='mt-1 break-words text-base font-semibold leading-tight'
                      style={{ color: getPlanVisualText(primaryVisual) }}
                    >
                      {primaryDailyQuota > 0
                        ? renderQuota(primaryDailyQuota)
                        : t('不限')}
                    </div>
                  </div>
                  <div
                    className='rounded-xl px-3 py-2'
                    style={{ background: getPlanVisualPanel(primaryVisual) }}
                  >
                    <div
                      className='flex items-center gap-1.5 text-xs'
                      style={{ color: getPlanVisualMutedText(primaryVisual) }}
                    >
                      <RefreshCw size={13} />
                      {t('重置')}
                    </div>
                    <div
                      className='mt-1 break-words text-sm font-semibold leading-tight'
                      style={{ color: getPlanVisualText(primaryVisual) }}
                    >
                      {primaryResetLabel}
                    </div>
                  </div>
                  <DetailItem
                    icon={<CalendarClock size={14} />}
                    label={t('下次重置')}
                    value={formatDateTime(
                      primarySubscription.subscription.next_reset_time,
                    )}
                  />
                  <DetailItem
                    icon={<CreditCard size={14} />}
                    label={t('到期时间')}
                    value={formatDateTime(
                      primarySubscription.subscription.end_time,
                    )}
                  />
                </div>

                <div className='relative mt-3 flex flex-wrap gap-2'>
                  <DetailItem
                    icon={<Gauge size={14} />}
                    label={t('扣费偏好')}
                    value={
                      displayBillingPreference === 'subscription_first'
                        ? t('优先权益')
                        : displayBillingPreference === 'subscription_only'
                          ? t('仅用权益')
                          : displayBillingPreference === 'wallet_only'
                            ? t('仅用钱包')
                            : t('优先钱包')
                    }
                  />
                </div>

                {activeList.length > 1 && (
                  <div
                    className='relative mt-3 text-xs'
                    style={{ color: getPlanVisualMutedText(primaryVisual) }}
                  >
                    {t('还有')} {activeList.length - 1}{' '}
                    {t('个月卡，可在最近订阅中查看')}
                  </div>
                )}
              </div>
            </>
          ) : null}
        </>
      )}
    </Card>
  );
};

export default SubscriptionStatusPanel;
