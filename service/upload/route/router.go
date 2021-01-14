package route

import (
	"LookForYou/service/upload/api"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 网关API路由
func Router() *gin.Engine {
	router := gin.Default()

	router.Static("/static/", "./static")

	router.POST("/file/upload", api.DoUploadHandler)

	// 使用gin插件支持跨域请求
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"}, // []string{"http://localhost:8080"},
		AllowMethods:  []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Range", "x-requested-with", "content-Type"},
		ExposeHeaders: []string{"Content-Length", "Accept-Ranges", "Content-Range", "Content-Disposition"},
		// AllowCredentials: true,
	}))

	// Use之后的所有handler都会经过拦截器进行token校验

	// 文件上传相关接口
	router.POST("/file/upload", api.DoUploadHandler)
	// 秒传接口
	router.POST("/file/fastupload", api.TryFastUploadHandler)

	// 分块上传接口
	router.POST("/file/mpupload/init", api.InitialMultipartUploadHandler)
	router.POST("/file/mpupload/uppart", api.UploadPartHandler)
	router.POST("/file/mpupload/complete", api.CompleteUploadHandler)

	return router
}
