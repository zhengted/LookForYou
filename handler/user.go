package handler

import (
	dblayer "LookForYou/db"
	"LookForYou/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

const (
	pwd_salt   = "#890"
	token_salt = "_tokensalt"
)

func SignupHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}

// DoSignupHandler:处理注册post请求
func DoSignupHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	if len(username) < 3 || len(passwd) < 5 {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Invalid parameter",
			"code": -1,
		})
		return
	}
	encPasswd := util.Sha1([]byte(passwd + pwd_salt))
	suc := dblayer.UserSignup(username, encPasswd)
	if suc {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup succeeded",
			"code": 0,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup failed",
			"code": -2,
		})
	}
}

func SignInHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}

// DoSignInHandler:响应登录页面
func DoSignInHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")

	encPasswd := util.Sha1([]byte(passwd + pwd_salt))
	// 1. 校验用户名及密码
	pwdChecked := dblayer.UserSignin(username, encPasswd)
	if pwdChecked == false {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Login Failed",
			"code": -1,
		})
		return
	}
	// 2. 生成访问凭证 token
	token := GenToken(username)
	upRes := dblayer.UpdateToken(username, token)
	if !upRes {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Generate token failed",
			"code": -2,
		})
		return
	}
	// 3. 登录成功后 重定向到首页
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: struct {
			Location string
			Username string
			Token    string
		}{
			Location: "/static/view/home.html",
			Username: username,
			Token:    token,
		},
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
}

// UserInfoHandler ： 查询用户信息
func UserInfoHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	//	token := c.Request.FormValue("token")

	// 2. 查询用户信息
	user, err := dblayer.GetUserInfo(username)
	if err != nil {
		c.JSON(http.StatusForbidden,
			gin.H{})
		return
	}

	// 3. 组装并且响应用户数据
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: user,
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
}

func GenToken(username string) string {
	// md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + token_salt))
	return tokenPrefix + ts[:8]
}

const ONEDAYTIME = 24 * 60 * 60

func IsTokenValid(token string, username string) bool {
	if len(token) != 40 {
		return false
	}

	// 判断token的时效性
	tokenTS := token[:8]
	if util.Hex2Dec(tokenTS) < time.Now().Unix()-ONEDAYTIME {
		return false
	}
	// 从数据库表中查询username对应的token信息
	tokenDB, err := dblayer.GetTokenFromDB(username)
	if err != nil {
		return false
	}
	// 对比两个token是否有效
	if tokenDB != token {
		log.Println("token not same", tokenDB, token)
		return false
	}
	return true
}
