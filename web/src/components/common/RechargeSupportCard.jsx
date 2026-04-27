import React from 'react';
import {
  Button,
  Card,
  Skeleton,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  CalendarClock,
  CheckCircle2,
  CircleDollarSign,
  ExternalLink,
  Gift,
  Gauge,
  MessageCircle,
  RefreshCw,
  Rocket,
  Sparkles,
  Wallet,
} from 'lucide-react';
import { openRechargeLink, renderQuota } from '../../helpers';
import {
  formatSubscriptionDuration,
  formatSubscriptionResetPeriod,
} from '../../helpers/subscriptionFormat';

const { Text, Paragraph } = Typography;

const QQ_GROUP = '217637139';
const OFFICIAL_SITE = 'https://pbroe.com/';
const FALLBACK_OFFICIAL_USD_RATE = 7.3;

const toolTags = ['Codex', 'CLI', 'VSCode', 'OpenClaw', '小龙虾', 'AstrBot'];

const modelTags = ['5.4', '5.3codex', '5.4mini', '5.2'];

const pricingItems = [
  { quota: '50 刀', quotaValue: 50, price: '12 元', priceValue: 12 },
  { quota: '100 刀', quotaValue: 100, price: '22 元', priceValue: 22 },
  { quota: '200 刀', quotaValue: 200, price: '42 元', priceValue: 42 },
  { quota: '500 刀', quotaValue: 500, price: '99 元', priceValue: 99 },
];
const starterUnitPrice =
  pricingItems[0].priceValue / pricingItems[0].quotaValue;

const getOfficialUsdRate = () => {
  if (typeof window === 'undefined') return FALLBACK_OFFICIAL_USD_RATE;

  try {
    const rawStatus = window.localStorage.getItem('status');
    if (!rawStatus) return FALLBACK_OFFICIAL_USD_RATE;

    const status = JSON.parse(rawStatus);
    return Number(status?.usd_exchange_rate) || FALLBACK_OFFICIAL_USD_RATE;
  } catch (error) {
    return FALLBACK_OFFICIAL_USD_RATE;
  }
};

const formatOfficialPrice = (value) => {
  if (!Number.isFinite(value)) return '约 0 元';
  return `约 ${value.toLocaleString('zh-CN', {
    maximumFractionDigits: 0,
  })} 元`;
};

const buildPricingPlans = (officialUsdRate) =>
  pricingItems.map((item, index) => {
    const unitPrice = item.priceValue / item.quotaValue;
    const equivalentStarterPrice = item.quotaValue * starterUnitPrice;
    const saveAmount = Math.max(equivalentStarterPrice - item.priceValue, 0);
    const officialPriceValue = item.quotaValue * officialUsdRate;
    const officialSaveValue = Math.max(officialPriceValue - item.priceValue, 0);

    return {
      ...item,
      unitPrice,
      saveAmount,
      officialPriceValue,
      officialPriceLabel: formatOfficialPrice(officialPriceValue),
      officialSaveValue,
      badge:
        index === 0
          ? '试用'
          : index === pricingItems.length - 1
            ? '主推'
            : index === pricingItems.length - 2
              ? '高频'
              : '常用',
    };
  });

const normalizePlan = (planWrapper) => planWrapper?.plan || planWrapper || null;

const rechargePackVisuals = [
  {
    quota: '50 刀',
    code: 'CELL-50',
    label: '点火测试',
    accent: 'rgba(8, 145, 178, 1)',
    border: 'rgba(8, 145, 178, 0.30)',
    background:
      'linear-gradient(135deg, rgba(236, 254, 255, 0.98), rgba(255, 255, 255, 1) 56%, rgba(240, 249, 255, 0.9))',
    rail: 'linear-gradient(90deg, rgba(8, 145, 178, 0.96), rgba(34, 211, 238, 0.72))',
    glow: '0 14px 30px rgba(8, 145, 178, 0.10)',
    level: '38%',
    usage: '低成本验证',
  },
  {
    quota: '100 刀',
    code: 'DRIVE-100',
    label: '日常推进',
    accent: 'rgba(37, 99, 235, 1)',
    border: 'rgba(37, 99, 235, 0.30)',
    background:
      'linear-gradient(135deg, rgba(239, 246, 255, 0.98), rgba(255, 255, 255, 1) 55%, rgba(238, 242, 255, 0.9))',
    rail: 'linear-gradient(90deg, rgba(37, 99, 235, 0.96), rgba(96, 165, 250, 0.72))',
    glow: '0 14px 30px rgba(37, 99, 235, 0.10)',
    level: '52%',
    usage: '日常 Coding',
  },
  {
    quota: '200 刀',
    code: 'ORBIT-200',
    label: '高频航段',
    accent: 'rgba(217, 119, 6, 1)',
    border: 'rgba(217, 119, 6, 0.30)',
    background:
      'linear-gradient(135deg, rgba(255, 251, 235, 0.98), rgba(255, 255, 255, 1) 55%, rgba(255, 247, 237, 0.9))',
    rail: 'linear-gradient(90deg, rgba(217, 119, 6, 0.96), rgba(251, 191, 36, 0.72))',
    glow: '0 14px 30px rgba(217, 119, 6, 0.10)',
    level: '70%',
    usage: '高频自动化',
  },
  {
    quota: '500 刀',
    code: 'CORE-500',
    label: '主推燃料舱',
    accent: 'rgba(5, 150, 105, 1)',
    border: 'rgba(5, 150, 105, 0.34)',
    background:
      'linear-gradient(135deg, rgba(236, 253, 245, 0.98), rgba(255, 255, 255, 1) 50%, rgba(239, 246, 255, 0.9))',
    rail: 'linear-gradient(90deg, rgba(5, 150, 105, 0.96), rgba(59, 130, 246, 0.72))',
    glow: '0 18px 38px rgba(5, 150, 105, 0.13)',
    level: '100%',
    usage: '长程主力',
  },
];

const getRechargePackVisual = (item, index) =>
  rechargePackVisuals.find((visual) => visual.quota === item.quota) ||
  rechargePackVisuals[index % rechargePackVisuals.length];

const subscriptionPlanVisuals = [
  {
    titleKeyword: 'Pro 50x',
    code: 'PRO-50X',
    label: '黑卡旗舰',
    dark: true,
    accent: 'rgba(248, 250, 252, 1)',
    muted: 'rgba(203, 213, 225, 1)',
    border: 'rgba(148, 163, 184, 0.34)',
    background:
      'linear-gradient(135deg, rgba(2, 6, 23, 0.98), rgba(15, 23, 42, 1) 54%, rgba(39, 39, 42, 0.96))',
    rail: 'linear-gradient(90deg, rgba(248, 250, 252, 0.96), rgba(148, 163, 184, 0.76))',
    glow: '0 22px 48px rgba(2, 6, 23, 0.24)',
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

const getSubscriptionPlanVisual = (plan, index) => {
  const title = plan?.title || '';
  return (
    subscriptionPlanVisuals.find((item) => title.includes(item.titleKeyword)) ||
    subscriptionPlanVisuals[index % subscriptionPlanVisuals.length]
  );
};

const getPlanVisualSurface = (visual, fallback = 'rgba(255, 255, 255, 0.72)') =>
  visual?.dark ? 'rgba(255, 255, 255, 0.12)' : fallback;

const getPlanVisualPanel = (visual) =>
  visual?.dark ? 'rgba(15, 23, 42, 0.74)' : 'rgba(255,255,255,0.80)';

const getPlanVisualText = (visual) =>
  visual?.dark ? 'rgba(248, 250, 252, 1)' : 'var(--semi-color-text-0)';

const getPlanVisualMutedText = (visual) =>
  visual?.dark ? 'rgba(203, 213, 225, 1)' : 'var(--semi-color-text-2)';

const getPlanPriceLabel = (plan) => {
  const price = Number(plan?.price_amount || 0);
  const displayPrice = price.toFixed(Number.isInteger(price) ? 0 : 2);
  return `${displayPrice} 元`;
};

const getPlanResetLabel = (plan, t) => {
  const resetText = formatSubscriptionResetPeriod(plan, t);
  if (plan?.quota_reset_period === 'daily') return t('每日重置');
  if (resetText === t('不重置')) return resetText;
  return `${resetText}${t('重置')}`;
};

const RechargeSupportCard = ({
  compact = false,
  onGoTopup,
  topUpLink = '',
  subscriptionPlans = [],
  subscriptionPlansLoading = false,
}) => {
  const { t } = useTranslation();
  const openOfficialSite = () => {
    window.open(OFFICIAL_SITE, '_blank', 'noopener,noreferrer');
  };
  const handleOpenRechargeLink = () => openRechargeLink(topUpLink);
  const officialUsdRate = getOfficialUsdRate();
  const pricingPlans = buildPricingPlans(officialUsdRate);
  const bestUnitPrice = Math.min(...pricingPlans.map((item) => item.unitPrice));
  const normalizedPricingPlans = pricingPlans.map((item) => ({
    ...item,
    isBestValue: item.unitPrice === bestUnitPrice,
  }));
  const featuredPricingPlan =
    normalizedPricingPlans.find((item) => item.isBestValue) ||
    normalizedPricingPlans[normalizedPricingPlans.length - 1];
  const subscriptionPlanItems = subscriptionPlans
    .map(normalizePlan)
    .filter(Boolean);

  if (compact) {
    return (
      <Card
        className='!rounded-2xl border-0 shadow-sm'
        bodyStyle={{ padding: 16 }}
      >
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0'>
            <div className='flex items-center gap-2 flex-wrap'>
              <Rocket size={18} className='text-amber-500' />
              <Text strong style={{ fontSize: 16 }}>
                {t('Codex API 接入服务')}
              </Text>
              <Tag color='orange'>{t('0.198元每刀起')}</Tag>
              <Tag color='green'>{t('超值倍率')}</Tag>
            </div>
            <div className='mt-2 text-sm text-[var(--semi-color-text-1)]'>
              {t(
                '平台当前主打 Codex 系列，适合日常 Coding、脚本、自动化、插件和 Bot 调用。',
              )}
            </div>
          </div>
        </div>

        <div className='mt-4 flex flex-wrap gap-2'>
          {toolTags.map((item) => (
            <Tag key={item} color='blue' size='large'>
              {item}
            </Tag>
          ))}
        </div>

        <div className='mt-3 flex flex-wrap gap-2'>
          {modelTags.map((item) => (
            <Tag key={item} color='cyan'>
              {item}
            </Tag>
          ))}
        </div>

        <div className='mt-4 space-y-3'>
          <Paragraph className='!mb-0'>
            {t(
              '余额用完可直接联系 QQ 群获取兑换码和接入帮助，少走弯路，拿到就能配。',
            )}
          </Paragraph>
          <div className='rounded-xl bg-[var(--semi-color-fill-0)] px-4 py-3'>
            <div className='flex items-center gap-2 text-sm font-medium'>
              <MessageCircle size={16} className='text-sky-500' />
              {t('Q群：')}
              {QQ_GROUP}
            </div>
            <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
              {t('兑换码和月卡都可通过淘宝购买，也可私聊客服确认。')}
            </div>
          </div>
          <Space wrap>
            <Button
              theme='solid'
              type='primary'
              icon={<Wallet size={14} />}
              onClick={onGoTopup}
            >
              {t('去钱包管理')}
            </Button>
            <Button
              theme='solid'
              type='warning'
              icon={<ExternalLink size={14} />}
              onClick={handleOpenRechargeLink}
            >
              {t('淘宝购买')}
            </Button>
            <Button
              theme='outline'
              type='tertiary'
              icon={<ExternalLink size={14} />}
              onClick={openOfficialSite}
            >
              {t('打开官网')}
            </Button>
          </Space>
        </div>
      </Card>
    );
  }

  return (
    <div className='w-full grid grid-cols-1 xl:grid-cols-[0.9fr_1.1fr] gap-4 items-start'>
      <Card className='!rounded-2xl shadow-sm' bodyStyle={{ padding: 0 }}>
        <div className='p-5 flex flex-col'>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div className='min-w-0'>
              <div className='flex items-center gap-2 flex-wrap'>
                <CircleDollarSign size={18} color='rgba(5, 150, 105, 1)' />
                <Text strong style={{ fontSize: 16 }}>
                  {t('Codex API 接入服务')}
                </Text>
                <Tag color='green'>{t('按量付费')}</Tag>
              </div>
              <div className='mt-2 text-sm leading-6 text-[var(--semi-color-text-2)]'>
                {t('按量扣费，使用钱包余额；余额不足时用兑换码补充。')}
              </div>
            </div>
            <Space wrap>
              <Button
                theme='solid'
                type='warning'
                icon={<ExternalLink size={14} />}
                onClick={handleOpenRechargeLink}
              >
                {t('淘宝购买')}
              </Button>
              <Button
                theme='outline'
                type='tertiary'
                icon={<ExternalLink size={14} />}
                onClick={openOfficialSite}
              >
                {t('官网')}
              </Button>
            </Space>
          </div>

          <div className='mt-4 flex flex-wrap gap-2'>
            {toolTags.map((item) => (
              <Tag key={item} color='blue' size='large'>
                {item}
              </Tag>
            ))}
          </div>

          <div className='mt-3 flex flex-wrap gap-2'>
            {modelTags.map((item) => (
              <Tag key={item} color='cyan'>
                {item}
              </Tag>
            ))}
          </div>

          <div className='mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2'>
            {normalizedPricingPlans.map((item, index) => {
              const visual = getRechargePackVisual(item, index);

              return (
                <div
                  key={item.quota}
                  className='group relative overflow-hidden rounded-2xl px-4 py-4 transition-all duration-200 hover:-translate-y-0.5'
                  style={{
                    background: visual.background,
                    border: `1px solid ${visual.border}`,
                    boxShadow: visual.glow,
                  }}
                >
                  <div
                    className='absolute inset-x-0 top-0 h-1.5'
                    style={{ background: visual.rail }}
                  />
                  <div
                    className='pointer-events-none absolute inset-0 opacity-60'
                    style={{
                      backgroundImage:
                        'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.36) 48%, transparent 68%), repeating-linear-gradient(90deg, rgba(255,255,255,0.24) 0 1px, transparent 1px 22px)',
                    }}
                  />
                  <div className='relative flex items-start justify-between gap-3'>
                    <div className='min-w-0'>
                      <div className='flex items-center gap-2 flex-wrap'>
                        <span
                          className='rounded-full px-2 py-0.5 text-[10px] font-semibold'
                          style={{
                            background: 'rgba(255,255,255,0.72)',
                            border: `1px solid ${visual.border}`,
                            color: visual.accent,
                          }}
                        >
                          {visual.code}
                        </span>
                        <Text
                          strong
                          style={{ display: 'block', color: visual.accent }}
                        >
                          {item.quota}
                        </Text>
                      </div>
                      <div
                        className='mt-2 inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium'
                        style={{
                          background: 'rgba(255, 255, 255, 0.66)',
                          color: visual.accent,
                        }}
                      >
                        <Sparkles size={12} />
                        {t(visual.label)}
                      </div>
                    </div>
                    <span
                      className='shrink-0 rounded-full px-2.5 py-1 text-xs font-medium'
                      style={{
                        background: item.isBestValue
                          ? 'rgba(220, 252, 231, 0.86)'
                          : 'rgba(255, 255, 255, 0.76)',
                        border: `1px solid ${visual.border}`,
                        color: visual.accent,
                      }}
                    >
                      {t(item.badge)}
                    </span>
                  </div>

                  <div className='relative mt-5 flex items-end justify-between gap-3'>
                    <div>
                      <div
                        className='text-3xl font-semibold leading-none'
                        style={{ color: visual.accent }}
                      >
                        {item.price}
                      </div>
                      <div className='mt-2 text-xs text-[var(--semi-color-text-2)]'>
                        {item.unitPrice.toFixed(2)} {t('元 / 刀')}
                      </div>
                    </div>
                    <div className='text-right'>
                      <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                        {t('性价比')}
                      </div>
                      <div
                        className='mt-2 h-1.5 w-20 overflow-hidden rounded-full bg-white/80'
                        style={{ border: `1px solid ${visual.border}` }}
                      >
                        <div
                          className='h-full rounded-full'
                          style={{
                            width: visual.level,
                            background: visual.rail,
                          }}
                        />
                      </div>
                      <div className='mt-2 text-xs font-medium text-[var(--semi-color-text-1)]'>
                        {item.saveAmount > 0
                          ? `${t('省')} ${item.saveAmount.toFixed(0)} ${t('元')}`
                          : t('点火入口')}
                      </div>
                    </div>
                  </div>

                  <div className='relative mt-4 rounded-xl bg-white/75 px-3 py-2'>
                    <div className='flex items-center justify-between gap-3 text-xs'>
                      <span className='truncate text-[var(--semi-color-text-2)]'>
                        {t(visual.usage)}
                      </span>
                      <span
                        className='shrink-0 font-semibold'
                        style={{ color: visual.accent }}
                      >
                        {t('钱包扣费')}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div
            className='mt-5 relative overflow-hidden rounded-2xl px-4 py-3'
            style={{
              background:
                'linear-gradient(135deg, rgba(240, 253, 250, 0.96), rgba(255, 255, 255, 1) 60%, rgba(239, 246, 255, 0.9))',
              border: '1px solid rgba(5, 150, 105, 0.18)',
            }}
          >
            <div
              className='absolute inset-y-0 left-0 w-1'
              style={{
                background:
                  'linear-gradient(180deg, rgba(5, 150, 105, 0.96), rgba(59, 130, 246, 0.72))',
              }}
            />
            <div className='relative flex items-center gap-2 text-sm font-medium text-[var(--semi-color-text-0)]'>
              <Gift size={16} className='text-emerald-600' />
              {t('兑换码补给站')}
            </div>
            <div className='relative mt-2 text-xs leading-6 text-[var(--semi-color-text-2)]'>
              {t('主推')} {featuredPricingPlan.quota} · {t('单价低至')}{' '}
              {featuredPricingPlan.unitPrice.toFixed(2)} {t('元 / 刀')} ·{' '}
              {t('官网同额')} {featuredPricingPlan.officialPriceLabel}
            </div>
            <div className='relative mt-3'>
              <Button
                size='small'
                theme='solid'
                type='warning'
                icon={<ExternalLink size={14} />}
                onClick={handleOpenRechargeLink}
              >
                {t('购买兑换码')}
              </Button>
            </div>
          </div>

          <div className='mt-auto pt-4 flex flex-wrap items-center justify-between gap-3'>
            <div className='flex items-center gap-2 text-sm text-sky-700'>
              <MessageCircle size={16} />
              <span>
                {t('Q群：')}
                {QQ_GROUP}
              </span>
            </div>
            <Paragraph copyable={{ content: QQ_GROUP }} className='!mb-0 !mt-0'>
              {t('复制群号')}
            </Paragraph>
          </div>
        </div>
      </Card>

      <Card className='!rounded-2xl shadow-sm' bodyStyle={{ padding: 0 }}>
        <div className='p-5 flex flex-col'>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div className='min-w-0'>
              <div className='flex items-center gap-2 flex-wrap'>
                <Sparkles size={18} color='var(--semi-color-primary)' />
                <Text strong style={{ fontSize: 16 }}>
                  {t('月卡套餐')}
                </Text>
                <Tag color='blue'>{t('每日额度')}</Tag>
              </div>
              <div className='mt-2 text-sm leading-6 text-[var(--semi-color-text-2)]'>
                {t('适合持续 Coding、自动化任务和团队项目，按天重置额度。')}
              </div>
            </div>
            <Tag color='blue'>{t('加群开通')}</Tag>
          </div>

          {subscriptionPlansLoading ? (
            <div className='mt-5 space-y-4'>
              <Skeleton.Title active style={{ width: '45%', height: 26 }} />
              <Skeleton.Paragraph active rows={4} />
              <Skeleton.Paragraph active rows={3} />
            </div>
          ) : subscriptionPlanItems.length > 0 ? (
            <>
              <div className='mt-5 grid grid-cols-1 gap-3 lg:grid-cols-2'>
                {subscriptionPlanItems.map((plan, index) => {
                  const displayPrice = getPlanPriceLabel(plan);
                  const totalAmount = Number(plan?.total_amount || 0);
                  const limit = Number(plan?.max_purchase_per_user || 0);
                  const isRecommended =
                    (plan?.title || '').trim() === '前进三：巡航';
                  const visual = getSubscriptionPlanVisual(plan, index);
                  const isPremium = Boolean(visual?.dark);

                  return (
                    <div
                      key={plan?.id || plan?.title}
                      className='group relative overflow-hidden rounded-2xl px-4 py-4 transition-all duration-200 hover:-translate-y-0.5'
                      style={{
                        background: visual.background,
                        border: `1px solid ${visual.border}`,
                        boxShadow: visual.glow,
                      }}
                    >
                      <div
                        className='absolute inset-x-0 top-0 h-1.5'
                        style={{ background: visual.rail }}
                      />
                      <div
                        className='pointer-events-none absolute inset-0 opacity-60'
                        style={{
                          backgroundImage:
                            'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.36) 48%, transparent 68%), repeating-linear-gradient(90deg, rgba(255,255,255,0.26) 0 1px, transparent 1px 24px)',
                        }}
                      />
                      <div className='relative flex items-start justify-between gap-3'>
                        <div className='min-w-0'>
                          <div className='flex items-center gap-2 flex-wrap'>
                            <span
                              className='rounded-full px-2 py-0.5 text-[10px] font-semibold'
                              style={{
                                background: getPlanVisualSurface(visual),
                                border: `1px solid ${visual.border}`,
                                color: visual.accent,
                              }}
                            >
                              {visual.code}
                            </span>
                            <Text
                              strong
                              ellipsis={{ showTooltip: true }}
                              style={{
                                display: 'block',
                                color: visual.accent,
                              }}
                            >
                              {plan?.title || t('订阅套餐')}
                            </Text>
                            {(isRecommended || isPremium) && (
                              <Tag
                                color={isPremium ? 'grey' : 'blue'}
                                shape='circle'
                                size='small'
                              >
                                {t(isPremium ? '旗舰' : '推荐')}
                              </Tag>
                            )}
                          </div>
                          {plan?.subtitle && (
                            <div
                              className='mt-1 text-xs leading-5'
                              style={{ color: getPlanVisualMutedText(visual) }}
                            >
                              {plan.subtitle}
                            </div>
                          )}
                          <div
                            className='mt-2 inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium'
                            style={{
                              background: getPlanVisualSurface(
                                visual,
                                'rgba(255, 255, 255, 0.66)',
                              ),
                              color: visual.accent,
                            }}
                          >
                            <Sparkles size={12} />
                            {t(visual.label)}
                          </div>
                        </div>
                        <div className='shrink-0 text-right'>
                          <div
                            className='text-[11px]'
                            style={{ color: getPlanVisualMutedText(visual) }}
                          >
                            {t('月价')}
                          </div>
                          <div
                            className='mt-1 text-2xl font-semibold leading-none'
                            style={{
                              color: visual.accent,
                            }}
                          >
                            {displayPrice}
                          </div>
                        </div>
                      </div>

                      <div className='relative mt-4 grid grid-cols-2 gap-2'>
                        <div
                          className='rounded-xl px-3 py-2'
                          style={{ background: getPlanVisualPanel(visual) }}
                        >
                          <div
                            className='flex items-center gap-1 text-[11px]'
                            style={{ color: getPlanVisualMutedText(visual) }}
                          >
                            <Gauge size={12} />
                            {t('每日额度')}
                          </div>
                          <div
                            className='mt-1 text-sm font-semibold'
                            style={{ color: getPlanVisualText(visual) }}
                          >
                            {totalAmount > 0
                              ? renderQuota(totalAmount)
                              : t('不限')}
                          </div>
                        </div>
                        <div
                          className='rounded-xl px-3 py-2'
                          style={{ background: getPlanVisualPanel(visual) }}
                        >
                          <div
                            className='flex items-center gap-1 text-[11px]'
                            style={{ color: getPlanVisualMutedText(visual) }}
                          >
                            <RefreshCw size={12} />
                            {t('重置')}
                          </div>
                          <div
                            className='mt-1 text-sm font-semibold'
                            style={{ color: getPlanVisualText(visual) }}
                          >
                            {getPlanResetLabel(plan, t)}
                          </div>
                        </div>
                      </div>

                      <div
                        className='relative mt-3 flex flex-wrap items-center justify-between gap-2 text-xs'
                        style={{ color: getPlanVisualMutedText(visual) }}
                      >
                        <div className='flex flex-wrap gap-2'>
                          <span className='inline-flex items-center gap-1'>
                            <CalendarClock size={12} />
                            {formatSubscriptionDuration(plan, t)}
                          </span>
                          <span className='inline-flex items-center gap-1'>
                            <CheckCircle2 size={12} />
                            {limit > 0
                              ? `${t('每档')} ${limit} ${t('次')}`
                              : t('不限购')}
                          </span>
                        </div>
                        <span
                          className='h-1.5 w-16 rounded-full'
                          style={{ background: visual.rail }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className='mt-auto pt-5'>
                <div className='text-xs leading-6 text-[var(--semi-color-text-2)]'>
                  {t('月卡介绍读取真实套餐数据；开通请加入 Q 群确认套餐档位。')}
                </div>
              </div>
            </>
          ) : (
            <div className='mt-5 rounded-2xl bg-[var(--semi-color-fill-0)] px-5 py-8 text-center'>
              <div className='text-base font-semibold text-[var(--semi-color-text-0)]'>
                {t('暂无可购买月卡')}
              </div>
              <div className='mt-2 text-sm text-[var(--semi-color-text-2)]'>
                {t('当前未返回套餐数据，可加入 Q 群确认开放状态。')}
              </div>
            </div>
          )}

          <div className='mt-4 rounded-2xl border border-sky-200 bg-sky-50 px-4 py-3'>
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div>
                <div className='flex items-center gap-2 text-sm font-medium text-sky-800'>
                  <MessageCircle size={16} />
                  {t('加群开通月卡')}
                </div>
                <div className='mt-1 text-xs text-sky-700'>
                  {t('进群发送套餐名，客服按对应月卡处理。')}
                </div>
              </div>
              <Space wrap>
                <Button
                  theme='solid'
                  type='warning'
                  icon={<ExternalLink size={14} />}
                  onClick={handleOpenRechargeLink}
                >
                  {t('淘宝购买月卡')}
                </Button>
                <Paragraph
                  copyable={{ content: QQ_GROUP }}
                  className='!mb-0 !mt-0'
                >
                  <span className='text-sm font-medium text-sky-800'>
                    {t('Q群：')}
                    {QQ_GROUP}
                  </span>
                </Paragraph>
              </Space>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default RechargeSupportCard;
