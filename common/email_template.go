package common

import (
	"fmt"
	"html"
	"strings"
)

const (
	emailPrimaryColor = "#2563eb"
	emailAccentColor  = "#10b981"
)

type emailTemplateData struct {
	Title         string
	PreviewText   string
	ContentHTML   string
	ServerAddress string
	Category      string
}

func BuildEmailVerificationTemplate(code string, validMinutes int, serverAddress string) string {
	escapedCode := html.EscapeString(code)
	content := fmt.Sprintf(`
<p style="margin:0 0 18px;color:#334155;font-size:16px;line-height:1.7;">您好，你正在进行 %s 邮箱验证。</p>
<div style="margin:22px 0 24px;padding:22px 18px;border:1px solid #bfdbfe;border-radius:8px;background:#eff6ff;text-align:center;">
  <div style="margin:0 0 10px;color:#2563eb;font-size:13px;font-weight:700;letter-spacing:0.08em;text-transform:uppercase;">验证码</div>
  <div style="font-family:Menlo,Consolas,'Courier New',monospace;font-size:34px;line-height:1.2;font-weight:800;letter-spacing:0.16em;color:#0f172a;">%s</div>
</div>
<p style="margin:0;color:#64748b;font-size:14px;line-height:1.7;">验证码 %d 分钟内有效。请勿向任何人泄露验证码，如果不是本人操作，请忽略本邮件。</p>`,
		html.EscapeString(SystemName), escapedCode, validMinutes)

	return buildBrandedEmail(emailTemplateData{
		Title:         "邮箱验证",
		PreviewText:   fmt.Sprintf("您的验证码是 %s，%d 分钟内有效。", code, validMinutes),
		ContentHTML:   content,
		ServerAddress: serverAddress,
		Category:      "账户安全邮件",
	})
}

func BuildPasswordResetTemplate(resetLink string, validMinutes int, serverAddress string) string {
	escapedLink := html.EscapeString(resetLink)
	content := fmt.Sprintf(`
<p style="margin:0 0 18px;color:#334155;font-size:16px;line-height:1.7;">您好，你正在进行 %s 密码重置。</p>
<p style="margin:0 0 24px;color:#64748b;font-size:14px;line-height:1.7;">点击下方按钮继续完成密码重置。链接 %d 分钟内有效，如果不是本人操作，请忽略本邮件。</p>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 24px;width:100%%;">
  <tr>
    <td align="center">
      <a href="%s" style="display:inline-block;padding:13px 24px;border-radius:8px;background:%s;color:#ffffff;font-size:15px;font-weight:700;text-decoration:none;">重置密码</a>
    </td>
  </tr>
</table>
<div style="padding:14px 16px;border:1px solid #e2e8f0;border-radius:8px;background:#f8fafc;">
  <p style="margin:0 0 8px;color:#64748b;font-size:13px;line-height:1.6;">如果按钮无法点击，请复制下面的链接到浏览器打开：</p>
  <a href="%s" style="color:%s;font-size:13px;line-height:1.6;word-break:break-all;text-decoration:underline;">%s</a>
</div>`,
		html.EscapeString(SystemName), validMinutes, escapedLink, emailPrimaryColor, escapedLink, emailPrimaryColor, escapedLink)

	return buildBrandedEmail(emailTemplateData{
		Title:         "密码重置",
		PreviewText:   fmt.Sprintf("你的密码重置链接 %d 分钟内有效。", validMinutes),
		ContentHTML:   content,
		ServerAddress: serverAddress,
		Category:      "账户安全邮件",
	})
}

func BuildNotificationEmailTemplate(title string, contentHTML string, serverAddress string) string {
	return buildBrandedEmail(emailTemplateData{
		Title:         title,
		PreviewText:   stripHTML(contentHTML),
		ContentHTML:   fmt.Sprintf(`<div style="color:#334155;font-size:15px;line-height:1.7;">%s</div>`, contentHTML),
		ServerAddress: serverAddress,
		Category:      "系统通知",
	})
}

func buildBrandedEmail(data emailTemplateData) string {
	title := strings.TrimSpace(data.Title)
	if title == "" {
		title = "通知"
	}
	brand := strings.TrimSpace(SystemName)
	if brand == "" {
		brand = "New API"
	}
	logoHTML := buildEmailLogoHTML(data.ServerAddress, brand)
	homeLink := buildEmailHomeLink(data.ServerAddress)
	category := strings.TrimSpace(data.Category)
	if category == "" {
		category = "系统邮件"
	}

	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f6f8fb;font-family:Lato,'Helvetica Neue',Arial,'Microsoft YaHei',sans-serif;color:#0f172a;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">%s</div>
  <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%%;background:#f6f8fb;">
    <tr>
      <td align="center" style="padding:32px 16px;">
        <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%%;max-width:600px;">
          <tr>
            <td style="padding:0 0 14px;">
              <table role="presentation" cellpadding="0" cellspacing="0" style="width:100%%;">
                <tr>
                  <td style="vertical-align:middle;">%s</td>
                  <td align="right" style="vertical-align:middle;color:#94a3b8;font-size:12px;">%s</td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="border-radius:8px;background:#ffffff;border:1px solid #e2e8f0;box-shadow:0 12px 30px rgba(15,23,42,0.06);overflow:hidden;">
              <div style="height:4px;background:linear-gradient(90deg,%s,%s);font-size:0;line-height:0;">&nbsp;</div>
              <div style="padding:32px 28px 30px;">
                <h1 style="margin:0 0 20px;color:#0f172a;font-size:24px;line-height:1.35;font-weight:800;">%s</h1>
                %s
              </div>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 6px 0;text-align:center;color:#94a3b8;font-size:12px;line-height:1.7;">
              <div>这封邮件由 %s 自动发送，请勿直接回复。</div>
              <div>如果不是本人操作，请忽略本邮件；为保证账户安全，请不要向任何人泄露验证码或链接。</div>
              %s
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(title),
		html.EscapeString(data.PreviewText),
		logoHTML,
		html.EscapeString(category),
		emailPrimaryColor,
		emailAccentColor,
		html.EscapeString(title),
		data.ContentHTML,
		html.EscapeString(brand),
		homeLink)
}

func buildEmailLogoHTML(serverAddress string, brand string) string {
	logoURL := resolveEmailLogoURL(Logo, serverAddress)
	escapedBrand := html.EscapeString(brand)
	if logoURL != "" {
		return fmt.Sprintf(`<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="vertical-align:middle;"><img src="%s" alt="%s" width="40" height="40" style="display:block;width:40px;height:40px;border-radius:999px;border:1px solid #e2e8f0;object-fit:cover;"></td><td style="padding-left:12px;vertical-align:middle;color:#0f172a;font-size:18px;font-weight:800;">%s</td></tr></table>`, html.EscapeString(logoURL), escapedBrand, escapedBrand)
	}
	initial := brandInitial(brand)
	return fmt.Sprintf(`<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="vertical-align:middle;"><div style="width:40px;height:40px;border-radius:999px;background:%s;color:#ffffff;text-align:center;line-height:40px;font-size:18px;font-weight:800;">%s</div></td><td style="padding-left:12px;vertical-align:middle;color:#0f172a;font-size:18px;font-weight:800;">%s</td></tr></table>`, emailPrimaryColor, html.EscapeString(initial), escapedBrand)
}

func resolveEmailLogoURL(logo string, serverAddress string) string {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return ""
	}
	lowerLogo := strings.ToLower(logo)
	if strings.HasPrefix(lowerLogo, "http://") || strings.HasPrefix(lowerLogo, "https://") {
		return logo
	}
	serverAddress = strings.TrimSpace(serverAddress)
	lowerServerAddress := strings.ToLower(serverAddress)
	if serverAddress == "" || !(strings.HasPrefix(lowerServerAddress, "http://") || strings.HasPrefix(lowerServerAddress, "https://")) {
		return ""
	}
	return strings.TrimRight(serverAddress, "/") + "/" + strings.TrimLeft(logo, "/")
}

func buildEmailHomeLink(serverAddress string) string {
	serverAddress = strings.TrimRight(strings.TrimSpace(serverAddress), "/")
	lowerServerAddress := strings.ToLower(serverAddress)
	if serverAddress == "" || !(strings.HasPrefix(lowerServerAddress, "http://") || strings.HasPrefix(lowerServerAddress, "https://")) {
		return ""
	}
	escapedServerAddress := html.EscapeString(serverAddress)
	return fmt.Sprintf(`<div style="margin-top:8px;"><a href="%s" style="color:%s;text-decoration:none;">访问 %s</a></div>`, escapedServerAddress, emailPrimaryColor, escapedServerAddress)
}

func brandInitial(brand string) string {
	brand = strings.TrimSpace(brand)
	if brand == "" {
		return "N"
	}
	for _, r := range brand {
		return strings.ToUpper(string(r))
	}
	return "N"
}

func stripHTML(value string) string {
	var builder strings.Builder
	inTag := false
	for _, r := range value {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				builder.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(builder.String())
}
