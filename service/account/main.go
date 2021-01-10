package main

import (
	"LookForYou/service/account/handler"
	"LookForYou/service/account/proto"
	"github.com/micro/go-micro"
	"log"
	"time"
)

func main() {
	// 创建一个服务
	service := micro.NewService(
		micro.Name("go.micro.service.user"),
		micro.RegisterTTL(time.Second*10),
		micro.RegisterInterval(time.Second*5),
	)
	service.Init()

	proto.RegisterUserServiceHandler(service.Server(), new(handler.User))

	if err := service.Run(); err != nil {
		log.Println(err.Error())
	}

}
