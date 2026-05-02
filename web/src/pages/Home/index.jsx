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

import React, { useContext, useEffect, useState } from 'react';
import { Button, Input } from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconPlay,
  IconCopy,
  IconExternalOpen,
  IconGithubLogo,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const qqGroupNumber = '217637139';
  const qqGroupQrCodeUrl = '/qq-group-qr.png';
  const ccSwitchInstallUrl =
    'https://github.com/farion1231/cc-switch/releases/latest';
  const ccSwitchRepoUrl = 'https://github.com/farion1231/cc-switch';
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const systemName = statusState?.status?.system_name || 'aheapi';
  const serviceAdvantageCards = [
    {
      code: t('01'),
      label: t('快响应'),
      title: t('90% 请求 2 秒内返回首字'),
      description: t('优化入口访问与上游调度，减少等待，降低日常调用延迟。'),
      visual: 'latency',
    },
    {
      code: t('02'),
      label: t('Plus / Pro 号池'),
      title: t('无 Free / 无 Team 号'),
      description: t(
        '优先使用 Plus / Pro 高权益资源池，减少低权益账号带来的排队和波动。',
      ),
      visual: 'pool',
    },
    {
      code: t('03'),
      label: t('配置简单'),
      title: t('几分钟完成接入'),
      description: t(
        '创建令牌、复制 Base URL 或一键导入 CC Switch，路径清晰。',
      ),
      visual: 'setup',
    },
  ];

  const quickStartSteps = [
    {
      number: '01',
      title: t('注册账号'),
      description: t(
        '完成注册或登录后即可进入控制台，开始配置你的专属 API 接入环境。',
      ),
    },
    {
      number: '02',
      title: t('创建令牌'),
      description: t(
        '在控制台一键创建令牌，支持多密钥管理，创建后即可作为 API Key 使用。',
      ),
    },
    {
      number: '03',
      title: t('导入 CC Switch'),
      description: t(
        '点击令牌页的 CC Switch 导入按钮，自动填入 Base URL、模型和 API Key。',
      ),
    },
  ];

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  const openExternalLink = (url) => {
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  return (
    <div className='w-full overflow-x-hidden'>
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='w-full overflow-x-hidden pt-16'>
          <section className='home-service-hero relative w-full overflow-hidden border-b border-semi-color-border'>
            <div className='home-hero-grid' />
            <div className='mx-auto max-w-6xl px-4 py-9 md:px-8 md:py-11 lg:min-h-[540px] lg:py-12'>
              <div className='home-hero-copy mx-auto max-w-4xl text-center'>
                <div className='inline-flex items-center border-b border-semi-color-border pb-2 text-sm font-semibold text-semi-color-text-1'>
                  {t('API 稳定接入服务')}
                </div>
                <h1 className='mx-auto mt-6 max-w-3xl text-5xl font-bold leading-tight text-semi-color-text-0 md:text-6xl lg:text-7xl'>
                  {systemName}
                </h1>
                <div className='mt-4 text-2xl font-bold leading-tight text-semi-color-text-0 md:text-3xl'>
                  {t('稳定快速的 API 服务')}
                </div>
                <p className='mx-auto mt-5 max-w-2xl text-base leading-8 text-semi-color-text-1 md:text-lg'>
                  {t(
                    '快响应、Plus / Pro 号池、配置简单，打开网站就能清楚知道怎么接入，日常调用更省心。',
                  )}
                </p>
                <div className='mt-8 flex flex-col justify-center gap-3 sm:flex-row sm:items-center'>
                  <Link to='/console/token'>
                    <Button
                      theme='solid'
                      type='primary'
                      size={isMobile ? 'default' : 'large'}
                      className='!h-12 !w-full !rounded-full px-7 sm:!w-auto'
                      icon={<IconPlay />}
                    >
                      {t('立即开始配置')}
                    </Button>
                  </Link>
                </div>
              </div>

              <div
                className='mt-8 grid gap-5 text-left md:mt-10 md:grid-cols-3'
                aria-label={t('核心优势')}
              >
                {serviceAdvantageCards.map((advantage, index) => (
                  <article
                    key={advantage.label}
                    className='home-service-card'
                    style={{ animationDelay: `${index * 90 + 120}ms` }}
                  >
                    <div className='home-service-card-top'>
                      <span>{advantage.code}</span>
                      <strong>{advantage.label}</strong>
                    </div>
                    <div className='home-service-card-visual'>
                      {advantage.visual === 'latency' && (
                        <div className='home-card-latency'>
                          <span />
                          <span />
                          <span />
                        </div>
                      )}
                      {advantage.visual === 'pool' && (
                        <div className='home-card-pool'>
                          <div className='home-card-pool-active'>
                            {t('PLUS / PRO')}
                          </div>
                          <div>{t('FREE')} 0</div>
                          <div>{t('TEAM')} 0</div>
                        </div>
                      )}
                      {advantage.visual === 'setup' && (
                        <div className='home-card-setup'>
                          <span>{t('Base URL')}</span>
                          <i />
                          <span>{t('Token')}</span>
                          <i />
                          <span>{t('CC Switch')}</span>
                        </div>
                      )}
                    </div>
                    <h2>{advantage.title}</h2>
                    <p>{advantage.description}</p>
                  </article>
                ))}
              </div>
            </div>
          </section>

          <section className='relative w-full overflow-x-hidden border-b border-semi-color-border bg-semi-color-bg-0'>
            <div className='mx-auto flex max-w-6xl items-center justify-center px-4 py-10 md:px-8 md:py-12'>
              <div className='w-full text-center'>
                <div className='mx-auto max-w-3xl'>
                  <div className='inline-flex items-center rounded-full border border-semi-color-border bg-white/75 px-4 py-1 text-sm text-semi-color-text-1 shadow-sm backdrop-blur dark:bg-black/20'>
                    {t('三步快速开始')}
                  </div>
                  <h1 className='mt-6 text-4xl font-bold leading-tight text-semi-color-text-0 md:text-5xl lg:text-6xl'>
                    {t('三步即可开始')}
                  </h1>
                  <p className='mx-auto mt-4 max-w-2xl text-base leading-7 text-semi-color-text-1 md:text-lg'>
                    {t(
                      '接入 API 极其简单，打开网站后完成配置，最快 3 分钟即可开始使用。',
                    )}
                  </p>
                </div>

                <div className='mt-8 grid gap-5 text-left md:mt-10 md:grid-cols-3'>
                  {quickStartSteps.map((step) => (
                    <div key={step.number} className='home-flow-card'>
                      <div className='text-sm font-semibold tracking-[0.22em] text-cyan-500'>
                        {step.number}
                      </div>
                      <div className='mt-5 text-2xl font-bold text-semi-color-text-0'>
                        {step.title}
                      </div>
                      <p className='mt-3 text-sm leading-7 text-semi-color-text-1 md:text-base'>
                        {step.description}
                      </p>
                    </div>
                  ))}
                </div>

                <div className='mx-auto mt-8 max-w-4xl rounded-[32px] border border-cyan-200/80 bg-cyan-500/[0.08] p-5 text-left shadow-[0_18px_50px_rgba(8,145,178,0.12)] backdrop-blur dark:border-cyan-500/30 dark:bg-cyan-500/10 md:mt-10 md:p-6'>
                  <div className='flex flex-col gap-4 md:flex-row md:items-center md:justify-between'>
                    <div className='max-w-2xl'>
                      <div className='text-xs font-semibold uppercase tracking-[0.24em] text-cyan-600 dark:text-cyan-300'>
                        {t('推荐接入')}
                      </div>
                      <div className='mt-2 text-2xl font-bold text-semi-color-text-0'>
                        {t('使用 CC Switch 一键配置')}
                      </div>
                      <p className='mt-3 text-sm leading-7 text-semi-color-text-1 md:text-base'>
                        {t(
                          '登录后进入控制台创建令牌，点击 CC Switch，即可自动填入 Base URL、模型和 API Key。',
                        )}
                      </p>
                      <div className='mt-3 text-sm leading-6 text-semi-color-text-2'>
                        {isDemoSiteMode && statusState?.status?.version
                          ? `${t('当前站点版本')} ${statusState.status.version}`
                          : t(
                              '如果工具不支持一键配置，也可以使用下方 Base URL 手动接入。',
                            )}
                      </div>
                    </div>
                    <div className='grid gap-3 md:w-auto'>
                      <Link to='/console/token'>
                        <Button
                          theme='solid'
                          type='primary'
                          size={isMobile ? 'default' : 'large'}
                          className='!h-11 !w-full !rounded-full px-6'
                          icon={<IconPlay />}
                        >
                          {t('前往令牌页')}
                        </Button>
                      </Link>
                    </div>
                  </div>
                  <div className='mt-4 flex flex-wrap items-center gap-2 border-t border-cyan-200/70 pt-4 text-sm text-semi-color-text-1 dark:border-cyan-500/20'>
                    <span>{t('还没安装 CC Switch？')}</span>
                    <Button
                      theme='borderless'
                      type='tertiary'
                      icon={<IconExternalOpen />}
                      onClick={() => openExternalLink(ccSwitchInstallUrl)}
                    >
                      {t('安装 CC Switch')}
                    </Button>
                    <Button
                      theme='borderless'
                      type='tertiary'
                      icon={<IconGithubLogo />}
                      onClick={() => openExternalLink(ccSwitchRepoUrl)}
                    >
                      {t('查看 GitHub 仓库')}
                    </Button>
                    <span className='w-full text-xs leading-5 text-semi-color-text-2 md:w-auto md:text-sm'>
                      {t(
                        '如果 GitHub 打不开或下载失败，请加入 QQ 群，在群文件中下载 CC Switch 安装包。',
                      )}
                    </span>
                  </div>
                  <div className='mt-4 rounded-2xl border border-cyan-200/70 bg-white/70 p-4 dark:border-cyan-500/20 dark:bg-black/20'>
                    <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
                      <div className='text-xs font-semibold uppercase tracking-[0.24em] text-semi-color-text-2'>
                        {t('手动配置备用')}
                      </div>
                      <Button
                        type='tertiary'
                        size='small'
                        icon={<IconCopy />}
                        onClick={handleCopyBaseURL}
                      >
                        {t('复制 Base URL')}
                      </Button>
                    </div>
                    <Input readOnly value={serverAddress} />
                  </div>
                </div>

                <div className='mx-auto mt-6 max-w-4xl overflow-hidden rounded-[28px] border border-semi-color-border bg-white/85 p-5 text-center shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:bg-black/25 md:p-7 md:text-left'>
                  <div className='grid items-stretch gap-5 md:grid-cols-[minmax(0,1fr)_260px] md:gap-7'>
                    <div className='flex min-w-0 flex-col justify-center'>
                      <div className='text-sm font-medium text-semi-color-text-2'>
                        {t('QQ 咨询群')}
                      </div>
                      <div className='mt-3 text-lg font-semibold leading-8 text-semi-color-text-0 md:text-xl'>
                        {t('加群@群主可领取20刀兑换码')}
                      </div>
                      <div className='mx-auto mt-5 w-full max-w-[360px] rounded-2xl border border-semi-color-border bg-semi-color-fill-0 px-5 py-4 text-center md:mx-0 md:max-w-[420px] md:px-6 md:py-5 md:text-left'>
                        <div className='text-sm text-semi-color-text-2'>
                          {t('QQ群号')}
                        </div>
                        <div className='mt-2 text-3xl font-bold tracking-[0.1em] text-semi-color-text-0 sm:text-4xl'>
                          {qqGroupNumber}
                        </div>
                      </div>
                    </div>
                    <div className='mx-auto flex w-full max-w-[260px] flex-col items-center justify-center rounded-[26px] border border-semi-color-border bg-white p-4 shadow-sm md:mx-0 md:justify-self-end dark:bg-white'>
                      <img
                        src={qqGroupQrCodeUrl}
                        alt={t('QQ 群二维码')}
                        className='aspect-square w-full max-w-[220px] rounded-2xl object-cover'
                      />
                      <div className='mt-3 text-center text-sm font-medium text-slate-600'>
                        {t('扫码加入QQ群')}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
