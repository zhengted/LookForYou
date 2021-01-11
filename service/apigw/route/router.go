package route

import (
	"LookForYou/service/apigw/handler"
	"github.com/gin-gonic/gin"
)

// 网关API路由
func Router() *gin.Engine {
	router := gin.Default()

	router.Static("/static/", "./static")
	router.GET("/user/signup", handler.SignupHandler)
	router.POST("/user/signup", handler.DoSignupHandler)

	return router
}
