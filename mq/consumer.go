package mq

import "log"

var done chan bool

func StartConsume(qName, cName string, callback func(msg []byte) bool) {
	//1. 调用channel.Consume获得消息信道
	msgs, err := channel.Consume(
		qName,
		cName,
		true,
		false,
		false,
		false, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//2. 循环从信道里获取队列消息

	done = make(chan bool)

	go func() {
		for msg := range msgs {
			//3. 调用callback方法处理消息
			res := callback(msg.Body)
			if !res {
				// TODO: 写道另一个队列用于异常情况的重试
			}
		}
	}()

	<-done
	channel.Close()
}
