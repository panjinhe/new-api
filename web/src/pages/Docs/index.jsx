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

import React from 'react';
import { Button, Tag } from '@douyinfe/semi-ui';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { builtInDocs } from '../../constants/docs.constants';

const Docs = () => {
  const { t } = useTranslation();

  return (
    <div className='mt-16 min-h-[calc(100svh-64px)] overflow-x-hidden bg-[radial-gradient(circle_at_top,_rgba(8,145,178,0.08),_transparent_40%),linear-gradient(180deg,_rgba(255,255,255,0.98),_rgba(248,250,252,0.92))] dark:bg-[radial-gradient(circle_at_top,_rgba(8,145,178,0.12),_transparent_38%),linear-gradient(180deg,_rgba(24,24,27,0.96),_rgba(15,23,42,0.92))]'>
      <div className='mx-auto max-w-6xl px-4 py-10 md:px-8 md:py-14'>
        <div className='mx-auto max-w-3xl text-center'>
          <div className='inline-flex items-center rounded-full border border-semi-color-border bg-white/75 px-4 py-1 text-sm text-semi-color-text-1 shadow-sm backdrop-blur dark:bg-black/20'>
            {t('教程及文档')}
          </div>
          <h1 className='mt-6 text-4xl font-bold leading-tight text-semi-color-text-0 md:text-5xl'>
            {t('教程及文档')}
          </h1>
          <p className='mx-auto mt-4 max-w-2xl text-base leading-7 text-semi-color-text-1 md:text-lg'>
            {t(
              '这里集中放接入教程、兼容方案与真实排障复盘，方便按问题场景快速查找处理方法。',
            )}
          </p>
        </div>

        <div className='mt-10 grid gap-6 md:grid-cols-2 xl:grid-cols-3'>
          {builtInDocs.map((doc) => (
            <article
              key={doc.slug}
              className='flex h-full flex-col rounded-[30px] border border-semi-color-border bg-white/85 p-6 shadow-[0_22px_70px_rgba(15,23,42,0.08)] backdrop-blur dark:bg-black/20'
            >
              <div className='flex items-center justify-between gap-3'>
                <div className='text-xs font-semibold uppercase tracking-[0.24em] text-cyan-600 dark:text-cyan-300'>
                  {t('精选教程')}
                </div>
                <div className='text-sm text-semi-color-text-2'>
                  {doc.updatedAt}
                </div>
              </div>

              <h2 className='mt-4 text-2xl font-bold leading-9 text-semi-color-text-0'>
                {doc.title}
              </h2>
              <p className='mt-3 flex-1 text-sm leading-7 text-semi-color-text-1 md:text-base'>
                {doc.summary}
              </p>

              <div className='mt-5 flex flex-wrap gap-2'>
                {doc.tags.map((tag) => (
                  <Tag key={tag} color='cyan' shape='circle'>
                    {tag}
                  </Tag>
                ))}
              </div>

              <div className='mt-6 rounded-2xl border border-semi-color-border bg-semi-color-fill-0 px-4 py-3 text-sm text-semi-color-text-2'>
                <div className='font-semibold text-semi-color-text-1'>
                  {doc.sourcePath}
                </div>
              </div>

              <div className='mt-6'>
                <Link to={`/docs/${doc.slug}`}>
                  <Button theme='solid' type='primary' className='!rounded-full px-6'>
                    {t('查看详情')}
                  </Button>
                </Link>
              </div>
            </article>
          ))}
        </div>
      </div>
    </div>
  );
};

export default Docs;
