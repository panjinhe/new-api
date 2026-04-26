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

export const DEFAULT_RECHARGE_LINK =
  'https://item.taobao.com/item.htm?id=1045334787699&spm=a213gs.v2success.0.0.24244831C2XbNf';

export const getRechargeLink = (topUpLink = '') => {
  if (typeof topUpLink !== 'string') {
    return DEFAULT_RECHARGE_LINK;
  }

  return topUpLink.trim() || DEFAULT_RECHARGE_LINK;
};

export const openRechargeLink = (topUpLink = '') => {
  if (typeof window === 'undefined') return;
  window.open(getRechargeLink(topUpLink), '_blank', 'noopener,noreferrer');
};
