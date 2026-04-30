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
  CircleDollarSign,
  CreditCard,
  Gauge,
  RefreshCw,
  ShieldCheck,
  Sparkles,
} from 'lucide-react';
import { renderQuota } from '../../helpers';
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

const getPlanVisualTexture = (visual) =>
  visual?.dark
    ? 'linear-gradient(112deg, transparent 0%, rgba(255, 255, 255, 0.08) 44%, transparent 62%), repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.035) 0 1px, transparent 1px 26px)'
    : 'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.36) 48%, transparent 68%), repeating-linear-gradient(90deg, rgba(255,255,255,0.22) 0 1px, transparent 1px 24px)';

const getPlanVisualPanel = (visual) =>
  visual?.dark
    ? 'linear-gradient(180deg, rgba(255, 255, 255, 0.105), rgba(255, 255, 255, 0.055))'
    : 'rgba(255,255,255,0.80)';

const getPlanVisualSurface = (visual) =>
  visual?.dark
    ? 'linear-gradient(180deg, rgba(245, 199, 108, 0.16), rgba(255, 255, 255, 0.07))'
    : 'rgba(255, 255, 255, 0.68)';

const getPlanVisualText = (visual) =>
  visual?.dark ? 'rgba(255, 251, 235, 0.96)' : 'var(--semi-color-text-0)';

const getPlanVisualMutedText = (visual) =>
  visual?.dark ? visual.muted : 'var(--semi-color-text-2)';

const getPlanComparisonLabel = (plan) => {
  const text = `${plan?.title || ''} ${plan?.subtitle || ''}`;
  if (/pro\s*50x|pro50x|黑卡/i.test(text)) return '等于 2.5 个 Pro 20x';
  if (text.includes('光速跃迁')) return '约等于 1.5个 Pro 20x';
  if (text.includes('加速')) return '约等于3个 pro 5x';
  if (text.includes('巡航')) return '约等于1.5个 pro5x';
  if (text.includes('启航')) return '约等于 4 个 Plus 账号';
  if (text.includes('探测')) return '约等于 1.4 个 Plus 账号';
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
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  catalogEnabled = false,
  onViewCatalog,
}) => {
  const [refreshing, setRefreshing] = useState(false);
  const [animatedPercent, setAnimatedPercent] = useState(0);

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
  const allList = useMemo(
    () => (allSubscriptions || []).filter((item) => item?.subscription),
    [allSubscriptions],
  );
  const primarySubscription = activeList[0] || null;
  const primaryPlan = getPlan(primarySubscription, planMap);
  const hasActiveSubscription = activeList.length > 0;
  const hasAnySubscription = allList.length > 0;
  const disableSubscriptionPreference = !hasActiveSubscription;
  const isSubscriptionPreference =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only';
  const displayBillingPreference =
    disableSubscriptionPreference && isSubscriptionPreference
      ? 'wallet_first'
      : billingPreference || 'wallet_first';
  const subscriptionPreferenceLabel =
    billingPreference === 'subscription_only' ? t('仅用订阅') : t('优先订阅');
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
  const primaryVisual = getSubscriptionPlanVisual(primaryPlan);
  const primaryVisualDark = Boolean(primaryVisual?.dark);
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
            ? `${t('优先订阅')} (${t('无生效')})`
            : t('优先订阅'),
          disabled: disableSubscriptionPreference,
        },
        { value: 'wallet_first', label: t('优先钱包') },
        {
          value: 'subscription_only',
          label: disableSubscriptionPreference
            ? `${t('仅用订阅')} (${t('无生效')})`
            : t('仅用订阅'),
          disabled: disableSubscriptionPreference,
        },
        { value: 'wallet_only', label: t('仅用钱包') },
      ]}
    />
  );

  const renderHistoryRows = () => {
    const historyRows = allList
      .filter(
        (item) =>
          item?.subscription?.id !== primarySubscription?.subscription?.id,
      )
      .slice(0, 3);

    if (historyRows.length === 0) return null;

    return (
      <>
        <Divider margin={12} />
        <div className='flex items-center justify-between gap-2 mb-2'>
          <Text strong size='small'>
            {t('最近订阅')}
          </Text>
          <Tag color='white' size='small' shape='circle'>
            {allList.length} {t('条记录')}
          </Tag>
        </div>
        <div className='space-y-2'>
          {historyRows.map((summary) => {
            const subscription = summary.subscription;
            const title = getPlanTitle(summary, planMap, t);
            return (
              <div
                key={subscription.id}
                className='flex flex-wrap items-center justify-between gap-2 rounded-lg px-3 py-2'
                style={{ background: 'var(--semi-color-fill-0)' }}
              >
                <div className='min-w-0 flex items-center gap-2'>
                  <Text
                    strong
                    ellipsis={{ rows: 1, showTooltip: true }}
                    style={{ maxWidth: 220 }}
                  >
                    {title}
                  </Text>
                  <StatusTag t={t} summary={summary} />
                </div>
                <Text type='tertiary' size='small'>
                  {formatDateTime(subscription.end_time)}
                </Text>
              </div>
            );
          })}
        </div>
      </>
    );
  };

  return (
    <Card className='!rounded-2xl shadow-sm h-full' bodyStyle={{ padding: 20 }}>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='flex items-center gap-2'>
            <ShieldCheck size={18} color='var(--semi-color-primary)' />
            <Text strong>{t('我的订阅')}</Text>
            {hasActiveSubscription ? (
              <Tag
                color='white'
                size='small'
                shape='circle'
                prefixIcon={<Badge dot type='success' />}
              >
                {activeList.length} {t('个生效中')}
              </Tag>
            ) : (
              <Tag color='white' size='small' shape='circle'>
                {t('无生效')}
              </Tag>
            )}
          </div>
          <Text type='tertiary' size='small'>
            {t('查看当前月卡、额度重置和扣费顺序')}
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
          {t('，当前无生效订阅，将自动使用钱包')}
        </div>
      )}

      {loading ? (
        <div className='mt-5 space-y-3'>
          <Skeleton.Title active style={{ width: '45%', height: 24 }} />
          <Skeleton.Paragraph active rows={3} />
        </div>
      ) : hasActiveSubscription ? (
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
              value={formatDateTime(primarySubscription.subscription.end_time)}
            />
          </div>

          <div className='relative mt-3 flex flex-wrap gap-2'>
            <DetailItem
              icon={<Gauge size={14} />}
              label={t('扣费偏好')}
              value={
                displayBillingPreference === 'subscription_first'
                  ? t('优先订阅')
                  : displayBillingPreference === 'subscription_only'
                    ? t('仅用订阅')
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
              {t('个生效订阅，可在最近订阅中查看')}
            </div>
          )}
        </div>
      ) : (
        <div
          className='mt-5 rounded-2xl px-4 py-5'
          style={{
            border: '1px dashed var(--semi-color-border)',
            background: 'var(--semi-color-fill-0)',
          }}
        >
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <div>
              <Text strong>{t('暂无生效订阅')}</Text>
              <div className='mt-1 text-sm text-[var(--semi-color-text-2)]'>
                {hasAnySubscription
                  ? t('历史订阅已过期或作废，当前请求将按钱包偏好扣费')
                  : t('购买月卡后，这里会展示每日额度、重置时间和到期时间')}
              </div>
            </div>
            {catalogEnabled && (
              <Button theme='light' type='primary' onClick={onViewCatalog}>
                {t('查看月卡')}
              </Button>
            )}
          </div>
        </div>
      )}

      {!loading && renderHistoryRows()}
    </Card>
  );
};

export default SubscriptionStatusPanel;
