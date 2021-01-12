package handler

import (
	"LookForYou/common"
	cfg "LookForYou/config"
	dblayer "LookForYou/db"
	proto "LookForYou/service/account/proto"
	"LookForYou/util"
	"context"
	"errors"
	"fmt"
	"time"
)

type User struct {
}

func (u *User) Signup(ctx context.Context, req *proto.ReqSignup, resp *proto.RespSignup) error {
	username := req.Username
	passwd := req.Password
	if len(username) < 3 || len(passwd) < 5 {
		resp.Code = common.StatusParamInvalid
		resp.Message = "注册参数无效"
		return nil
	}
	encPasswd := util.Sha1([]byte(passwd + cfg.PWD_salt))
	suc := dblayer.UserSignup(username, encPasswd)
	if suc {
		resp.Code = common.StatusOK
		resp.Message = "注册成功"
	} else {
		resp.Code = common.StatusRegisterFailed
		resp.Message = "注册失败"
	}
	return nil
}

func (u *User) Signin(ctx context.Context, req *proto.ReqSignin, resp *proto.RespSignin) error {
	username := req.Username
	passwd := req.Password

	encPasswd := util.Sha1([]byte(passwd + cfg.PWD_salt))
	// 1. 校验用户名及密码
	pwdChecked := dblayer.UserSignin(username, encPasswd)
	if pwdChecked == false {
		resp.Code = common.StatusLoginFailed
		resp.Token = ""
		resp.Message = "login failed"
		return errors.New("login failed")
	}
	// 2. 生成访问凭证 token
	token := GenToken(username)
	upRes := dblayer.UpdateToken(username, token)
	if !upRes {
		resp.Code = common.StatusServerError
		resp.Token = token
		resp.Message = "Generate token Data"
		return errors.New("Generate token Data failed")
	}
	// 3. 登录成功后 重定向到首页
	resp.Code = common.StatusOK
	resp.Token = token
	resp.Message = "OK"
	return nil
}

func GenToken(username string) string {
	// md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + cfg.Token_salt))
	return tokenPrefix + ts[:8]
}

func (u *User) UserInfo(ctx context.Context, req *proto.ReqUserInfo, resp *proto.RespUserInfo) error {

	return nil
}

func (u *User) UserFiles(ctx context.Context, req *proto.ReqUserFile, resp *proto.RespUserFile) error {

	return nil
}

func (u *User) UserFileRename(ctx context.Context, req *proto.ReqUserFileRename, resp *proto.RespUserFileRename) error {

	return nil
}
