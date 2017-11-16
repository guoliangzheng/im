package service

import (
	"github.com/labstack/echo"
	"github.com/uber-go/zap"
)

type Stream struct {
	upa      *UpdateAddr
	rpc      *Rpc
	cache    *Cache
	hash     *Hash
	nats     *natsInfo
	taskChan chan *taskMsg
}

var gStream *Stream

func New() *Stream {
	stream := &Stream{
		taskChan: make(chan *taskMsg, 100),
	}
	return stream
}

func (s *Stream) Init() {

	// init etcd
	upa := NewUpdateAddr()
	upa.Init()

	// init cache
	cache := NewCache()
	cache.Init()

	// init rpc service
	rpc := NewRpc()
	rpc.Init()

	// hash 初始化
	hash := NewHash()
	//	init other

	// @ToDo
	nats, err := newnatsInfo([]string{"nats://10.7.14.236:4222", "nats://10.7.14.26:4222"})
	if err != nil {
		Logger.Panic("nats", zap.Error(err))
	}

	s.rpc = rpc
	s.upa = upa
	s.cache = cache
	s.hash = hash
	s.nats = nats

	gStream = s
}

func (s *Stream) Start(isStatic bool) {

	loadConfig(isStatic)

	// init queue
	InitQueue(0)
	// stream 初始化所有功能服务
	s.Init()

	// rpc start
	s.rpc.Start()

	// upa start
	s.upa.Start()

	// 处理登陆发送离线消息或者订阅主题的离线消息
	startDealTask(20)

	go httpStart()
}

func (s *Stream) Close() error {
	s.upa.Close()
	s.rpc.Close()
	CloseQueue()
	return nil
}

func httpStart() {
	// 启动Http服务
	e := echo.New()

	// 启动配置热更新
	e.GET("/reload", reload)

	// e.Run(standard.New(":8907"))

	err := e.Start(":8907")
	if err != nil {
		e.Logger.Fatal(err.Error())
	}
}
