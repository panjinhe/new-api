import React from 'react';
import { Button, Card, Space, Tag, Typography } from '@douyinfe/semi-ui';
import {
  CircleDollarSign,
  ExternalLink,
  Gift,
  MessageCircle,
  Rocket,
  Wallet,
} from 'lucide-react';

const { Text, Paragraph } = Typography;

const QQ_GROUP = '217637139';
const OFFICIAL_SITE = 'https://pbroe.com/';
const FALLBACK_OFFICIAL_USD_RATE = 7.3;

const toolTags = [
  'Codex',
  'CLI',
  'VSCode',
  'OpenClaw',
  '小龙虾',
  'AstrBot',
];

const modelTags = ['5.4', '5.3codex', '5.4mini', '5.2'];

const pricingItems = [
  { quota: '50 刀', quotaValue: 50, price: '10 元', priceValue: 10 },
  { quota: '100 刀', quotaValue: 100, price: '18 元', priceValue: 18 },
  { quota: '200 刀', quotaValue: 200, price: '35 元', priceValue: 35 },
  { quota: '500 刀', quotaValue: 500, price: '80 元', priceValue: 80 },
];
const starterUnitPrice = pricingItems[0].priceValue / pricingItems[0].quotaValue;

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
    const savingsRate =
      equivalentStarterPrice > 0
        ? Math.round((saveAmount / equivalentStarterPrice) * 100)
        : 0;

    return {
      ...item,
      unitPrice,
      saveAmount,
      savingsRate,
      officialPriceValue,
      officialPriceLabel: formatOfficialPrice(officialPriceValue),
      officialSaveValue,
      summary:
        index === 0
          ? '适合先体验，低门槛上手'
          : index === 1
            ? '适合日常补充，成本更优'
            : index === pricingItems.length - 2
              ? '适合高频使用，省得更明显'
              : '适合长期稳定调用，单价最低',
      badge:
        index === 0
          ? '入门试用'
          : index === pricingItems.length - 1
            ? '超值推荐'
            : index === pricingItems.length - 2
              ? '进阶优选'
              : '常用档位',
    };
  });

const RechargeSupportCard = ({ compact = false, onGoTopup }) => {
  const openOfficialSite = () => {
    window.open(OFFICIAL_SITE, '_blank', 'noopener,noreferrer');
  };
  const officialUsdRate = getOfficialUsdRate();
  const pricingPlans = buildPricingPlans(officialUsdRate);
  const bestUnitPrice = Math.min(
    ...pricingPlans.map((item) => item.unitPrice),
  );
  const normalizedPricingPlans = pricingPlans.map((item) => ({
    ...item,
    isBestValue: item.unitPrice === bestUnitPrice,
  }));
  const featuredPricingPlan =
    normalizedPricingPlans.find((item) => item.isBestValue) ||
    normalizedPricingPlans[normalizedPricingPlans.length - 1];
  const secondaryPricingPlans = normalizedPricingPlans.filter(
    (item) => item.quota !== featuredPricingPlan.quota,
  );

  return (
    <Card
      className='!rounded-2xl border-0 shadow-sm'
      bodyStyle={{ padding: compact ? 16 : 20 }}
    >
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='flex items-center gap-2 flex-wrap'>
            <Rocket size={18} className='text-amber-500' />
            <Text strong style={{ fontSize: 16 }}>
              Codex API 接入服务
            </Text>
            <Tag color='orange'>0.16元每刀</Tag>
            <Tag color='green'>超值倍率</Tag>
          </div>
          <div className='mt-2 text-sm text-[var(--semi-color-text-1)]'>
            平台当前主打 Codex 系列，适合日常 Coding、脚本、自动化、插件和
            Bot 调用。
          </div>
        </div>
        {!compact && (
          <Button
            theme='solid'
            type='primary'
            icon={<ExternalLink size={14} />}
            onClick={openOfficialSite}
          >
            官网
          </Button>
        )}
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

      {compact ? (
        <div className='mt-4 space-y-3'>
          <Paragraph className='!mb-0'>
            余额用完可直接联系 QQ 群获取兑换码和接入帮助，少走弯路，拿到就能配。
          </Paragraph>
          <div className='rounded-xl bg-[var(--semi-color-fill-0)] px-4 py-3'>
            <div className='flex items-center gap-2 text-sm font-medium'>
              <MessageCircle size={16} className='text-sky-500' />
              Q群：{QQ_GROUP}
            </div>
            <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
              新客首单额外赠送 20 刀额度，购买兑换码可私聊客服。
            </div>
          </div>
          <Space wrap>
            <Button
              theme='solid'
              type='primary'
              icon={<Wallet size={14} />}
              onClick={onGoTopup}
            >
              去钱包管理
            </Button>
            <Button
              theme='outline'
              type='tertiary'
              icon={<ExternalLink size={14} />}
              onClick={openOfficialSite}
            >
              打开官网
            </Button>
          </Space>
        </div>
      ) : (
        <>
          <div className='mt-4 flex justify-center'>
            <div
              className='w-full max-w-4xl overflow-hidden rounded-2xl px-4 py-4'
              style={{
                background:
                  'linear-gradient(180deg, rgba(236, 253, 245, 0.92), rgba(255, 255, 255, 1))',
                border: '1px solid rgba(16, 185, 129, 0.16)',
                boxShadow: '0 20px 45px rgba(16, 185, 129, 0.08)',
              }}
            >
              <div className='flex flex-wrap items-center justify-between gap-3'>
                <div>
                  <div className='flex items-center gap-2 text-sm font-medium'>
                    <CircleDollarSign size={16} className='text-emerald-500' />
                    定价
                  </div>
                  <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                    用得越多单价越低，主推档位更适合高频 Coding 与自动化调用。
                  </div>
                </div>
                <div className='flex flex-wrap gap-2'>
                  <Tag color='green' size='large'>
                    主推 {featuredPricingPlan.quota} 低至{' '}
                    {featuredPricingPlan.unitPrice.toFixed(2)} 元 / 刀
                  </Tag>
                  <Tag color='blue'>档位越高越省</Tag>
                </div>
              </div>

              <div className='mt-4 grid grid-cols-1 gap-4 xl:grid-cols-5'>
                <div
                  className='xl:col-span-3 overflow-hidden rounded-[28px] p-5'
                  style={{
                    background:
                      'linear-gradient(135deg, rgba(255, 247, 237, 0.98), rgba(236, 253, 245, 0.98) 58%, rgba(239, 246, 255, 0.96))',
                    border: '1px solid rgba(251, 191, 36, 0.22)',
                    boxShadow: '0 22px 48px rgba(251, 191, 36, 0.12)',
                  }}
                >
                  <div className='flex flex-wrap items-start justify-between gap-3'>
                    <div>
                      <Tag color='orange' size='large'>
                        主推套餐
                      </Tag>
                      <div className='mt-3 text-xs uppercase tracking-[0.24em] text-amber-700'>
                        Best Value Pack
                      </div>
                    </div>
                    <div className='rounded-full bg-white/80 px-3 py-1 text-xs font-medium text-emerald-700'>
                      比 50 刀累充省 {featuredPricingPlan.saveAmount.toFixed(0)} 元
                    </div>
                  </div>

                  <div className='mt-5 flex flex-wrap items-end gap-x-4 gap-y-2'>
                    <div className='text-5xl font-black tracking-tight text-[var(--semi-color-text-0)]'>
                      {featuredPricingPlan.quota}
                    </div>
                    <div className='pb-2'>
                      <div className='text-lg font-semibold text-[var(--semi-color-text-1)]'>
                        实付 {featuredPricingPlan.price}
                      </div>
                      <div className='mt-1 text-sm text-[var(--semi-color-text-2)] line-through'>
                        官网价 {featuredPricingPlan.officialPriceLabel}
                      </div>
                    </div>
                  </div>

                  <div className='mt-3 max-w-xl text-sm leading-6 text-[var(--semi-color-text-1)]'>
                    高频写代码、跑脚本、做自动化和 Bot 调用，直接上{' '}
                    {featuredPricingPlan.quota}
                    档位更省心，单价已经压到当前主推区最低。
                  </div>

                  <div className='mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3'>
                    <div className='rounded-2xl bg-white/85 px-4 py-3 shadow-sm'>
                      <div className='text-[11px] uppercase tracking-[0.18em] text-[var(--semi-color-text-2)]'>
                        单刀成本
                      </div>
                      <div className='mt-2 text-2xl font-bold text-[var(--semi-color-text-0)]'>
                        {featuredPricingPlan.unitPrice.toFixed(2)} 元
                      </div>
                    </div>
                    <div className='rounded-2xl bg-white/85 px-4 py-3 shadow-sm'>
                      <div className='text-[11px] uppercase tracking-[0.18em] text-[var(--semi-color-text-2)]'>
                        相对入门档
                      </div>
                      <div className='mt-2 text-2xl font-bold text-emerald-700'>
                        省 {featuredPricingPlan.saveAmount.toFixed(0)} 元
                      </div>
                    </div>
                    <div className='rounded-2xl bg-white/85 px-4 py-3 shadow-sm'>
                      <div className='text-[11px] uppercase tracking-[0.18em] text-[var(--semi-color-text-2)]'>
                        官网同额
                      </div>
                      <div className='mt-2 text-base font-semibold text-[var(--semi-color-text-0)]'>
                        {featuredPricingPlan.officialPriceLabel}
                      </div>
                    </div>
                  </div>

                  <div
                    className='mt-5 rounded-2xl px-4 py-3 text-sm'
                    style={{
                      background: 'rgba(255, 255, 255, 0.72)',
                      border: '1px solid rgba(251, 191, 36, 0.16)',
                    }}
                  >
                    <span className='font-semibold text-amber-700'>建议：</span>
                    如果你已经确定会持续使用 Codex，优先选{' '}
                    {featuredPricingPlan.quota}，价格观感和实际性价比都会更稳。
                  </div>
                </div>

                <div className='xl:col-span-2 grid grid-cols-1 gap-3'>
                  {secondaryPricingPlans.map((item) => (
                    <div
                      key={item.quota}
                      className='rounded-2xl p-4 transition-all duration-200 hover:-translate-y-1 hover:shadow-lg'
                      style={{
                        background: 'rgba(255, 255, 255, 0.92)',
                        border: '1px solid rgba(148, 163, 184, 0.18)',
                      }}
                    >
                      <div className='flex items-center justify-between gap-3'>
                        <div>
                          <div className='text-xs font-medium text-[var(--semi-color-text-2)]'>
                            {item.badge}
                          </div>
                          <div className='mt-1 text-2xl font-bold text-[var(--semi-color-text-0)]'>
                            {item.quota}
                          </div>
                        </div>
                        <Tag color={item.saveAmount > 0 ? 'green' : 'grey'}>
                          {item.saveAmount > 0
                            ? `省 ${item.saveAmount.toFixed(0)} 元`
                            : '先试用'}
                        </Tag>
                      </div>

                      <div className='mt-4 flex items-end justify-between gap-3'>
                        <div>
                          <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                            实付
                          </div>
                          <div className='mt-1 text-3xl font-bold leading-none text-[var(--semi-color-text-0)]'>
                            {item.price}
                          </div>
                          <div className='mt-2 text-xs text-[var(--semi-color-text-2)] line-through'>
                            官网价 {item.officialPriceLabel}
                          </div>
                        </div>
                        <div className='rounded-xl bg-slate-50 px-3 py-2 text-right'>
                          <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                            单价
                          </div>
                          <div className='mt-1 text-sm font-semibold text-[var(--semi-color-text-0)]'>
                            {item.unitPrice.toFixed(2)} 元 / 刀
                          </div>
                        </div>
                      </div>

                      <div className='mt-4 text-xs leading-6 text-[var(--semi-color-text-2)]'>
                        {item.summary}
                      </div>
                      <div className='mt-2 text-xs font-medium text-emerald-700'>
                        对比官网同额，约省 {item.officialSaveValue.toFixed(0)} 元
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className='mt-4 grid grid-cols-2 gap-2 text-xs text-[var(--semi-color-text-2)] sm:grid-cols-4'>
                {pricingPlans.map((item) => (
                  <div
                    key={`${item.quota}-footnote`}
                    className='rounded-xl bg-white/70 px-3 py-2 text-center'
                    style={{ border: '1px solid rgba(148, 163, 184, 0.12)' }}
                  >
                    <div className='font-medium text-[var(--semi-color-text-1)]'>
                      {item.quota}
                    </div>
                    <div className='mt-1'>
                      {item.unitPrice.toFixed(3).replace(/0$/, '')} 元 / 刀
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
          <div className='mt-3 grid grid-cols-1 gap-3'>
            <div className='rounded-xl bg-[var(--semi-color-fill-0)] px-4 py-3'>
              <div className='flex items-center gap-2 text-sm font-medium'>
                <Gift size={16} className='text-rose-500' />
                额外说明
              </div>
              <div className='mt-1 text-xs leading-6 text-[var(--semi-color-text-2)]'>
                GPT Plus 账号一个月 80 元全程质保，Gemini Pro 账号一年 70
                元保一个月，5x(200元) 和 20x(320元) 账号无质保。
              </div>
            </div>
          </div>

          <div className='mt-4 rounded-2xl border border-sky-200 bg-sky-50 px-4 py-4'>
            <div className='flex flex-wrap items-center gap-2'>
              <MessageCircle size={16} className='text-sky-600' />
              <Text strong>购买兑换码私聊客服，Q群：{QQ_GROUP}</Text>
            </div>
            <div className='mt-2 text-sm text-sky-700'>
              邀请人可享新客首单额外赠送 20 刀额度。
            </div>
            <div className='mt-3 flex flex-wrap gap-2'>
              <Button
                theme='solid'
                type='primary'
                icon={<ExternalLink size={14} />}
                onClick={openOfficialSite}
              >
                打开官网
              </Button>
              <Paragraph copyable={{ content: QQ_GROUP }} className='!mb-0 !mt-0'>
                Q群：{QQ_GROUP}
              </Paragraph>
            </div>
          </div>
        </>
      )}
    </Card>
  );
};

export default RechargeSupportCard;
