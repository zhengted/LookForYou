package main

import (
	"LookForYou/config"
	dblayer "LookForYou/db"
	"LookForYou/mq"
	"LookForYou/store/oss"
	"bufio"
	"encoding/json"
	"log"
	"os"
)

func ProcessTransfer(msg []byte) bool {
	// 1. 解析msg
	pubData := mq.TransferData{}
	err := json.Unmarshal(msg, pubData)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	// 2. 根据临时存储路径
	filed, err := os.Open(pubData.CurLocation)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	// 3. 通过文件句柄将文件内容读出来并上传到OSS
	err = oss.Bucket().PutObject(
		pubData.DestLocation,
		bufio.NewReader(filed),
	)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	// 4. 更新文件的存储路径到文件表
	return dblayer.UpdateFileLocation(pubData.FileHash, pubData.DestLocation)

}

func main() {
	mq.StartConsume(
		config.TransOSSQueueName,
		"transfer_oss",
		ProcessTransfer,
	)
}
