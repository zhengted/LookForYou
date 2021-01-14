package main

import (
	cfg "LookForYou/config"
	"LookForYou/service/upload/route"
	upRpc "LookForYou/service/upload/rpc"
	upProto "LookForYou/service/upload/proto"
	"github.com/micro/go-micro"
	"log"
)

// 启动RPC的服务
func startRpcService() {
	service := micro.NewService(
		micro.Name("go.micro.service.upload"))
	service.Init()
	upProto.RegisterUploadServiceHandler(service.Server(),new(upRpc.Upload))
	if err := service.Run(); err != nil {
		log.Println(err.Error())
	}
}

// 启动API的服务
func startApiService() {
	router := route.Router()
	router.Run(cfg.UploadServiceHost)
}

func main() {
	go startRpcService()	// 会堵塞所以开一个线程

	startApiService()
}
