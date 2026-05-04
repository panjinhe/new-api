package middleware

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type turnstileCheckResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.TurnstileCheckEnabled {
			session := sessions.Default(c)
			turnstileChecked := session.Get("turnstile")
			if turnstileChecked != nil {
				c.Next()
				return
			}
			response := c.Query("turnstile")
			if response == "" {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile token 为空",
				})
				c.Abort()
				return
			}
			client := http.Client{Timeout: 10 * time.Second}
			rawRes, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
				"secret":   {common.TurnstileSecretKey},
				"response": {response},
				"remoteip": {c.ClientIP()},
			})
			if err != nil {
				common.SysLog("Turnstile siteverify request failed: " + err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile 校验服务暂时不可用，请稍后重试",
				})
				c.Abort()
				return
			}
			defer rawRes.Body.Close()
			if rawRes.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(io.LimitReader(rawRes.Body, 1024))
				common.SysLog("Turnstile siteverify returned status " + rawRes.Status + ": " + strings.TrimSpace(string(body)))
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile 校验服务暂时不可用，请稍后重试",
				})
				c.Abort()
				return
			}
			var res turnstileCheckResponse
			err = common.DecodeJson(rawRes.Body, &res)
			if err != nil {
				common.SysLog("Turnstile siteverify response decode failed: " + err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile 校验服务返回异常，请稍后重试",
				})
				c.Abort()
				return
			}
			if !res.Success {
				common.SysLog("Turnstile check failed: ip=" + c.ClientIP() +
					", hostname=" + res.Hostname +
					", action=" + res.Action +
					", cdata=" + res.CData +
					", challenge_ts=" + res.ChallengeTS +
					", error_codes=" + strings.Join(res.ErrorCodes, ","))
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile 校验失败，请刷新重试！",
				})
				c.Abort()
				return
			}
			session.Set("turnstile", true)
			err = session.Save()
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message": "无法保存会话信息，请重试",
					"success": false,
				})
				return
			}
		}
		c.Next()
	}
}
