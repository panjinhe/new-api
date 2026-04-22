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

import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Modal, Row, Spin, Space } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showInfo,
  showSuccess,
  showWarning,
} from '../../../helpers';

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    QuotaForInviter: '',
    QuotaForInvitee: '',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const quotaPerUnit = Number(props.options?.QuotaPerUnit || 0);
  const presetFiveDollarQuota =
    Number.isFinite(quotaPerUnit) && quotaPerUnit > 0
      ? Math.round(quotaPerUnit * 5)
      : 0;

  function buildRequestQueue() {
    const updateArray = compareObjects(inputs, inputsRow);
    return updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
  }

  async function persistSettings({ quietIfUnchanged = false } = {}) {
    const requestQueue = buildRequestQueue();
    if (!requestQueue.length) {
      if (!quietIfUnchanged) showWarning(t('你似乎并没有修改什么'));
      return true;
    }
    setLoading(true);
    try {
      const res = await Promise.all(requestQueue);
      if (requestQueue.length === 1) {
        if (res.includes(undefined)) return false;
      } else if (requestQueue.length > 1) {
        if (res.includes(undefined)) {
          showError(t('部分保存失败，请重试'));
          return false;
        }
      }
      showSuccess(t('保存成功'));
      await props.refresh();
      return true;
    } catch {
      showError(t('保存失败，请重试'));
      return false;
    } finally {
      setLoading(false);
    }
  }

  function onSubmit() {
    void persistSettings();
  }

  function applyFiveDollarPreset() {
    if (presetFiveDollarQuota <= 0) {
      showInfo(t('请先在通用设置中确认额度单位'));
      return;
    }
    const nextInputs = {
      ...inputs,
      QuotaForNewUser: String(presetFiveDollarQuota),
    };
    setInputs(nextInputs);
    refForm.current?.setValue('QuotaForNewUser', presetFiveDollarQuota);
  }

  function saveAndGrantTrialQuota() {
    const quota = parseInt(inputs.QuotaForNewUser, 10);
    if (!Number.isFinite(quota) || quota <= 0) {
      showError(t('新用户初始额度必须大于 0'));
      return;
    }

    Modal.confirm({
      title: t('确认补发现有用户体验额度？'),
      content: t(
        '将先保存当前额度设置，再给所有现有用户发放与“新用户初始额度”相同的体验额度，此操作会立即生效。',
      ),
      onOk: async () => {
        const saved = await persistSettings({ quietIfUnchanged: true });
        if (!saved) return;

        setLoading(true);
        try {
          const res = await API.post('/api/option/grant_quota_to_all_users', {
            quota,
          });
          const { success, message, data } = res.data;
          if (!success) {
            showError(message);
            return;
          }
          showSuccess(
            t('已为 {{count}} 个用户发放体验额度', {
              count: data?.affected_users || 0,
            }),
          );
          await props.refresh();
        } catch (error) {
          showError(error?.message || t('批量发放失败，请重试'));
        } finally {
          setLoading(false);
        }
      },
    });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);
  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('额度设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('新用户初始额度')}
                  field={'QuotaForNewUser'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForNewUser: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('请求预扣费额度')}
                  field={'PreConsumedQuota'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t('请求结束后多退少补')}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      PreConsumedQuota: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('邀请新用户奖励额度')}
                  field={'QuotaForInviter'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：2000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInviter: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('新用户使用邀请码奖励额度')}
                  field={'QuotaForInvitee'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={''}
                  placeholder={t('例如：1000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInvitee: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col>
                <Form.Switch
                  label={t('对免费模型启用预消耗')}
                  field={'quota_setting.enable_free_model_pre_consume'}
                  extraText={t(
                    '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                  )}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'quota_setting.enable_free_model_pre_consume': value,
                    })
                  }
                />
              </Col>
            </Row>

            <Row>
              <Space wrap>
                <Button size='default' onClick={applyFiveDollarPreset}>
                  {t('一键填入 5 美元额度')}
                </Button>
                <Button size='default' onClick={onSubmit}>
                  {t('保存额度设置')}
                </Button>
                <Button
                  theme='solid'
                  type='primary'
                  onClick={saveAndGrantTrialQuota}
                >
                  {t('保存并给所有现有用户补发')}
                </Button>
              </Space>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
