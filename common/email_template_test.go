package common

import (
	"html"
	"strings"
	"testing"
)

func withEmailBrand(t *testing.T, systemName string, logo string) {
	t.Helper()
	oldSystemName := SystemName
	oldLogo := Logo
	SystemName = systemName
	Logo = logo
	t.Cleanup(func() {
		SystemName = oldSystemName
		Logo = oldLogo
	})
}

func TestBuildEmailVerificationTemplate(t *testing.T) {
	withEmailBrand(t, "AHE API", "")

	emailHTML := BuildEmailVerificationTemplate("123456", 10, "https://api.example.com")

	for _, want := range []string{"AHE API", "邮箱验证", "123456", "10 分钟内有效", "如果不是本人操作，请忽略"} {
		if !strings.Contains(emailHTML, want) {
			t.Fatalf("verification template should contain %q", want)
		}
	}
	if strings.Contains(emailHTML, "<img ") {
		t.Fatal("empty logo should not render an image tag")
	}
}

func TestBuildPasswordResetTemplate(t *testing.T) {
	withEmailBrand(t, "AHE API", "/logo.png")
	link := "https://api.example.com/user/reset?email=user%40example.com&token=abc123"

	emailHTML := BuildPasswordResetTemplate(link, 15, "https://api.example.com")
	escapedLink := html.EscapeString(link)

	for _, want := range []string{"密码重置", "重置密码", escapedLink, "15 分钟内有效", `src="https://api.example.com/logo.png"`} {
		if !strings.Contains(emailHTML, want) {
			t.Fatalf("password reset template should contain %q", want)
		}
	}
}

func TestResolveEmailLogoURL(t *testing.T) {
	tests := []struct {
		name          string
		logo          string
		serverAddress string
		want          string
	}{
		{name: "empty", logo: "", serverAddress: "https://api.example.com", want: ""},
		{name: "absolute", logo: "https://cdn.example.com/logo.png", serverAddress: "https://api.example.com", want: "https://cdn.example.com/logo.png"},
		{name: "relative with slash", logo: "/logo.png", serverAddress: "https://api.example.com", want: "https://api.example.com/logo.png"},
		{name: "relative without slash", logo: "logo.png", serverAddress: "https://api.example.com/", want: "https://api.example.com/logo.png"},
		{name: "invalid server", logo: "/logo.png", serverAddress: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEmailLogoURL(tt.logo, tt.serverAddress)
			if got != tt.want {
				t.Fatalf("resolveEmailLogoURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmailTemplateEscapesUserControlledValues(t *testing.T) {
	withEmailBrand(t, `<script>alert("brand")</script>`, "")
	link := `https://api.example.com/reset?next=<script>alert("x")</script>`

	emailHTML := BuildPasswordResetTemplate(link, 5, "https://api.example.com")

	if strings.Contains(emailHTML, `<script>`) {
		t.Fatal("template should escape script tags")
	}
	if !strings.Contains(emailHTML, "&lt;script&gt;") {
		t.Fatal("template should contain escaped script text")
	}
}

func TestBuildNotificationEmailTemplate(t *testing.T) {
	withEmailBrand(t, "AHE API", "https://cdn.example.com/logo.png")

	emailHTML := BuildNotificationEmailTemplate("额度提醒", `<p>剩余额度：<strong>100</strong></p>`, "https://api.example.com")

	for _, want := range []string{"额度提醒", "AHE API", `<strong>100</strong>`, `src="https://cdn.example.com/logo.png"`} {
		if !strings.Contains(emailHTML, want) {
			t.Fatalf("notification template should contain %q", want)
		}
	}
}
