package handler

import (
	"LookForYou/common"
	"LookForYou/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

func HTTPInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")

		if len(username) < 3 || !IsTokenValid(token, username) {
			c.Abort() // 中间件校验失败 通知ginEngine后续不再执行
			resp := util.NewRespMsg(
				int(common.StatusInvalidToken),
				"token无效",
				nil,
			)
			c.JSON(http.StatusOK, resp)
			return
		}
		c.Next()
	}
}
