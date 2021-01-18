package rpc

import (
	cfg "LookForYou/service/download/config"
	dlProto "LookForYou/service/download/proto"
	"context"
)

// Dwonload :download结构体
type Download struct{}

// DownloadEntry : 获取下载入口
func (u *Download) DownloadEntry(
	ctx context.Context,
	req *dlProto.ReqEntry,
	res *dlProto.RespEntry) error {

	res.Entry = cfg.DownloadEntry
	return nil
}
