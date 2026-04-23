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
import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';
import NotFound from '../NotFound';
import { getBuiltInDocBySlug } from '../../constants/docs.constants';

const DocsDetail = () => {
  const { t } = useTranslation();
  const { slug } = useParams();
  const doc = getBuiltInDocBySlug(slug);

  if (!doc) {
    return <NotFound />;
  }

  return (
    <div className='mt-16 min-h-[calc(100svh-64px)] overflow-x-hidden bg-[radial-gradient(circle_at_top,_rgba(8,145,178,0.08),_transparent_40%),linear-gradient(180deg,_rgba(255,255,255,0.98),_rgba(248,250,252,0.92))] dark:bg-[radial-gradient(circle_at_top,_rgba(8,145,178,0.12),_transparent_38%),linear-gradient(180deg,_rgba(24,24,27,0.96),_rgba(15,23,42,0.92))]'>
      <div className='mx-auto max-w-5xl px-4 py-10 md:px-8 md:py-14'>
        <Link to='/docs'>
          <Button theme='borderless' type='tertiary' className='!px-0'>
            {t('返回文档列表')}
          </Button>
        </Link>

        <section className='mt-4 rounded-[32px] border border-semi-color-border bg-white/88 p-6 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur dark:bg-black/20 md:p-8'>
          <div className='flex flex-wrap gap-2'>
            {doc.tags.map((tag) => (
              <Tag key={tag} color='cyan' shape='circle'>
                {tag}
              </Tag>
            ))}
          </div>

          <h1 className='mt-5 text-3xl font-bold leading-tight text-semi-color-text-0 md:text-4xl'>
            {doc.title}
          </h1>
          <p className='mt-4 text-base leading-8 text-semi-color-text-1'>
            {doc.summary}
          </p>

          <div className='mt-6 flex flex-wrap items-center gap-3 text-sm text-semi-color-text-2'>
            <span>{doc.updatedAt}</span>
            <span className='hidden md:inline'>/</span>
            <span className='font-mono'>{doc.sourcePath}</span>
          </div>
        </section>

        <section className='mt-6 rounded-[32px] border border-semi-color-border bg-white/90 p-6 shadow-[0_24px_80px_rgba(15,23,42,0.06)] backdrop-blur dark:bg-black/15 md:p-8'>
          <MarkdownRenderer content={doc.content} />
        </section>
      </div>
    </div>
  );
};

export default DocsDetail;
