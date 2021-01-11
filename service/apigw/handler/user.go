package handler

import (
	cmn "LookForYou/common"
	"LookForYou/service/account/proto"
	"LookForYou/util"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/micro/go-micro"
	"log"
	"net/http"
)

var (
	userCli proto.UserService
)

func init() {
	// 创建一个服务
	service := micro.NewService()
	// 初始化，解析命令行参数等
	service.Init()

	// 初始化一个rpcClient
	userCli = proto.NewUserService("go.micro.service.user", service.Client())
}

func SignupHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}

// DoSignupHandler:处理注册post请求
func DoSignupHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	resp, err := userCli.Signup(context.TODO(), &proto.ReqSignup{
		Username: username,
		Password: passwd,
	})
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": resp.Code,
		"msg":  resp.Message,
	})
}

func SigninHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}

func DoSigninHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	rpcResp, err := userCli.Signin(context.TODO(), &proto.ReqSignin{
		Username: username,
		Password: passwd,
	})
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	if rpcResp.Code != cmn.StatusOK {
		c.JSON(200, gin.H{
			"msg":  "登录失败",
			"code": rpcResp.Code,
		})
		return
	}

	cliResp := util.RespMsg{
		Code: int(cmn.StatusOK),
		Msg:  "登录成昆",
		Data: struct {
			Location string
			Username string
			Token    string
		}{
			Location: "/static/view/home.html",
			Username: username,
			Token:    rpcResp.Token,
		},
	}
	c.Data(http.StatusOK, "application/json", cliResp.JSONBytes())
}
