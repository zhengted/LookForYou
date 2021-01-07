package oss

import (
	cfg "LookForYou/config"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var ossCli *oss.Client

func Client() *oss.Client {
	if ossCli != nil {
		return ossCli
	}
	ossCli, err := oss.New(cfg.OSSEndPoint,
		cfg.OSSAccesskeyID, cfg.OSSAccesskeySecret)
	if err != nil {
		fmt.Println("create oss client:", err.Error())
		return nil
	}
	return ossCli
}

func Bucket() *oss.Bucket {
	cli := Client()
	if cli == nil {
		return nil
	}
	bucket, err := cli.Bucket(cfg.OSSBucket)
	if err != nil {
		fmt.Println("get bucket:", err.Error())
		return nil
	}
	return bucket
}
