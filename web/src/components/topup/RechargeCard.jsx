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
  Avatar,
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
} from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { getCurrencyConfig } from '../../helpers/render';
import SubscriptionPlansCard from './SubscriptionPlansCard';
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
  renderQuota,
  statusLoading,
  topupInfo,
  onOpenHistory,
  enableWaffoTopUp,
  enableWaffoPancakeTopUp,
  onlineTopUpEntryEnabled = true,
  subscriptionEntryEnabled = true,
  subscriptionLoading = false,
  subscriptionPlans = [],
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
}) => {
  const onlineFormApiRef = useRef(null);
  const redeemFormApiRef = useRef(null);
  const initialTabSetRef = useRef(false);
  const showAmountSkeleton = useMinimumLoadingTime(amountLoading);
  const [activeTab, setActiveTab] = useState('topup');
  const hasOnlineTopUp =
    enableOnlineTopUp ||
    enableStripeTopUp ||
    enableCreemTopUp ||
    enableWaffoTopUp ||
    enableWaffoPancakeTopUp;
  const shouldShowOnlineTopUp = onlineTopUpEntryEnabled && hasOnlineTopUp;
  const shouldShowSubscription =
    subscriptionEntryEnabled &&
    !subscriptionLoading &&
    subscriptionPlans.length > 0;
  const redemptionOnlyMode = !shouldShowOnlineTopUp && !shouldShowSubscription;
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
    setActiveTab(shouldShowSubscription ? 'subscription' : 'topup');
    initialTabSetRef.current = true;
  }, [shouldShowSubscription, subscriptionLoading]);

  useEffect(() => {
    if (!shouldShowSubscription && activeTab !== 'topup') {
      setActiveTab('topup');
    }
  }, [shouldShowSubscription, activeTab]);
  const topupContent = (
    <Space vertical style={{ width: '100%' }}>
      {/* 统计数据 */}
      <Card
        className='!rounded-xl w-full'
        bodyStyle={shouldShowOnlineTopUp ? undefined : { display: 'none' }}
        cover={
          <div
            className='relative h-30'
            style={{
              '--palette-primary-darkerChannel': '37 99 235',
              backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              backgroundRepeat: 'no-repeat',
            }}
          >
            <div className='relative z-10 h-full flex flex-col justify-between p-4'>
              <div className='flex justify-between items-center'>
                <Text strong style={{ color: 'white', fontSize: '16px' }}>
                  {t('账户统计')}
                </Text>
              </div>

              {/* 统计数据 */}
              <div className='grid grid-cols-3 gap-6 mt-4'>
                {/* 当前余额 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <Wallet
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('当前余额')}
                    </Text>
                  </div>
                </div>

                {/* 历史消耗 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.used_quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <TrendingUp
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('历史消耗')}
                    </Text>
                  </div>
                </div>

                {/* 请求次数 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {userState?.user?.request_count || 0}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <BarChart2
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('请求次数')}
                    </Text>
                  </div>
                </div>
              </div>
            </div>
          </div>
        }
      >
        {/* 在线充值表单 */}
        {shouldShowOnlineTopUp ? (
          statusLoading ? (
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
                          background:
                            'linear-gradient(135deg, rgba(16, 185, 129, 0.08), rgba(59, 130, 246, 0.08))',
                          border: '1px solid rgba(148, 163, 184, 0.2)',
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
                                  ? 'linear-gradient(180deg, rgba(219, 234, 254, 0.95), rgba(255, 255, 255, 1))'
                                  : isBestValue
                                    ? 'linear-gradient(180deg, rgba(236, 253, 245, 0.98), rgba(255, 255, 255, 1))'
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
                                      ? 'rgba(219, 234, 254, 0.72)'
                                      : 'rgba(255, 255, 255, 0.92)',
                                    border:
                                      '1px solid rgba(148, 163, 184, 0.18)',
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
                                      background: 'rgba(248, 250, 252, 0.92)',
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
                                      background: 'rgba(248, 250, 252, 0.92)',
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
                                      ? 'rgba(209, 250, 229, 0.85)'
                                      : 'rgba(248, 250, 252, 0.88)',
                                  }}
                                >
                                  <Text
                                    strong
                                    style={{
                                      fontSize: '12px',
                                      color: isBestValue
                                        ? 'rgba(4, 120, 87, 1)'
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
                          className='cursor-pointer !rounded-2xl transition-all hover:shadow-md border-gray-200 hover:border-gray-300'
                          bodyStyle={{ textAlign: 'center', padding: '16px' }}
                        >
                          <div className='font-medium text-lg mb-2'>
                            {product.name}
                          </div>
                          <div className='text-sm text-gray-600 mb-2'>
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
          )
        ) : null}
      </Card>

      {/* 兑换码充值 */}
      <Card
        className='!rounded-xl w-full'
        title={
          <Text type='tertiary' strong>
            {t('兑换码充值')}
          </Text>
        }
      >
        <Form
          getFormApi={(api) => (redeemFormApiRef.current = api)}
          initValues={{ redemptionCode: redemptionCode }}
        >
          <Form.Input
            field='redemptionCode'
            noLabel={true}
            placeholder={t('请输入兑换码')}
            value={redemptionCode}
            onChange={(value) => setRedemptionCode(value)}
            prefix={<IconGift />}
            suffix={
              <div className='flex items-center gap-2'>
                <Button
                  type='primary'
                  theme='solid'
                  onClick={topUp}
                  loading={isSubmitting}
                >
                  {t('兑换额度')}
                </Button>
              </div>
            }
            showClear
            style={{ width: '100%' }}
            extraText={
              !redemptionOnlyMode &&
              topUpLink && (
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
              )
            }
          />
        </Form>
      </Card>

      <RechargeSupportCard />
    </Space>
  );

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* 卡片头部 */}
      <div className='flex items-center justify-between mb-4'>
        <div className='flex items-center'>
          <Avatar size='small' color='blue' className='mr-3 shadow-md'>
            <CreditCard size={16} />
          </Avatar>
          <div>
            <Typography.Text className='text-lg font-medium'>
              {redemptionOnlyMode ? t('钱包管理') : t('账户充值')}
            </Typography.Text>
            <div className='text-xs'>
              {redemptionOnlyMode
                ? t('使用兑换码兑换额度')
                : t('多种充值方式，安全便捷')}
            </div>
          </div>
        </div>
        <Button
          icon={<Receipt size={16} />}
          theme='solid'
          onClick={onOpenHistory}
        >
          {t('账单')}
        </Button>
      </div>

      {shouldShowSubscription ? (
        <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Sparkles size={16} />
                {t('订阅套餐')}
              </div>
            }
            itemKey='subscription'
          >
            <div className='py-2'>
              <SubscriptionPlansCard
                t={t}
                loading={subscriptionLoading}
                plans={subscriptionPlans}
                payMethods={payMethods}
                enableOnlineTopUp={enableOnlineTopUp}
                enableStripeTopUp={enableStripeTopUp}
                enableCreemTopUp={enableCreemTopUp}
                billingPreference={billingPreference}
                onChangeBillingPreference={onChangeBillingPreference}
                activeSubscriptions={activeSubscriptions}
                allSubscriptions={allSubscriptions}
                reloadSubscriptionSelf={reloadSubscriptionSelf}
                withCard={false}
              />
            </div>
          </TabPane>
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
        </Tabs>
      ) : (
        topupContent
      )}
    </Card>
  );
};

export default RechargeCard;
