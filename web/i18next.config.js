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

import { defineConfig } from 'i18next-cli';

/** @type {import('i18next-cli').I18nextToolkitConfig} */
export default defineConfig({
  locales: ['zh-CN', 'en'],
  extract: {
    input: ['src/**/*.{js,jsx,ts,tsx}'],
    ignore: ['src/i18n/**/*'],
    output: 'src/i18n/locales/{{language}}.json',
    ignoredAttributes: [
      'accept',
      'align',
      'allow',
      'alt',
      'aria-label',
      'autoComplete',
      'cacheKey',
      'checkedText',
      'className',
      'clipRule',
      'color',
      'crossOrigin',
      'd',
      'data-index',
      'data-name',
      'data-testid',
      'data-type',
      'defaultActiveKey',
      'dir',
      'direction',
      'editorType',
      'field',
      'filterMode',
      'fill',
      'fillRule',
      'height',
      'hoverStyle',
      'htmlType',
      'id',
      'itemStyle',
      'itemKey',
      'key',
      'keyPrefix',
      'language',
      'labelPosition',
      'layout',
      'margin',
      'maxHeight',
      'mode',
      'name',
      'okType',
      'overflow',
      'path',
      'placement',
      'position',
      'rel',
      'role',
      'rowKey',
      'sandbox',
      'searchPosition',
      'selectedStyle',
      'shape',
      'size',
      'spacing',
      'stroke',
      'stroke-width',
      'style',
      'theme',
      'title',
      'trigger',
      'uploadTrigger',
      'uncheckedText',
      'validateStatus',
      'value',
      'variant',
      'viewBox',
      'width',
    ],
    sort: true,
    disablePlurals: false,
    removeUnusedKeys: false,
    nsSeparator: false,
    keySeparator: false,
    mergeNamespaces: true,
  },
});
