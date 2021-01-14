package rpc

import (
	"LookForYou/service/upload/config"
	upProto "LookForYou/service/upload/proto"
	"context"
)

type Upload struct {
}

func (u *Upload) UploadEntry(
	ctx context.Context, req *upProto.ReqEntry, resp *upProto.RespEntry) error {
	resp.Entry = config.UploadEntry
	return nil
}
