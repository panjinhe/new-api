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

import claudeCodeCodexCompatContent from '../../../docs/installation/claude-code-codex-compat.zh-CN.md?raw';
import codexPro20xCacheHitFixContent from '../../../docs/installation/codex-pro20x-cache-hit-fix.zh-CN.md?raw';

export const builtInDocs = [
  {
    slug: 'codex-pro20x-cache-hit-fix',
    title: 'Codex-pro20x 缓存低命中排查与修复记录',
    summary:
      '复盘 Codex-pro20x-1 连续缓存读为 0 的排查过程：如何用 prefix hash、渠道对照、上游 usage 数据定位到 Session_id 缺失，并通过代码修复恢复缓存命中。',
    tags: ['Codex-pro20x', '缓存命中', '使用日志', '排障记录'],
    sourcePath: 'docs/installation/codex-pro20x-cache-hit-fix.zh-CN.md',
    updatedAt: '2026-04-24',
    featured: true,
    content: codexPro20xCacheHitFixContent.trim(),
  },
  {
    slug: 'claude-code-codex-compat',
    title: 'Claude Code 接入 Codex/GPT-5.4 的实现与排障记录',
    summary:
      '复盘我们如何补齐 /v1/messages 兼容层、把 claude-opus-4-6 映射到 gpt-5.4，并解决 CCSwitch、登录态、Token 与部署链路中的典型问题。',
    tags: ['Claude Code', 'Codex', 'GPT-5.4', '排障记录'],
    sourcePath: 'docs/installation/claude-code-codex-compat.zh-CN.md',
    updatedAt: '2026-04-24',
    featured: true,
    content: claudeCodeCodexCompatContent.trim(),
  },
];

export const featuredBuiltInDocs = builtInDocs.filter((doc) => doc.featured);

export const hasBuiltInDocs = builtInDocs.length > 0;

export const getBuiltInDocBySlug = (slug) =>
  builtInDocs.find((doc) => doc.slug === slug);
