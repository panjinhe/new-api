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
  { quota: '50 刀', price: '10 元' },
  { quota: '100 刀', price: '18 元' },
  { quota: '200 刀', price: '35 元' },
  { quota: '500 刀', price: '80 元' },
];

const RechargeSupportCard = ({ compact = false, onGoTopup }) => {
  const openOfficialSite = () => {
    window.open(OFFICIAL_SITE, '_blank', 'noopener,noreferrer');
  };

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
              Q群：217637139
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
            <div className='w-full max-w-xl rounded-xl bg-[var(--semi-color-fill-0)] px-4 py-3'>
              <div className='flex items-center gap-2 text-sm font-medium'>
                <CircleDollarSign size={16} className='text-emerald-500' />
                定价与有效期
              </div>
              <div className='mt-3 overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-white'>
                <div className='grid grid-cols-2 bg-[var(--semi-color-fill-0)] text-xs font-medium text-[var(--semi-color-text-1)]'>
                  <div className='px-3 py-2'>额度</div>
                  <div className='border-l border-[var(--semi-color-border)] px-3 py-2'>
                    价格
                  </div>
                </div>
                {pricingItems.map((item, index) => (
                  <div
                    key={item.quota}
                    className='grid grid-cols-2 text-xs text-[var(--semi-color-text-2)]'
                  >
                    <div
                      className={`px-3 py-2 ${index !== 0 ? 'border-t border-[var(--semi-color-border)]' : ''}`}
                    >
                      {item.quota}
                    </div>
                    <div
                      className={`border-l border-[var(--semi-color-border)] px-3 py-2 ${index !== 0 ? 'border-t border-[var(--semi-color-border)]' : ''}`}
                    >
                      {item.price}
                    </div>
                  </div>
                ))}
              </div>
              <div className='mt-2 text-xs leading-6 text-[var(--semi-color-text-2)]'>
                量大价格可谈 10000刀+/月。
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
              <Text strong>购买兑换码私聊客服，Q群：217637139</Text>
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
