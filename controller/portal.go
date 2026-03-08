package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/forbearing/gst/authn/jwt"
	"github.com/forbearing/gst/config"
	. "github.com/forbearing/gst/response"
	"github.com/gin-gonic/gin"
)

type portal struct{}

var Portal = new(portal)

func (p *portal) Tick(c *gin.Context) {
	feishuAuthURL := "https://open.feishu.cn/open-apis/authen/v1/user_auth_page_beta?app_id=%s&redirect_uri="
	targetURL := "%s/api/feishu/qrlogin"
	// redirectURL := fmt.Sprintf(feishuAuthUrl, config.App.FeishuConfig.AppID) + url.QueryEscape(fmt.Sprintf(targetUrl, config.App.ServerConfig.Domain))
	redirectURL := fmt.Sprintf(feishuAuthURL, config.App.Feishu.AppID) + url.QueryEscape(fmt.Sprintf(targetURL, config.App.Server.Domain))
	if config.App.Mode == config.Dev {
		redirectURL = fmt.Sprintf(feishuAuthURL, config.App.Feishu.AppID) + url.QueryEscape(fmt.Sprintf(targetURL, "http://172.31.8.8:8001"))
	}
	fmt.Println("============= redirect: ", redirectURL)

	header := c.Request.Header.Get("Authorization")
	if len(header) == 0 {
		// ResponseJSON(c, CodeNeedLogin)
		JSON(c, CodeSuccess, gin.H{
			"redirect": redirectURL,
		})
		return
	}

	// 按空格分割
	items := strings.SplitN(header, " ", 2)
	if len(items) != 2 {
		// ResponseJSON(c, CodeInvalidToken)
		JSON(c, CodeSuccess, gin.H{
			"redirect": redirectURL,
		})
		return
	}
	if items[0] != "Bearer" {
		// ResponseJSON(c, CodeInvalidToken)
		JSON(c, CodeSuccess, gin.H{
			"redirect": redirectURL,
		})
		return
	}

	// items[1] 是获取到的 tokenString, 我们使用之前定义好的解析 jwt 的函数来解析它
	if _, err := jwt.ParseToken(items[1]); err != nil {
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		JSON(c, CodeSuccess, gin.H{
			"redirect": redirectURL,
		})
		return
	}
	JSON(c, CodeSuccess, gin.H{
		"redirect": config.App.Domain,
	})
}
