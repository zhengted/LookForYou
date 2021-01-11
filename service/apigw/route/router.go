package route

import (
	"LookForYou/service/apigw/handler"
	"github.com/gin-gonic/gin"
)

// 网关API路由
func Router() *gin.Engine {
	router := gin.Default()

	router.Static("/static/", "./static")

	// 用户注册
	router.GET("/user/signup", handler.SignupHandler)
	router.POST("/user/signup", handler.DoSignupHandler)

	// 用户登录
	router.GET("/user/signin", handler.SigninHandler)
	router.POST("/user/signin", handler.DoSigninHandler)

	return router
}
