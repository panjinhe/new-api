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
  CreditCard,
  Gauge,
  RefreshCw,
  ShieldCheck,
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
          className='mt-5 rounded-2xl p-4'
          style={{
            background:
              'linear-gradient(180deg, rgba(239, 246, 255, 0.92), rgba(255, 255, 255, 0.96))',
            border: '1px solid rgba(59, 130, 246, 0.18)',
          }}
        >
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div className='min-w-0'>
              <div className='flex flex-wrap items-center gap-2'>
                <Typography.Title
                  heading={5}
                  ellipsis={{ rows: 1, showTooltip: true }}
                  style={{ margin: 0, maxWidth: 320 }}
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
                style={{ display: 'block', maxWidth: 520 }}
              >
                {primaryPlan?.subtitle ||
                  (primaryPlan
                    ? `${formatSubscriptionDuration(primaryPlan, t)} · ${formatSubscriptionResetPeriod(primaryPlan, t)}`
                    : t('历史套餐信息不可用，按订阅快照展示额度'))}
              </Text>
            </div>
            <div className='text-right'>
              <div className='text-2xl font-semibold text-[var(--semi-color-text-0)] leading-none'>
                {getRemainingDays(primarySubscription)}
              </div>
              <Text type='tertiary' size='small'>
                {t('剩余天数')}
              </Text>
            </div>
          </div>

          <div className='mt-4'>
            <div className='mb-2 flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--semi-color-text-2)]'>
              <span>{t('每日额度使用')}</span>
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
                stroke='rgba(37, 99, 235, 1)'
                aria-label='subscription usage'
              />
            ) : null}
          </div>

          <div className='mt-4 flex flex-wrap gap-2'>
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
            <div className='mt-3 text-xs text-[var(--semi-color-text-2)]'>
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
