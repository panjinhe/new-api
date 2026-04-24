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
import { Button, Input, Modal, Tag } from '@douyinfe/semi-ui';
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
import NoticeModal from '../../components/layout/NoticeModal';
import { featuredBuiltInDocs } from '../../constants/docs.constants';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const qqGroupNumber = '217637139';
  const ccSwitchInstallUrl =
    'https://github.com/farion1231/cc-switch/releases/latest';
  const ccSwitchRepoUrl = 'https://github.com/farion1231/cc-switch';
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [contactModalVisible, setContactModalVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const featuredTutorials = featuredBuiltInDocs;

  const quickStartSteps = [
    {
      number: '01',
      title: t('注册账号'),
      description: t(
        '完成注册或登录后即可进入控制台，开始配置你的专属 Codex 接入环境。',
      ),
    },
    {
      number: '02',
      title: t('获取 API 密钥'),
      description: t(
        '在控制台一键创建令牌，支持多密钥管理，创建后即可作为 api-key 使用。',
      ),
    },
    {
      number: '03',
      title: t('替换 Base URL'),
      description: t(
        '将原有配置中的 Base URL 替换为本站地址，填入模型与密钥即可开始使用。',
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

  const handleCopyQQGroup = async () => {
    const ok = await copy(qqGroupNumber);
    if (ok) {
      showSuccess(t('QQ群号已复制到剪切板'));
    }
  };

  const openExternalLink = (url) => {
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      <Modal
        title={t('咨询更多')}
        visible={contactModalVisible}
        onCancel={() => setContactModalVisible(false)}
        footer={null}
        centered
      >
        <div className='space-y-4'>
          <div className='text-sm leading-7 text-semi-color-text-1'>
            {t('加群@群主可领取20刀兑换码')}
          </div>
          <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='text-xs font-semibold uppercase tracking-[0.24em] text-semi-color-text-2'>
              {t('QQ群号')}
            </div>
            <div className='mt-2 text-2xl font-bold tracking-[0.08em] text-semi-color-text-0'>
              {qqGroupNumber}
            </div>
          </div>
          <Button theme='solid' type='primary' block onClick={handleCopyQQGroup}>
            {t('复制群号')}
          </Button>
        </div>
      </Modal>
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='w-full overflow-x-hidden'>
          <div className='relative w-full overflow-x-hidden border-b border-semi-color-border min-h-[calc(100svh-64px)]'>
            <div className='blur-ball blur-ball-indigo' />
            <div className='blur-ball blur-ball-teal' />
            <div className='mx-auto flex min-h-[calc(100svh-64px)] max-w-6xl items-center justify-center px-4 py-16 md:px-8 md:py-20'>
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
                      '接入你的 Codex API 极其简单，打开网站后完成配置，最快 3 分钟即可开始编码。',
                    )}
                  </p>
                </div>

                <div className='mt-10 grid gap-5 text-left md:mt-12 md:grid-cols-3'>
                  {quickStartSteps.map((step) => (
                    <div
                      key={step.number}
                      className='rounded-[28px] border border-semi-color-border bg-white/80 p-6 shadow-[0_18px_50px_rgba(15,23,42,0.06)] backdrop-blur dark:bg-black/20'
                    >
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
                    <Input readonly value={serverAddress} />
                  </div>
                </div>

                <div className='mx-auto mt-6 max-w-2xl rounded-[28px] border border-semi-color-border bg-white/85 px-6 py-6 text-center shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:bg-black/25'>
                  <div className='text-xs font-semibold uppercase tracking-[0.24em] text-semi-color-text-2'>
                    {t('咨询更多')}
                  </div>
                  <div className='mt-3 text-base leading-7 text-semi-color-text-1 md:text-lg'>
                    {t('加群@群主可领取20刀兑换码')}
                  </div>
                  <div className='mt-5 text-sm text-semi-color-text-2'>
                    {t('QQ群号')}
                  </div>
                  <div className='mt-2 text-3xl font-bold tracking-[0.08em] text-semi-color-text-0'>
                    {qqGroupNumber}
                  </div>
                  <div className='mt-6 flex justify-center'>
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='!h-11 !rounded-full px-8'
                      onClick={() => setContactModalVisible(true)}
                    >
                      {t('咨询更多')}
                    </Button>
                  </div>
                </div>

                {featuredTutorials.length > 0 && (
                  <section className='mx-auto mt-6 max-w-4xl rounded-[32px] border border-semi-color-border bg-white/85 p-5 text-left shadow-[0_24px_70px_rgba(15,23,42,0.08)] backdrop-blur dark:bg-black/25 md:p-6'>
                    <div className='flex flex-col gap-4 border-b border-semi-color-border pb-6 md:flex-row md:items-start md:justify-between'>
                      <div className='max-w-2xl'>
                        <div className='inline-flex items-center rounded-full border border-semi-color-border bg-semi-color-fill-0 px-4 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-semi-color-text-2'>
                          {t('精选教程')}
                        </div>
                        <div className='mt-4 text-2xl font-bold text-semi-color-text-0 md:text-3xl'>
                          {t('教程及文档')}
                        </div>
                        <p className='mt-3 text-sm leading-7 text-semi-color-text-1 md:text-base'>
                          {t(
                            '这里会持续整理接入教程、兼容方案和真实排障复盘。',
                          )}
                        </p>
                      </div>
                      <div className='flex shrink-0'>
                        <Link to='/docs'>
                          <Button
                            size={isMobile ? 'default' : 'large'}
                            className='!h-11 !rounded-full px-6'
                          >
                            {t('查看全部文档')}
                          </Button>
                        </Link>
                      </div>
                    </div>

                    <div className='mt-5 space-y-4'>
                      {featuredTutorials.map((tutorial, index) => (
                        <article
                          key={tutorial.slug}
                          className='group rounded-[28px] border border-semi-color-border bg-semi-color-fill-0 p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-cyan-300 hover:bg-white hover:shadow-[0_18px_48px_rgba(8,145,178,0.10)] dark:hover:border-cyan-500/50 dark:hover:bg-black/30'
                        >
                          <div className='flex flex-col gap-4 md:flex-row md:items-start md:justify-between'>
                            <div className='min-w-0 flex-1'>
                              <div className='flex flex-wrap items-center gap-3'>
                                <span className='text-xs font-semibold uppercase tracking-[0.24em] text-cyan-600 dark:text-cyan-300'>
                                  {String(index + 1).padStart(2, '0')}
                                </span>
                                <span className='text-sm text-semi-color-text-2'>
                                  {tutorial.updatedAt}
                                </span>
                              </div>
                              <h3 className='mt-3 text-xl font-bold leading-8 text-semi-color-text-0 md:text-2xl'>
                                {tutorial.title}
                              </h3>
                              <p className='mt-3 text-sm leading-7 text-semi-color-text-1 md:text-base'>
                                {tutorial.summary}
                              </p>
                              <div className='mt-4 flex flex-wrap gap-2'>
                                {tutorial.tags.map((tag) => (
                                  <Tag key={tag} color='cyan' shape='circle'>
                                    {tag}
                                  </Tag>
                                ))}
                              </div>
                            </div>
                            <div className='shrink-0 md:pt-8'>
                              <Link to={`/docs/${tutorial.slug}`}>
                                <Button
                                  theme={index === 0 ? 'solid' : 'light'}
                                  type='primary'
                                  size={isMobile ? 'default' : 'large'}
                                  className='!h-11 !w-full !rounded-full px-6 md:!w-auto'
                                >
                                  {t('查看详情')}
                                </Button>
                              </Link>
                            </div>
                          </div>
                        </article>
                      ))}
                    </div>
                  </section>
                )}
              </div>
            </div>
          </div>
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
