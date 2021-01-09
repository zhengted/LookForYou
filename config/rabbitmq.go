package config

const (
	// AsyncTransferEnable: 是否开启文件异步转移
	AsyncTransferEnable = true
	// RabbitURL: RabbitMQ服务的入口URL
	RabbitURL = "amqp://guest:guest@127.0.0.1:5672/"
	// TransExchangeName: 用于文件transfer的交换机
	TransExchangeName = "uploadserver.trans"
	// TransOSSQueueName: OSS队列转移名
	TransOSSQueueName = "uploadserver.trans.oss"
	// TransOSSErrQueueName: OSS转移失败后写入另一个队列的队列名
	TransOSSErrQueueName = "uploadserver.trans.oss.err"
	// TransOssRoutingKey: routingkey
	TransOssRoutingKey = "oss"
)
