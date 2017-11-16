package service

import (
	"github.com/uber-go/zap"
)

func consumerInit() {
	// 订阅PubApns
	nc.QueueSubscribe(Conf.Nats.PubApnsTopic, Conf.Nats.PubApnsGroup, pubApns)

	// 订阅SetToken
	nc.QueueSubscribe(Conf.Nats.SetTokenTopic, Conf.Nats.SetTokenGroup, setToken)

	// 订阅DelToken
	_, err := nc.QueueSubscribe(Conf.Nats.DelTokenTopic, Conf.Nats.DelTokenGroup, delToken)
	if err != nil {
		Logger.Fatal("subscribe nats error", zap.Error(err))
	}
}
