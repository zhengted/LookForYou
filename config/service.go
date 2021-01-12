package config

import (
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-plugins/registry/consul"
)

const (
	// UploadServiceHost : 上传服务监听的地址
	UploadServiceHost = "0.0.0.0:8080"
)

// RegistryConsul : 配置 consul
func RegistryConsul() registry.Registry {
	return consul.NewRegistry(
		// TODO ip需根据实际情况来修改
		registry.Addrs("49.234.178.60:8500"),
	)
}

// RegistryClient : 注册中心client
func RegistryClient(r registry.Registry) selector.Selector {
	return selector.NewSelector(
		selector.Registry(r),                      //传入consul注册
		selector.SetStrategy(selector.RoundRobin), //指定查询机制
	)
}
