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

import React, { useEffect, useRef, useState } from 'react';
import {
  Typography,
  Card,
  Button,
  Skeleton,
  Form,
  Space,
  Row,
  Col,
  Spin,
  Tooltip,
  Tabs,
  TabPane,
  Tag,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe } from 'react-icons/si';
import {
  CreditCard,
  Coins,
  Wallet,
  BarChart2,
  TrendingUp,
  Receipt,
  Sparkles,
  CheckCircle2,
  TicketCheck,
  Activity,
  CalendarDays,
} from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { useActualTheme } from '../../context/Theme';
import { getCurrencyConfig } from '../../helpers/render';
import SubscriptionPlanCatalog from './SubscriptionPlanCatalog';
import SubscriptionStatusPanel from './SubscriptionStatusPanel';
import RechargeSupportCard from '../common/RechargeSupportCard';

const { Text } = Typography;

const getUsdExchangeRate = () => {
  if (typeof window === 'undefined') return 7;

  const statusStr = localStorage.getItem('status');
  let usdRate = 7;
  try {
    if (statusStr) {
      const status = JSON.parse(statusStr);
      usdRate = status?.usd_exchange_rate || 7;
    }
  } catch (e) {}

  return usdRate;
};

const getPresetDisplayInfo = ({
  preset,
  priceRatio,
  topupInfo,
  currencyType,
  currencyRate,
  usdRate,
}) => {
  const discount =
    preset.discount || topupInfo?.discount?.[preset.value] || 1.0;
  const originalPrice = preset.value * priceRatio;
  const discountedPrice = originalPrice * discount;
  const hasDiscount = discount < 1.0;
  const save = originalPrice - discountedPrice;

  let displayValue = preset.value;
  let displayActualPay = discountedPrice;
  let displayOriginalPay = originalPrice;
  let displaySave = save;

  if (currencyType === 'USD') {
    displayActualPay = discountedPrice / usdRate;
    displayOriginalPay = originalPrice / usdRate;
    displaySave = save / usdRate;
  } else if (currencyType === 'CNY') {
    displayValue = preset.value * usdRate;
  } else if (currencyType === 'CUSTOM') {
    displayValue = preset.value * currencyRate;
    displayActualPay = (discountedPrice / usdRate) * currencyRate;
    displayOriginalPay = (originalPrice / usdRate) * currencyRate;
    displaySave = (save / usdRate) * currencyRate;
  }

  return {
    discount,
    hasDiscount,
    save,
    displayValue,
    displayActualPay,
    displayOriginalPay,
    displaySave,
    unitPrice:
      displayValue > 0 && Number.isFinite(displayValue)
        ? displayActualPay / displayValue
        : 0,
  };
};

const formatUnitPrice = (value) => {
  if (!Number.isFinite(value)) return '0.00';
  if (value >= 10) return value.toFixed(2);
  if (value >= 1) return value.toFixed(3);
  return value.toFixed(4);
};

const getDiscountBadgeText = (discount, t) => {
  if (!Number.isFinite(discount) || discount >= 1) {
    return t('直充价');
  }

  const translatedDiscount = t('折');
  if (translatedDiscount.includes('off')) {
    return `${((1 - parseFloat(discount)) * 100).toFixed(0)}% ${translatedDiscount}`;
  }

  return `${(discount * 10).toFixed(1)}${translatedDiscount}`;
};

const RechargeCard = ({
  t,
  enableOnlineTopUp,
  enableStripeTopUp,
  enableCreemTopUp,
  creemProducts,
  creemPreTopUp,
  presetAmounts,
  selectedPreset,
  selectPresetAmount,
  formatLargeNumber,
  priceRatio,
  topUpCount,
  minTopUp,
  renderQuotaWithAmount,
  getAmount,
  setTopUpCount,
  setSelectedPreset,
  renderAmount,
  amountLoading,
  payMethods,
  preTopUp,
  paymentLoading,
  payWay,
  redemptionCode,
  setRedemptionCode,
  topUp,
  isSubmitting,
  topUpLink,
  openTopUpLink,
  userState,
  walletUsageStats,
  renderQuota,
  statusLoading,
  topupInfo,
  onOpenHistory,
  enableWaffoTopUp,
  enableWaffoPancakeTopUp,
  onlineTopUpEntryEnabled = true,
  subscriptionCatalogEnabled = false,
  subscriptionLoading = false,
  subscriptionCatalogLoading = false,
  subscriptionPlans = [],
  billingPreference,
  onChangeBillingPreference,
  quotaBuckets,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  redeemSuccessEffect = null,
}) => {
  const actualTheme = useActualTheme();
  const onlineFormApiRef = useRef(null);
  const redeemFormApiRef = useRef(null);
  const initialTabSetRef = useRef(false);
  const showAmountSkeleton = useMinimumLoadingTime(amountLoading);
  const [activeTab, setActiveTab] = useState('topup');
  const isDarkMode = actualTheme === 'dark';
  const hasOnlineTopUp =
    enableOnlineTopUp ||
    enableStripeTopUp ||
    enableCreemTopUp ||
    enableWaffoTopUp ||
    enableWaffoPancakeTopUp;
  const shouldShowOnlineTopUp = onlineTopUpEntryEnabled && hasOnlineTopUp;
  const shouldShowSubscriptionCatalog = subscriptionCatalogEnabled;
  const redemptionOnlyMode =
    !shouldShowOnlineTopUp && !shouldShowSubscriptionCatalog;
  const regularPayMethods = payMethods || [];
  const { symbol, rate, type } = getCurrencyConfig();
  const usdRate = getUsdExchangeRate();
  const bestValuePreset = presetAmounts.reduce((bestPreset, preset) => {
    const currentPresetInfo = getPresetDisplayInfo({
      preset,
      priceRatio,
      topupInfo,
      currencyType: type,
      currencyRate: rate,
      usdRate,
    });

    if (!bestPreset) {
      return {
        value: preset.value,
        unitPrice: currentPresetInfo.unitPrice,
      };
    }

    if (currentPresetInfo.unitPrice < bestPreset.unitPrice) {
      return {
        value: preset.value,
        unitPrice: currentPresetInfo.unitPrice,
      };
    }

    if (
      currentPresetInfo.unitPrice === bestPreset.unitPrice &&
      preset.value > bestPreset.value
    ) {
      return {
        value: preset.value,
        unitPrice: currentPresetInfo.unitPrice,
      };
    }

    return bestPreset;
  }, null);

  useEffect(() => {
    if (initialTabSetRef.current) return;
    if (subscriptionLoading) return;
    setActiveTab(
      !shouldShowOnlineTopUp && shouldShowSubscriptionCatalog
        ? 'subscription'
        : 'topup',
    );
    initialTabSetRef.current = true;
  }, [
    shouldShowOnlineTopUp,
    shouldShowSubscriptionCatalog,
    subscriptionLoading,
  ]);

  useEffect(() => {
    if (!shouldShowSubscriptionCatalog && activeTab !== 'topup') {
      setActiveTab('topup');
    }
  }, [shouldShowSubscriptionCatalog, activeTab]);

  const walletMetrics = [
    {
      label: t('历史消耗'),
      value: renderQuota(userState?.user?.used_quota || 0),
      icon: TrendingUp,
      accent: 'rgba(124, 58, 237, 1)',
      background: 'rgba(245, 243, 255, 0.88)',
      darkBackground: 'rgba(88, 28, 135, 0.20)',
    },
    {
      label: t('请求次数'),
      value: formatLargeNumber
        ? formatLargeNumber(userState?.user?.request_count || 0)
        : userState?.user?.request_count || 0,
      icon: BarChart2,
      accent: 'rgba(14, 116, 144, 1)',
      background: 'rgba(240, 249, 255, 0.92)',
      darkBackground: 'rgba(14, 116, 144, 0.18)',
    },
    {
      label: t('今日消耗'),
      value: walletUsageStats?.loading
        ? '--'
        : renderQuota(walletUsageStats?.todayQuota || 0),
      icon: Activity,
      accent: 'rgba(5, 150, 105, 1)',
      background: 'rgba(236, 253, 245, 0.92)',
      darkBackground: 'rgba(5, 150, 105, 0.18)',
    },
    {
      label: t('近 7 日消耗'),
      value: walletUsageStats?.loading
        ? '--'
        : renderQuota(walletUsageStats?.weekQuota || 0),
      icon: CalendarDays,
      accent: 'rgba(217, 119, 6, 1)',
      background: 'rgba(255, 251, 235, 0.92)',
      darkBackground: 'rgba(217, 119, 6, 0.18)',
    },
  ];

  const accountStatsPanel = (
    <Card
      className='!rounded-xl shadow-sm overflow-hidden'
      bodyStyle={{ padding: 0 }}
      style={{
        border: isDarkMode
          ? '1px solid rgba(148, 163, 184, 0.16)'
          : '1px solid rgba(15, 23, 42, 0.08)',
        background: isDarkMode
          ? 'linear-gradient(135deg, rgba(15, 23, 42, 0.96), rgba(17, 24, 39, 0.98))'
          : 'linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.98))',
      }}
    >
      <div className='relative overflow-hidden'>
        <div
          className='absolute inset-y-0 left-0 w-1'
          style={{
            background:
              'linear-gradient(180deg, rgba(14, 165, 233, 0.96), rgba(5, 150, 105, 0.9), rgba(99, 102, 241, 0.72))',
          }}
        />
        <div
          className='pointer-events-none absolute inset-0'
          style={{
            background: isDarkMode
              ? 'radial-gradient(circle at 14% 0%, rgba(14, 165, 233, 0.12), transparent 32%), radial-gradient(circle at 78% 110%, rgba(5, 150, 105, 0.10), transparent 30%)'
              : 'radial-gradient(circle at 12% 0%, rgba(14, 165, 233, 0.10), transparent 32%), radial-gradient(circle at 76% 110%, rgba(5, 150, 105, 0.09), transparent 30%)',
          }}
        />

        <div className='relative grid grid-cols-1 lg:grid-cols-[minmax(250px,0.9fr)_minmax(0,1.8fr)_auto]'>
          <div className='flex min-w-0 items-center gap-3 px-4 py-4 lg:px-5'>
            <div
              className='flex h-10 w-10 shrink-0 items-center justify-center rounded-xl'
              style={{
                background: isDarkMode
                  ? 'rgba(14, 165, 233, 0.16)'
                  : 'rgba(14, 165, 233, 0.10)',
                color: 'var(--semi-color-primary)',
              }}
            >
              <Wallet size={18} />
            </div>
            <div className='min-w-0'>
              <div className='flex flex-wrap items-center gap-2'>
                <Text strong>{t('钱包概览')}</Text>
                <Tag color='cyan' shape='circle' size='small'>
                  {t('长期余额')}
                </Tag>
              </div>
              <div className='mt-1 text-2xl font-semibold leading-tight text-[var(--semi-color-text-0)] break-words lg:text-3xl'>
                {renderQuota(userState?.user?.quota || 0)}
              </div>
              <Text type='tertiary' size='small'>
                {t('永久或长期余额，仅在钱包中展示')}
              </Text>
            </div>
          </div>

          <div
            className='grid grid-cols-2 md:grid-cols-4'
            style={{
              borderLeft: isDarkMode
                ? '1px solid rgba(148, 163, 184, 0.12)'
                : '1px solid rgba(15, 23, 42, 0.06)',
              borderTop: isDarkMode
                ? '1px solid rgba(148, 163, 184, 0.12)'
                : '1px solid rgba(15, 23, 42, 0.06)',
            }}
          >
            {walletMetrics.map((metric, index) => {
              const MetricIcon = metric.icon;
              return (
                <div
                  key={metric.label}
                  className='min-w-0 px-4 py-3'
                  style={{
                    borderLeft:
                      index % 4 === 0
                        ? 'none'
                        : isDarkMode
                          ? '1px solid rgba(148, 163, 184, 0.10)'
                          : '1px solid rgba(15, 23, 42, 0.055)',
                  }}
                >
                  <div className='flex items-center gap-2 text-xs text-[var(--semi-color-text-2)]'>
                    <MetricIcon size={14} color={metric.accent} />
                    <span className='truncate'>{metric.label}</span>
                  </div>
                  <div className='mt-1.5 text-base font-semibold leading-tight text-[var(--semi-color-text-0)] break-words'>
                    {metric.value}
                  </div>
                </div>
              );
            })}
          </div>

          <div
            className='flex items-center justify-start px-4 py-3 lg:justify-end'
            style={{
              borderLeft: isDarkMode
                ? '1px solid rgba(148, 163, 184, 0.12)'
                : '1px solid rgba(15, 23, 42, 0.06)',
              borderTop: isDarkMode
                ? '1px solid rgba(148, 163, 184, 0.12)'
                : '1px solid rgba(15, 23, 42, 0.06)',
            }}
          >
            <Button
              size='small'
              theme='light'
              type='tertiary'
              icon={<Receipt size={14} />}
              onClick={onOpenHistory}
            >
              {t('账单')}
            </Button>
          </div>
        </div>
      </div>
    </Card>
  );

  const topupContent = (
    <Space vertical style={{ width: '100%' }}>
      {shouldShowOnlineTopUp ? (
        <Card
          className='!rounded-xl w-full'
          title={
            <Text type='tertiary' strong>
              {t('额度充值')}
            </Text>
          }
        >
          {statusLoading ? (
            <div className='py-8 flex justify-center'>
              <Spin size='large' />
            </div>
          ) : (
            <Form
              getFormApi={(api) => (onlineFormApiRef.current = api)}
              initValues={{ topUpCount: topUpCount }}
            >
              <div className='space-y-6'>
                {(enableOnlineTopUp ||
                  enableStripeTopUp ||
                  enableWaffoTopUp ||
                  enableWaffoPancakeTopUp) && (
                  <Row gutter={12}>
                    <Col xs={24} sm={24} md={24} lg={10} xl={10}>
                      <Form.InputNumber
                        field='topUpCount'
                        label={t('充值数量')}
                        disabled={
                          !enableOnlineTopUp &&
                          !enableStripeTopUp &&
                          !enableWaffoTopUp &&
                          !enableWaffoPancakeTopUp
                        }
                        placeholder={
                          t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)
                        }
                        value={topUpCount}
                        min={minTopUp}
                        max={999999999}
                        step={1}
                        precision={0}
                        onChange={async (value) => {
                          if (value && value >= 1) {
                            setTopUpCount(value);
                            setSelectedPreset(null);
                            await getAmount(value);
                          }
                        }}
                        onBlur={(e) => {
                          const value = parseInt(e.target.value);
                          if (!value || value < 1) {
                            setTopUpCount(1);
                            getAmount(1);
                          }
                        }}
                        formatter={(value) => (value ? `${value}` : '')}
                        parser={(value) =>
                          value ? parseInt(value.replace(/[^\d]/g, '')) : 0
                        }
                        extraText={
                          <Skeleton
                            loading={showAmountSkeleton}
                            active
                            placeholder={
                              <Skeleton.Title
                                style={{
                                  width: 120,
                                  height: 20,
                                  borderRadius: 6,
                                }}
                              />
                            }
                          >
                            <Text type='secondary' className='text-red-600'>
                              {t('实付金额：')}
                              <span style={{ color: 'red' }}>
                                {renderAmount()}
                              </span>
                            </Text>
                          </Skeleton>
                        }
                        style={{ width: '100%' }}
                      />
                    </Col>
                    {regularPayMethods.length > 0 && (
                      <Col xs={24} sm={24} md={24} lg={14} xl={14}>
                        <Form.Slot label={t('选择支付方式')}>
                          <Space wrap>
                            {regularPayMethods.map((payMethod) => {
                              const minTopupVal =
                                Number(payMethod.min_topup) || 0;
                              const isStripe = payMethod.type === 'stripe';
                              const isWaffo =
                                typeof payMethod.type === 'string' &&
                                payMethod.type.startsWith('waffo:');
                              const isWaffoPancake =
                                payMethod.type === 'waffo_pancake';
                              const disabled =
                                (!enableOnlineTopUp &&
                                  !isStripe &&
                                  !isWaffo &&
                                  !isWaffoPancake) ||
                                (!enableStripeTopUp && isStripe) ||
                                (!enableWaffoTopUp && isWaffo) ||
                                (!enableWaffoPancakeTopUp && isWaffoPancake) ||
                                minTopupVal > Number(topUpCount || 0);

                              const buttonEl = (
                                <Button
                                  key={payMethod.type}
                                  theme='outline'
                                  type='tertiary'
                                  onClick={() => preTopUp(payMethod.type)}
                                  disabled={disabled}
                                  loading={
                                    paymentLoading && payWay === payMethod.type
                                  }
                                  icon={
                                    payMethod.type === 'alipay' ? (
                                      <SiAlipay size={18} color='#1677FF' />
                                    ) : payMethod.type === 'wxpay' ? (
                                      <SiWechat size={18} color='#07C160' />
                                    ) : payMethod.type === 'stripe' ? (
                                      <SiStripe size={18} color='#635BFF' />
                                    ) : payMethod.icon ? (
                                      <img
                                        src={payMethod.icon}
                                        alt={payMethod.name}
                                        style={{
                                          width: 18,
                                          height: 18,
                                          objectFit: 'contain',
                                        }}
                                      />
                                    ) : payMethod.type === 'waffo_pancake' ? (
                                      <CreditCard
                                        size={18}
                                        color='var(--semi-color-primary)'
                                      />
                                    ) : (
                                      <CreditCard
                                        size={18}
                                        color={
                                          payMethod.color ||
                                          'var(--semi-color-text-2)'
                                        }
                                      />
                                    )
                                  }
                                  className='!rounded-lg !px-4 !py-2'
                                >
                                  {payMethod.name}
                                </Button>
                              );

                              return disabled &&
                                minTopupVal > Number(topUpCount || 0) ? (
                                <Tooltip
                                  content={
                                    t('此支付方式最低充值金额为') +
                                    ' ' +
                                    minTopupVal
                                  }
                                  key={payMethod.type}
                                >
                                  {buttonEl}
                                </Tooltip>
                              ) : (
                                <React.Fragment key={payMethod.type}>
                                  {buttonEl}
                                </React.Fragment>
                              );
                            })}
                          </Space>
                        </Form.Slot>
                      </Col>
                    )}
                  </Row>
                )}

                {(enableOnlineTopUp ||
                  enableStripeTopUp ||
                  enableWaffoTopUp) && (
                  <Form.Slot
                    label={
                      <div className='flex items-center gap-2'>
                        <span>{t('选择充值额度')}</span>
                        {(() => {
                          if (type === 'USD') return null;

                          return (
                            <span
                              style={{
                                color: 'var(--semi-color-text-2)',
                                fontSize: '12px',
                                fontWeight: 'normal',
                              }}
                            >
                              {t('(1 $ =')} {rate.toFixed(2)} {symbol}
                              {')'}
                            </span>
                          );
                        })()}
                      </div>
                    }
                  >
                    <div className='space-y-3'>
                      <div
                        className='rounded-2xl px-4 py-3 flex flex-wrap items-center gap-2'
                        style={{
                          background: isDarkMode
                            ? 'linear-gradient(135deg, rgba(16, 185, 129, 0.16), rgba(59, 130, 246, 0.16))'
                            : 'linear-gradient(135deg, rgba(16, 185, 129, 0.08), rgba(59, 130, 246, 0.08))',
                          border: isDarkMode
                            ? '1px solid rgba(148, 163, 184, 0.24)'
                            : '1px solid rgba(148, 163, 184, 0.2)',
                        }}
                      >
                        <Sparkles
                          size={16}
                          style={{ color: 'var(--semi-color-primary)' }}
                        />
                        <Text strong>{t('大额档位通常更划算')}</Text>
                        <Text
                          type='tertiary'
                          style={{ fontSize: '12px', lineHeight: 1.5 }}
                        >
                          {t(
                            '重点看实付、立省和折合单价，推荐优先选择高面额档位。',
                          )}
                        </Text>
                      </div>

                      <div className='grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3'>
                        {presetAmounts.map((preset, index) => {
                          const {
                            discount,
                            hasDiscount,
                            displayValue,
                            displayActualPay,
                            displayOriginalPay,
                            displaySave,
                            unitPrice,
                          } = getPresetDisplayInfo({
                            preset,
                            priceRatio,
                            topupInfo,
                            currencyType: type,
                            currencyRate: rate,
                            usdRate,
                          });
                          const isSelected = selectedPreset === preset.value;
                          const isBestValue =
                            bestValuePreset?.value === preset.value;

                          return (
                            <Card
                              key={preset.value ?? index}
                              className='relative overflow-hidden !rounded-2xl transition-all duration-200 hover:-translate-y-1 hover:shadow-lg'
                              style={{
                                cursor: 'pointer',
                                border: isSelected
                                  ? '2px solid var(--semi-color-primary)'
                                  : isBestValue
                                    ? '1px solid rgba(59, 130, 246, 0.35)'
                                    : '1px solid var(--semi-color-border)',
                                height: '100%',
                                width: '100%',
                                background: isSelected
                                  ? isDarkMode
                                    ? 'linear-gradient(180deg, rgba(37, 99, 235, 0.30), rgba(15, 23, 42, 0.96))'
                                    : 'linear-gradient(180deg, rgba(219, 234, 254, 0.95), rgba(255, 255, 255, 1))'
                                  : isBestValue
                                    ? isDarkMode
                                      ? 'linear-gradient(180deg, rgba(5, 150, 105, 0.24), rgba(15, 23, 42, 0.96))'
                                      : 'linear-gradient(180deg, rgba(236, 253, 245, 0.98), rgba(255, 255, 255, 1))'
                                    : isDarkMode
                                      ? 'linear-gradient(180deg, rgba(24, 24, 27, 0.96), rgba(15, 23, 42, 0.92))'
                                      : 'linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(255, 255, 255, 1))',
                                boxShadow: isSelected
                                  ? '0 18px 34px rgba(59, 130, 246, 0.14)'
                                  : undefined,
                              }}
                              bodyStyle={{ padding: '0' }}
                              onClick={() => {
                                selectPresetAmount(preset);
                                onlineFormApiRef.current?.setValue(
                                  'topUpCount',
                                  preset.value,
                                );
                              }}
                            >
                              <div
                                className='h-1.5 w-full'
                                style={{
                                  background: isSelected
                                    ? 'linear-gradient(90deg, rgba(37, 99, 235, 1), rgba(96, 165, 250, 1))'
                                    : isBestValue
                                      ? 'linear-gradient(90deg, rgba(16, 185, 129, 1), rgba(59, 130, 246, 1))'
                                      : isDarkMode
                                        ? 'linear-gradient(90deg, rgba(71, 85, 105, 0.95), rgba(30, 41, 59, 0.95))'
                                        : 'linear-gradient(90deg, rgba(226, 232, 240, 1), rgba(248, 250, 252, 1))',
                                }}
                              />
                              <div className='p-4'>
                                <div className='flex items-start justify-between gap-3'>
                                  <div>
                                    <Text
                                      strong
                                      style={{
                                        fontSize: '13px',
                                        color: isBestValue
                                          ? 'var(--semi-color-primary)'
                                          : 'var(--semi-color-text-0)',
                                      }}
                                    >
                                      {isBestValue
                                        ? t('超值推荐')
                                        : t('灵活充值')}
                                    </Text>
                                    <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                                      {t('到账快，适合直接补充额度')}
                                    </div>
                                  </div>
                                  <div className='flex flex-col items-end gap-1'>
                                    <Tag
                                      color={
                                        hasDiscount
                                          ? isBestValue
                                            ? 'blue'
                                            : 'green'
                                          : 'grey'
                                      }
                                    >
                                      {getDiscountBadgeText(discount, t)}
                                    </Tag>
                                    {isSelected && (
                                      <Tag color='blue'>{t('当前选择')}</Tag>
                                    )}
                                  </div>
                                </div>

                                <div className='mt-5'>
                                  <div className='flex items-center gap-2 text-[var(--semi-color-text-0)]'>
                                    <Coins size={18} />
                                    <Typography.Title
                                      heading={5}
                                      style={{ margin: 0 }}
                                    >
                                      {formatLargeNumber(displayValue)} {symbol}
                                    </Typography.Title>
                                  </div>
                                  <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                                    {t('到账额度')}
                                  </div>
                                </div>

                                <div
                                  className='mt-4 rounded-2xl px-4 py-3'
                                  style={{
                                    background: isSelected
                                      ? isDarkMode
                                        ? 'rgba(37, 99, 235, 0.22)'
                                        : 'rgba(219, 234, 254, 0.72)'
                                      : isDarkMode
                                        ? 'rgba(15, 23, 42, 0.72)'
                                        : 'rgba(255, 255, 255, 0.92)',
                                    border: isDarkMode
                                      ? '1px solid rgba(148, 163, 184, 0.20)'
                                      : '1px solid rgba(148, 163, 184, 0.18)',
                                  }}
                                >
                                  <div className='text-xs text-[var(--semi-color-text-2)]'>
                                    {t('实付金额')}
                                  </div>
                                  <div className='mt-2 flex items-end gap-2'>
                                    <span
                                      style={{
                                        color: 'var(--semi-color-text-0)',
                                        fontSize: '28px',
                                        fontWeight: 700,
                                        lineHeight: 1,
                                      }}
                                    >
                                      {symbol}
                                      {displayActualPay.toFixed(2)}
                                    </span>
                                    {hasDiscount && (
                                      <span
                                        style={{
                                          color: 'var(--semi-color-text-2)',
                                          fontSize: '13px',
                                          textDecoration: 'line-through',
                                          marginBottom: '2px',
                                        }}
                                      >
                                        {symbol}
                                        {displayOriginalPay.toFixed(2)}
                                      </span>
                                    )}
                                  </div>
                                  <div
                                    className='mt-2 text-xs'
                                    style={{
                                      color: hasDiscount
                                        ? 'rgba(5, 150, 105, 1)'
                                        : 'var(--semi-color-text-2)',
                                    }}
                                  >
                                    {hasDiscount
                                      ? `${t('立省')} ${symbol}${displaySave.toFixed(2)}`
                                      : t('当前已是稳定直充价')}
                                  </div>
                                </div>

                                <div className='mt-4 grid grid-cols-2 gap-2'>
                                  <div
                                    className='rounded-xl px-3 py-3'
                                    style={{
                                      background: isDarkMode
                                        ? 'rgba(15, 23, 42, 0.62)'
                                        : 'rgba(248, 250, 252, 0.92)',
                                    }}
                                  >
                                    <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                                      {t('折合单价')}
                                    </div>
                                    <div className='mt-1 font-semibold text-[var(--semi-color-text-0)]'>
                                      {symbol}
                                      {formatUnitPrice(unitPrice)}
                                    </div>
                                  </div>
                                  <div
                                    className='rounded-xl px-3 py-3'
                                    style={{
                                      background: isDarkMode
                                        ? 'rgba(15, 23, 42, 0.62)'
                                        : 'rgba(248, 250, 252, 0.92)',
                                    }}
                                  >
                                    <div className='text-[11px] text-[var(--semi-color-text-2)]'>
                                      {hasDiscount
                                        ? t('优惠力度')
                                        : t('购买体验')}
                                    </div>
                                    <div className='mt-1 font-semibold text-[var(--semi-color-text-0)]'>
                                      {hasDiscount
                                        ? `${getDiscountBadgeText(discount, t)}`
                                        : t('到账即用')}
                                    </div>
                                  </div>
                                </div>

                                <div
                                  className='mt-4 flex items-center justify-between rounded-xl px-3 py-2'
                                  style={{
                                    background: isBestValue
                                      ? isDarkMode
                                        ? 'rgba(5, 150, 105, 0.18)'
                                        : 'rgba(209, 250, 229, 0.85)'
                                      : isDarkMode
                                        ? 'rgba(15, 23, 42, 0.54)'
                                        : 'rgba(248, 250, 252, 0.88)',
                                  }}
                                >
                                  <Text
                                    strong
                                    style={{
                                      fontSize: '12px',
                                      color: isBestValue
                                        ? isDarkMode
                                          ? 'rgba(52, 211, 153, 1)'
                                          : 'rgba(4, 120, 87, 1)'
                                        : 'var(--semi-color-text-1)',
                                    }}
                                  >
                                    {isBestValue
                                      ? t('推荐给高频用户')
                                      : t('适合灵活补充额度')}
                                  </Text>
                                  <Text
                                    style={{
                                      fontSize: '12px',
                                      color: 'var(--semi-color-text-2)',
                                    }}
                                  >
                                    {t('省心直观')}
                                  </Text>
                                </div>
                              </div>
                            </Card>
                          );
                        })}
                      </div>
                    </div>
                  </Form.Slot>
                )}

                {/* Creem 充值区域 */}
                {enableCreemTopUp && creemProducts.length > 0 && (
                  <Form.Slot label={t('Creem 充值')}>
                    <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3'>
                      {creemProducts.map((product, index) => (
                        <Card
                          key={index}
                          onClick={() => creemPreTopUp(product)}
                          className='cursor-pointer !rounded-2xl transition-all hover:shadow-md border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
                          bodyStyle={{ textAlign: 'center', padding: '16px' }}
                        >
                          <div className='font-medium text-lg mb-2'>
                            {product.name}
                          </div>
                          <div className='text-sm text-gray-600 dark:text-gray-300 mb-2'>
                            {t('充值额度')}: {product.quota}
                          </div>
                          <div className='text-lg font-semibold text-blue-600'>
                            {product.currency === 'EUR' ? '€' : '$'}
                            {product.price}
                          </div>
                        </Card>
                      ))}
                    </div>
                  </Form.Slot>
                )}
              </div>
            </Form>
          )}
        </Card>
      ) : null}

      {/* 兑换码充值 */}
      <Card
        className='!rounded-2xl w-full relative overflow-hidden transition-all duration-300'
        bodyStyle={{ padding: 0 }}
        style={{
          border: redeemSuccessEffect
            ? '1px solid rgba(5, 150, 105, 0.42)'
            : '1px solid rgba(14, 165, 233, 0.20)',
          boxShadow: redeemSuccessEffect
            ? '0 18px 44px rgba(5, 150, 105, 0.15)'
            : '0 14px 34px rgba(14, 165, 233, 0.08)',
        }}
      >
        <div
          className='absolute inset-x-0 top-0 h-1.5'
          style={{
            background:
              'linear-gradient(90deg, rgba(14, 165, 233, 0.94), rgba(5, 150, 105, 0.88), rgba(245, 158, 11, 0.78))',
          }}
        />
        <div
          className='pointer-events-none absolute inset-0 opacity-70'
          style={{
            backgroundImage: isDarkMode
              ? 'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.06) 45%, transparent 64%), radial-gradient(circle at 12% 12%, rgba(14, 165, 233, 0.16), transparent 26%), radial-gradient(circle at 88% 8%, rgba(5, 150, 105, 0.16), transparent 22%)'
              : 'linear-gradient(115deg, transparent 0%, rgba(255,255,255,0.42) 45%, transparent 64%), radial-gradient(circle at 12% 12%, rgba(14, 165, 233, 0.12), transparent 26%), radial-gradient(circle at 88% 8%, rgba(5, 150, 105, 0.12), transparent 22%)',
          }}
        />
        {redeemSuccessEffect && (
          <div className='pointer-events-none absolute inset-0'>
            <div className='absolute right-8 top-8 h-14 w-14 rounded-full bg-emerald-300/20 animate-ping' />
            <div className='absolute bottom-5 left-10 h-2 w-24 rounded-full bg-emerald-400/40 animate-pulse' />
          </div>
        )}

        <div className='relative grid grid-cols-1 gap-4 p-4 lg:grid-cols-[0.82fr_1.18fr] lg:items-center'>
          <div className='min-w-0'>
            <div className='flex flex-wrap items-center gap-2'>
              <span
                className='inline-flex h-10 w-10 items-center justify-center rounded-2xl'
                style={{
                  background: redeemSuccessEffect
                    ? isDarkMode
                      ? 'rgba(5, 150, 105, 0.18)'
                      : 'rgba(220, 252, 231, 0.92)'
                    : isDarkMode
                      ? 'rgba(14, 116, 144, 0.18)'
                      : 'rgba(240, 249, 255, 0.96)',
                  color: redeemSuccessEffect
                    ? isDarkMode
                      ? 'rgba(52, 211, 153, 1)'
                      : 'rgba(5, 150, 105, 1)'
                    : isDarkMode
                      ? 'rgba(34, 211, 238, 1)'
                      : 'rgba(14, 116, 144, 1)',
                  border: redeemSuccessEffect
                    ? isDarkMode
                      ? '1px solid rgba(52, 211, 153, 0.26)'
                      : '1px solid rgba(5, 150, 105, 0.24)'
                    : isDarkMode
                      ? '1px solid rgba(34, 211, 238, 0.24)'
                      : '1px solid rgba(14, 165, 233, 0.20)',
                }}
              >
                {redeemSuccessEffect ? (
                  <CheckCircle2 size={19} />
                ) : (
                  <TicketCheck size={19} />
                )}
              </span>
              <div className='min-w-0'>
                <div className='flex flex-wrap items-center gap-2'>
                  <Text strong style={{ fontSize: 16 }}>
                    {redeemSuccessEffect ? t('补给已到账') : t('兑换码补给')}
                  </Text>
                  <Tag
                    color={redeemSuccessEffect ? 'green' : 'cyan'}
                    shape='circle'
                    size='small'
                  >
                    {redeemSuccessEffect
                      ? t('已到账')
                      : t('自动识别额度码、一周畅用包和套餐码')}
                  </Tag>
                </div>
                <Text type='tertiary' size='small'>
                  {redeemSuccessEffect
                    ? redeemSuccessEffect.summary
                    : t('输入兑换码完成补给')}
                </Text>
              </div>
            </div>

            <div className='mt-3 text-xs leading-5 text-[var(--semi-color-text-2)]'>
              {t('兑换后会自动刷新余额、一周畅用包或套餐权益')}
            </div>
          </div>

          <Form
            getFormApi={(api) => (redeemFormApiRef.current = api)}
            initValues={{ redemptionCode: redemptionCode }}
          >
            <div>
              <div className='grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-center'>
                <Form.Input
                  field='redemptionCode'
                  noLabel={true}
                  fieldStyle={{
                    paddingTop: 0,
                    paddingBottom: 0,
                    overflow: 'visible',
                  }}
                  placeholder={t('请输入兑换码')}
                  value={redemptionCode}
                  onChange={(value) => setRedemptionCode(value)}
                  prefix={<IconGift />}
                  showClear
                  style={{ width: '100%', height: 32 }}
                />
                <Button
                  type='primary'
                  theme='solid'
                  className='w-full md:w-auto md:min-w-[116px] md:self-center'
                  onClick={topUp}
                  loading={isSubmitting}
                  style={{ height: 32 }}
                >
                  {isSubmitting ? t('正在兑换') : t('立即兑换')}
                </Button>
              </div>
              {!redemptionOnlyMode && topUpLink && (
                <div className='mt-1.5 text-xs leading-5'>
                  <Text type='tertiary'>
                    {t('在找兑换码？')}
                    <Text
                      type='secondary'
                      underline
                      className='cursor-pointer'
                      onClick={openTopUpLink}
                    >
                      {t('购买兑换码')}
                    </Text>
                  </Text>
                </div>
              )}
            </div>
          </Form>
        </div>
      </Card>

      <RechargeSupportCard
        topUpLink={topUpLink}
        subscriptionPlans={subscriptionPlans}
        subscriptionPlansLoading={subscriptionCatalogLoading}
      />
    </Space>
  );

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex items-center gap-3'>
          <div
            className='h-10 w-10 rounded-xl flex items-center justify-center'
            style={{
              background: 'var(--semi-color-primary-light-default)',
              color: 'var(--semi-color-primary)',
            }}
          >
            <CreditCard size={18} />
          </div>
          <div>
            <Typography.Title heading={4} style={{ margin: 0 }}>
              {t('账户权益')}
            </Typography.Title>
            <Text type='tertiary' size='small'>
              {redemptionOnlyMode
                ? t('查看余额与已购订阅，使用兑换码补充额度')
                : t('查看余额、订阅权益与充值入口')}
            </Text>
          </div>
        </div>
      </div>

      <div className='space-y-4'>
        {accountStatsPanel}
        <SubscriptionStatusPanel
          t={t}
          loading={subscriptionLoading}
          plans={subscriptionPlans}
          billingPreference={billingPreference}
          onChangeBillingPreference={onChangeBillingPreference}
          quotaBuckets={quotaBuckets}
          activeSubscriptions={activeSubscriptions}
          allSubscriptions={allSubscriptions}
          reloadSubscriptionSelf={reloadSubscriptionSelf}
          catalogEnabled={shouldShowSubscriptionCatalog}
          onViewCatalog={() => setActiveTab('subscription')}
        />
      </div>

      {shouldShowSubscriptionCatalog ? (
        <Tabs type='line' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Wallet size={16} />
                {t('额度充值')}
              </div>
            }
            itemKey='topup'
          >
            <div className='py-2'>{topupContent}</div>
          </TabPane>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Sparkles size={16} />
                {t('月卡套餐')}
              </div>
            }
            itemKey='subscription'
          >
            <div className='py-2'>
              <SubscriptionPlanCatalog
                t={t}
                loading={subscriptionCatalogLoading}
                plans={subscriptionPlans}
                payMethods={payMethods}
                enableOnlineTopUp={enableOnlineTopUp}
                enableStripeTopUp={enableStripeTopUp}
                enableCreemTopUp={enableCreemTopUp}
                allSubscriptions={allSubscriptions}
                topUpLink={topUpLink}
              />
            </div>
          </TabPane>
        </Tabs>
      ) : (
        topupContent
      )}
    </div>
  );
};

export default RechargeCard;
