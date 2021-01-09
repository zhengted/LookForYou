package mq

import "LookForYou/common"

// TransferData:转移队列中消息载体结构
type TransferData struct {
	FileHash      string
	CurLocation   string
	DestLocation  string
	DestStoreType common.StoreType
}
