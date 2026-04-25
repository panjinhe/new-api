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
import { renderQuota } from '../../helpers';
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
  { quota: '50 刀', quotaValue: 50, price: '10 元', priceValue: 10 },
  { quota: '100 刀', quotaValue: 100, price: '18 元', priceValue: 18 },
  { quota: '200 刀', quotaValue: 200, price: '35 元', priceValue: 35 },
  { quota: '500 刀', quotaValue: 500, price: '80 元', priceValue: 80 },
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
  subscriptionPlans = [],
  subscriptionPlansLoading = false,
}) => {
  const { t } = useTranslation();
  const openOfficialSite = () => {
    window.open(OFFICIAL_SITE, '_blank', 'noopener,noreferrer');
  };
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
              <Tag color='orange'>{t('0.16元每刀')}</Tag>
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
              {t('新客首单额外赠送 20 刀额度，购买兑换码可私聊客服。')}
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
    <div className='w-full grid grid-cols-1 xl:grid-cols-[0.9fr_1.1fr] gap-4 items-stretch'>
      <Card
        className='!rounded-2xl shadow-sm h-full'
        bodyStyle={{ padding: 0 }}
      >
        <div className='h-full p-5 flex flex-col'>
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
            <Button
              theme='outline'
              type='tertiary'
              icon={<ExternalLink size={14} />}
              onClick={openOfficialSite}
            >
              {t('官网')}
            </Button>
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
            {normalizedPricingPlans.map((item) => (
              <div
                key={item.quota}
                className='rounded-2xl px-4 py-3 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md'
                style={{
                  background: item.isBestValue
                    ? 'linear-gradient(180deg, rgba(236, 253, 245, 0.95), rgba(255, 255, 255, 1))'
                    : 'var(--semi-color-fill-0)',
                  border: item.isBestValue
                    ? '1px solid rgba(16, 185, 129, 0.35)'
                    : '1px solid var(--semi-color-border)',
                }}
              >
                <div className='flex items-center justify-between gap-2'>
                  <Text strong>{item.quota}</Text>
                  <Tag color={item.isBestValue ? 'green' : 'grey'} size='small'>
                    {t(item.badge)}
                  </Tag>
                </div>
                <div className='mt-3 flex items-end justify-between gap-2'>
                  <div>
                    <div className='text-2xl font-semibold leading-none text-[var(--semi-color-text-0)]'>
                      {item.price}
                    </div>
                    <div className='mt-2 text-xs text-[var(--semi-color-text-2)]'>
                      {item.unitPrice.toFixed(2)} {t('元 / 刀')}
                    </div>
                  </div>
                  {item.saveAmount > 0 && (
                    <div className='rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700'>
                      {t('省')} {item.saveAmount.toFixed(0)} {t('元')}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>

          <div className='mt-5 rounded-2xl bg-[var(--semi-color-fill-0)] px-4 py-3'>
            <div className='flex items-center gap-2 text-sm font-medium text-[var(--semi-color-text-0)]'>
              <Gift size={16} className='text-rose-500' />
              {t('兑换码与接入支持')}
            </div>
            <div className='mt-2 text-xs leading-6 text-[var(--semi-color-text-2)]'>
              {t('主推')} {featuredPricingPlan.quota} · {t('单价低至')}{' '}
              {featuredPricingPlan.unitPrice.toFixed(2)} {t('元 / 刀')} ·{' '}
              {t('官网同额')} {featuredPricingPlan.officialPriceLabel}
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

      <Card
        className='!rounded-2xl shadow-sm h-full'
        bodyStyle={{ padding: 0 }}
      >
        <div className='h-full p-5 flex flex-col'>
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
                {subscriptionPlanItems.map((plan) => {
                  const displayPrice = getPlanPriceLabel(plan);
                  const totalAmount = Number(plan?.total_amount || 0);
                  const limit = Number(plan?.max_purchase_per_user || 0);
                  const isRecommended =
                    (plan?.title || '').trim() === '前进三：巡航';

                  return (
                    <div
                      key={plan?.id || plan?.title}
                      className='rounded-2xl px-4 py-4 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md'
                      style={{
                        background: isRecommended
                          ? 'linear-gradient(135deg, rgba(239, 246, 255, 0.98), rgba(255, 255, 255, 1) 58%, rgba(236, 253, 245, 0.92))'
                          : 'var(--semi-color-fill-0)',
                        border: isRecommended
                          ? '1px solid rgba(37, 99, 235, 0.35)'
                          : '1px solid var(--semi-color-border)',
                      }}
                    >
                      <div className='flex items-start justify-between gap-3'>
                        <div className='min-w-0'>
                          <div className='flex items-center gap-2 flex-wrap'>
                            <Text
                              strong
                              ellipsis={{ showTooltip: true }}
                              style={{
                                display: 'block',
                                color: isRecommended
                                  ? 'var(--semi-color-primary)'
                                  : 'var(--semi-color-text-0)',
                              }}
                            >
                              {plan?.title || t('订阅套餐')}
                            </Text>
                            {isRecommended && (
                              <Tag color='blue' shape='circle' size='small'>
                                {t('推荐')}
                              </Tag>
                            )}
                          </div>
                          {plan?.subtitle && (
                            <div className='mt-1 text-xs leading-5 text-[var(--semi-color-text-2)]'>
                              {plan.subtitle}
                            </div>
                          )}
                        </div>
                        <div className='shrink-0 text-right'>
                          <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                            {t('月价')}
                          </div>
                          <div
                            className='mt-1 text-2xl font-semibold leading-none'
                            style={{
                              color: isRecommended
                                ? 'var(--semi-color-primary)'
                                : 'var(--semi-color-text-0)',
                            }}
                          >
                            {displayPrice}
                          </div>
                        </div>
                      </div>

                      <div className='mt-4 grid grid-cols-2 gap-2'>
                        <div className='rounded-xl bg-white/80 px-3 py-2'>
                          <div className='flex items-center gap-1 text-[11px] text-[var(--semi-color-text-2)]'>
                            <Gauge size={12} />
                            {t('每日额度')}
                          </div>
                          <div className='mt-1 text-sm font-semibold text-[var(--semi-color-text-0)]'>
                            {totalAmount > 0
                              ? renderQuota(totalAmount)
                              : t('不限')}
                          </div>
                        </div>
                        <div className='rounded-xl bg-white/80 px-3 py-2'>
                          <div className='flex items-center gap-1 text-[11px] text-[var(--semi-color-text-2)]'>
                            <RefreshCw size={12} />
                            {t('重置')}
                          </div>
                          <div className='mt-1 text-sm font-semibold text-[var(--semi-color-text-0)]'>
                            {getPlanResetLabel(plan, t)}
                          </div>
                        </div>
                      </div>

                      <div className='mt-3 flex flex-wrap gap-2 text-xs text-[var(--semi-color-text-2)]'>
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
              <Paragraph
                copyable={{ content: QQ_GROUP }}
                className='!mb-0 !mt-0'
              >
                <span className='text-sm font-medium text-sky-800'>
                  {t('Q群：')}
                  {QQ_GROUP}
                </span>
              </Paragraph>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default RechargeSupportCard;
