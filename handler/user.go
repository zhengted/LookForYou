package handler

import (
	dblayer "LookForYou/db"
	"LookForYou/util"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	pwd_salt   = "#890"
	token_salt = "_tokensalt"
)

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data, err := ioutil.ReadFile("./static/view/signup.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Failed to direct signup html, err:" + err.Error())
			return
		}
		w.Write(data)
		return
	}
	r.ParseForm()
	username := r.Form.Get("username")
	passwd := r.Form.Get("password")
	if len(username) < 3 || len(passwd) < 5 {
		w.Write([]byte("Invalid parameter"))
		return
	}
	encPasswd := util.Sha1([]byte(passwd + pwd_salt))
	suc := dblayer.UserSignup(username, encPasswd)
	if suc {
		w.Write([]byte("SUCCESS"))
	} else {
		w.Write([]byte("FAILED"))
	}
}

func SignInHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	password := r.Form.Get("password")
	encPasswd := util.Sha1([]byte(password + pwd_salt))
	// 1. 校验用户名及密码
	pwdChecked := dblayer.UserSignin(username, encPasswd)
	if pwdChecked == false {
		w.Write([]byte("FAILED"))
		return
	}
	// 2. 生成访问凭证 token
	token := GenToken(username)
	upRes := dblayer.UpdateToken(username, token)
	if !upRes {
		w.Write([]byte("FAILED"))
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
			Location: "http://" + r.Host + "/static/view/home.html",
			Username: username,
			Token:    token,
		},
	}
	w.Write(resp.JSONBytes())
}

func UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求参数
	r.ParseForm()
	username := r.Form.Get("username")
	token := r.Form.Get("token")

	// 2. 验证token是否有效
	if !IsTokenValid(token) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 3. 查询用户信息
	user, err := dblayer.GetUserInfo(username)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// 4. 组装并相应用户数据
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: user,
	}
	w.Write(resp.JSONBytes())
}

func GenToken(username string) string {
	// md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + token_salt))
	return tokenPrefix + ts[:8]
}

func IsTokenValid(token string) bool {
	// TODO: 判断token的时效性
	// TODO: 从数据库表中查询username对应的token信息
	// TODO: 对比两个token是否有效
	return true
}
