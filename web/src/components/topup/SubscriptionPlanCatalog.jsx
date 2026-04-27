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

import React, { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Divider,
  Skeleton,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { CheckCircle2, ExternalLink, Rocket, Sparkles } from 'lucide-react';
import {
  API,
  openRechargeLink,
  renderQuota,
  showError,
  showSuccess,
} from '../../helpers';
import { getCurrencyConfig } from '../../helpers/render';
import {
  formatSubscriptionDuration,
  formatSubscriptionResetPeriod,
} from '../../helpers/subscriptionFormat';
import SubscriptionPurchaseModal from './modals/SubscriptionPurchaseModal';

const { Text } = Typography;

function getEpayMethods(payMethods = []) {
  return (payMethods || []).filter(
    (method) =>
      method?.type && method.type !== 'stripe' && method.type !== 'creem',
  );
}

function submitEpayForm({ url, params }) {
  const form = document.createElement('form');
  form.action = url;
  form.method = 'POST';
  const isSafari =
    navigator.userAgent.indexOf('Safari') > -1 &&
    navigator.userAgent.indexOf('Chrome') < 1;
  if (!isSafari) {
    form.target = '_blank';
  }
  Object.keys(params || {}).forEach((key) => {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = key;
    input.value = params[key];
    form.appendChild(input);
  });
  document.body.appendChild(form);
  form.submit();
  document.body.removeChild(form);
}

const getDisplayPrice = (plan) => {
  const { symbol, rate } = getCurrencyConfig();
  const price = Number(plan?.price_amount || 0);
  const convertedPrice = price * rate;
  return {
    symbol,
    displayPrice: convertedPrice.toFixed(
      Number.isInteger(convertedPrice) ? 0 : 2,
    ),
  };
};

const isPremiumBlackPlan = (plan) => {
  const text = `${plan?.title || ''} ${plan?.subtitle || ''}`;
  return /pro\s*50x|pro50x|黑卡|每日\s*\$?300/i.test(text);
};

const SubscriptionPlanCatalog = ({
  t,
  loading = false,
  plans = [],
  payMethods = [],
  enableOnlineTopUp = false,
  enableStripeTopUp = false,
  enableCreemTopUp = false,
  allSubscriptions = [],
  topUpLink = '',
}) => {
  const [open, setOpen] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState(null);
  const [paying, setPaying] = useState(false);
  const [selectedEpayMethod, setSelectedEpayMethod] = useState('');

  const epayMethods = useMemo(() => getEpayMethods(payMethods), [payMethods]);

  const planPurchaseCountMap = useMemo(() => {
    const map = new Map();
    (allSubscriptions || []).forEach((item) => {
      const planId = item?.subscription?.plan_id;
      if (!planId) return;
      map.set(planId, (map.get(planId) || 0) + 1);
    });
    return map;
  }, [allSubscriptions]);

  const getPlanPurchaseCount = (planId) =>
    planPurchaseCountMap.get(planId) || 0;

  const openBuy = (planWrapper) => {
    setSelectedPlan(planWrapper);
    setSelectedEpayMethod(epayMethods?.[0]?.type || '');
    setOpen(true);
  };

  const closeBuy = () => {
    setOpen(false);
    setSelectedPlan(null);
    setPaying(false);
  };

  const payStripe = async () => {
    if (!selectedPlan?.plan?.stripe_price_id) {
      showError(t('该套餐未配置 Stripe'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/stripe/pay', {
        plan_id: selectedPlan.plan.id,
      });
      if (res.data?.message === 'success') {
        window.open(res.data.data?.pay_link, '_blank');
        showSuccess(t('已打开支付页面'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const payCreem = async () => {
    if (!selectedPlan?.plan?.creem_product_id) {
      showError(t('该套餐未配置 Creem'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/creem/pay', {
        plan_id: selectedPlan.plan.id,
      });
      if (res.data?.message === 'success') {
        window.open(res.data.data?.checkout_url, '_blank');
        showSuccess(t('已打开支付页面'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const payEpay = async () => {
    if (!selectedEpayMethod) {
      showError(t('请选择支付方式'));
      return;
    }
    setPaying(true);
    try {
      const res = await API.post('/api/subscription/epay/pay', {
        plan_id: selectedPlan.plan.id,
        payment_method: selectedEpayMethod,
      });
      if (res.data?.message === 'success') {
        submitEpayForm({ url: res.data.url, params: res.data.data });
        showSuccess(t('已发起支付'));
        closeBuy();
      } else {
        const errorMsg =
          typeof res.data?.data === 'string'
            ? res.data.data
            : res.data?.message || t('支付失败');
        showError(errorMsg);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaying(false);
    }
  };

  const skeleton = (
    <div className='grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4'>
      {[1, 2, 3].map((item) => (
        <Card
          key={item}
          className='!rounded-2xl w-full h-full'
          bodyStyle={{ padding: 18 }}
        >
          <Skeleton.Title active style={{ width: '62%', height: 24 }} />
          <Skeleton.Paragraph active rows={1} />
          <Skeleton.Title
            active
            style={{ width: '44%', height: 36, marginTop: 16 }}
          />
          <Skeleton.Paragraph active rows={3} />
          <Skeleton.Button active block style={{ marginTop: 16 }} />
        </Card>
      ))}
    </div>
  );

  return (
    <>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-end justify-between gap-3'>
          <div>
            <div className='flex items-center gap-2'>
              <Rocket size={18} color='var(--semi-color-primary)' />
              <Text strong>{t('月卡套餐')}</Text>
            </div>
            <Text type='tertiary' size='small'>
              {t('购买区独立展示，已购权益以上方“我的订阅”为准')}
            </Text>
          </div>
          <Button
            theme='solid'
            type='warning'
            icon={<ExternalLink size={14} />}
            onClick={() => openRechargeLink(topUpLink)}
          >
            {t('淘宝购买月卡')}
          </Button>
        </div>

        {loading ? (
          skeleton
        ) : plans.length > 0 ? (
          <div className='grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4'>
            {plans.map((item) => {
              const plan = item?.plan;
              const totalAmount = Number(plan?.total_amount || 0);
              const { symbol, displayPrice } = getDisplayPrice(plan);
              const isRecommended =
                (plan?.title || '').trim() === '前进三：巡航';
              const isPremium = isPremiumBlackPlan(plan);
              const limit = Number(plan?.max_purchase_per_user || 0);
              const purchaseCount = getPlanPurchaseCount(plan?.id);
              const reachedLimit = limit > 0 && purchaseCount >= limit;
              const resetText = formatSubscriptionResetPeriod(plan, t);
              const benefits = [
                `${t('有效期')}: ${formatSubscriptionDuration(plan, t)}`,
                resetText === t('不重置')
                  ? null
                  : `${t('额度重置')}: ${resetText}`,
                totalAmount > 0
                  ? `${t('总额度')}: ${renderQuota(totalAmount)}`
                  : `${t('总额度')}: ${t('不限')}`,
                limit > 0 ? `${t('限购')} ${limit}` : null,
                plan?.upgrade_group
                  ? `${t('升级分组')}: ${plan.upgrade_group}`
                  : null,
              ].filter(Boolean);

              return (
                <Card
                  key={plan?.id}
                  className='!rounded-2xl transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md h-full'
                  bodyStyle={{ padding: 0 }}
                  style={{
                    border: isPremium
                      ? '1px solid rgba(148, 163, 184, 0.34)'
                      : isRecommended
                        ? '1px solid rgba(37, 99, 235, 0.45)'
                        : '1px solid var(--semi-color-border)',
                    background: isPremium
                      ? 'linear-gradient(135deg, rgba(2, 6, 23, 0.98), rgba(15, 23, 42, 1) 54%, rgba(39, 39, 42, 0.96))'
                      : undefined,
                    boxShadow: isPremium
                      ? '0 22px 48px rgba(2, 6, 23, 0.24)'
                      : isRecommended
                        ? '0 18px 40px rgba(37, 99, 235, 0.10)'
                        : undefined,
                  }}
                >
                  <div className='h-full p-4 flex flex-col'>
                    <div className='flex items-start justify-between gap-3'>
                      <div className='min-w-0'>
                        <Typography.Title
                          heading={5}
                          ellipsis={{ rows: 1, showTooltip: true }}
                          style={{
                            margin: 0,
                            color: isPremium
                              ? 'rgba(248, 250, 252, 1)'
                              : undefined,
                          }}
                        >
                          {plan?.title || t('订阅套餐')}
                        </Typography.Title>
                        {plan?.subtitle && (
                          <Text
                            type='tertiary'
                            size='small'
                            ellipsis={{ rows: 1, showTooltip: true }}
                            style={{
                              display: 'block',
                              color: isPremium
                                ? 'rgba(203, 213, 225, 1)'
                                : undefined,
                            }}
                          >
                            {plan.subtitle}
                          </Text>
                        )}
                      </div>
                      {(isRecommended || isPremium) && (
                        <Tag
                          color={isPremium ? 'grey' : 'blue'}
                          shape='circle'
                          size='small'
                        >
                          <Sparkles size={10} className='mr-1' />
                          {t(isPremium ? '旗舰' : '推荐')}
                        </Tag>
                      )}
                    </div>

                    <div className='mt-5 flex items-end gap-1'>
                      <span
                        className='text-lg font-semibold'
                        style={{
                          color: isPremium
                            ? 'rgba(248, 250, 252, 1)'
                            : 'var(--semi-color-primary)',
                        }}
                      >
                        {symbol}
                      </span>
                      <span
                        className='text-4xl font-semibold leading-none'
                        style={{
                          color: isPremium
                            ? 'rgba(248, 250, 252, 1)'
                            : 'var(--semi-color-primary)',
                        }}
                      >
                        {displayPrice}
                      </span>
                      <Text
                        type='tertiary'
                        size='small'
                        style={{
                          color: isPremium
                            ? 'rgba(203, 213, 225, 1)'
                            : undefined,
                        }}
                      >
                        / {t('月')}
                      </Text>
                    </div>

                    <div className='mt-5 space-y-2'>
                      {benefits.map((benefit) => (
                        <div
                          key={benefit}
                          className='flex items-center gap-2 text-sm'
                          style={{
                            color: isPremium
                              ? 'rgba(226, 232, 240, 1)'
                              : 'var(--semi-color-text-1)',
                          }}
                        >
                          <CheckCircle2
                            size={14}
                            color={
                              isPremium
                                ? 'rgba(248, 250, 252, 0.92)'
                                : 'rgba(5, 150, 105, 1)'
                            }
                          />
                          <span>{benefit}</span>
                        </div>
                      ))}
                    </div>

                    <div className='mt-auto pt-4'>
                      <Divider margin={12} />
                      {(() => {
                        const tip = reachedLimit
                          ? t('已达到购买上限') + ` (${purchaseCount}/${limit})`
                          : '';
                        const subscribeButton = (
                          <Button
                            theme={isRecommended ? 'solid' : 'outline'}
                            type='primary'
                            className='flex-1'
                            disabled={reachedLimit}
                            onClick={() => {
                              if (!reachedLimit) {
                                openBuy(item);
                              }
                            }}
                          >
                            {reachedLimit ? t('已达上限') : t('立即订阅')}
                          </Button>
                        );
                        const taobaoButton = (
                          <Button
                            theme='outline'
                            type='warning'
                            className='flex-1'
                            icon={<ExternalLink size={14} />}
                            onClick={() => openRechargeLink(topUpLink)}
                          >
                            {t('淘宝购买')}
                          </Button>
                        );
                        const actions = (
                          <div className='grid grid-cols-2 gap-2'>
                            {reachedLimit ? (
                              <Tooltip content={tip} position='top'>
                                {subscribeButton}
                              </Tooltip>
                            ) : (
                              subscribeButton
                            )}
                            {taobaoButton}
                          </div>
                        );
                        return actions;
                      })()}
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>
        ) : (
          <Card className='!rounded-2xl' bodyStyle={{ padding: 24 }}>
            <div className='text-center text-[var(--semi-color-text-2)]'>
              {t('暂无可购买套餐')}
            </div>
          </Card>
        )}
      </div>

      <SubscriptionPurchaseModal
        t={t}
        visible={open}
        onCancel={closeBuy}
        selectedPlan={selectedPlan}
        paying={paying}
        selectedEpayMethod={selectedEpayMethod}
        setSelectedEpayMethod={setSelectedEpayMethod}
        epayMethods={epayMethods}
        enableOnlineTopUp={enableOnlineTopUp}
        enableStripeTopUp={enableStripeTopUp}
        enableCreemTopUp={enableCreemTopUp}
        purchaseLimitInfo={
          selectedPlan?.plan?.id
            ? {
                limit: Number(selectedPlan?.plan?.max_purchase_per_user || 0),
                count: getPlanPurchaseCount(selectedPlan?.plan?.id),
              }
            : null
        }
        onPayStripe={payStripe}
        onPayCreem={payCreem}
        onPayEpay={payEpay}
        topUpLink={topUpLink}
      />
    </>
  );
};

export default SubscriptionPlanCatalog;
