package handler

import (
	"LookForYou/common"
	cfg "LookForYou/config"
	dblayer "LookForYou/db"
	"LookForYou/service/account/proto"
	"LookForYou/util"
	"context"
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
