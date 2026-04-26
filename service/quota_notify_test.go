package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func TestBuildQuotaNotifyContentEmailIncludesRechargeSupport(t *testing.T) {
	originalServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://example.com/"
	t.Cleanup(func() {
		system_setting.ServerAddress = originalServerAddress
	})

	content := buildQuotaNotifyContent("您的额度即将用尽", 0, dto.NotifyTypeEmail)

	for _, want := range []string{
		"您的额度即将用尽",
		"https://example.com/console/topup",
		"淘宝充值（复制链接打开淘宝）",
		quotaNotifyTaobaoURL,
		quotaNotifyTaobaoCommand,
		quotaNotifyTaobaoTitle,
		quotaNotifySupportQQGroup,
		"https://example.com" + quotaNotifyPromoImagePath,
		"<img src=",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("quota notify content missing %q: %s", want, content)
		}
	}
}

func TestBuildQuotaNotifyContentPushUsesPlainText(t *testing.T) {
	content := buildQuotaNotifyContent("您的额度即将用尽", 0, dto.NotifyTypeBark)

	if strings.Contains(content, "<img") || strings.Contains(content, "<a ") {
		t.Fatalf("push quota notify content should be plain text: %s", content)
	}
	for _, want := range []string{quotaNotifyTaobaoURL, quotaNotifySupportQQGroup} {
		if !strings.Contains(content, want) {
			t.Fatalf("push quota notify content missing %q: %s", want, content)
		}
	}
}
