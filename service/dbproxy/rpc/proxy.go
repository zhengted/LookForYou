package rpc

import (
	"LookForYou/service/dbproxy/proto"
	"context"
)

type DBProxy struct {
}

func (db *DBProxy) ExecuteAction(ctx context.Context, req *proto.ReqExec, out *proto.RespExec) error {

	return nil
}
