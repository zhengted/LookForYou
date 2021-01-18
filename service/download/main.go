package main

import (
	"fmt"
	"time"

	micro "github.com/micro/go-micro"

	"LookForYou/config"
	cfg "LookForYou/service/download/config"
	dlProto "LookForYou/service/download/proto"
	"LookForYou/service/download/route"
	dlRpc "LookForYou/service/download/rpc"
)

func startRPCService() {
	service := micro.NewService(
		micro.Name("go.micro.service.download"), // 在注册中心中的服务名称
		micro.RegisterTTL(time.Second*10),
		micro.RegisterInterval(time.Second*5),
		micro.Registry(config.RegistryConsul()), // micro(v1.18) 需要显式指定consul (modifiled at 2020.04)
	)
	service.Init()

	dlProto.RegisterDownloadServiceHandler(service.Server(), new(dlRpc.Download))
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}

func startAPIService() {
	router := route.Router()
	router.Run(cfg.DownloadServiceHost)
}

func main() {
	// api 服务
	go startAPIService()

	// rpc 服务
	startRPCService()
}
