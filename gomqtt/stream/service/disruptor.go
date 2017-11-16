package service

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"stathat.com/c/consistent"

	disruptor "github.com/smartystreets/go-disruptor"
	"im.zgl/gomqtt/proto"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

const (
	CACHE_TEXT_INSERT = 1000 // 插入
	CACHE_JSON_INSERT = 1001 // 插入
	CACHE_TEXT_GET    = 2000 // 获取
	CACHE_JSON_GET    = 2001 // 获取
	CACHE_TEXT_SELECT = 3000 // 查询
	CACHE_JSON_SELECT = 3001 // 查询
	CACHE_TEXT_DELETE = 4000 // 删除
	CACHE_JSON_DELETE = 4001 // 删除
)

var gQueue map[string]*Controller
var gQueueHash *consistent.Consistent

func InitQueue(queueNum int) {
	gQueue = make(map[string]*Controller)
	gRetChan = make(chan *CacheRet, 100)
	gQueueHash = consistent.New()
	if queueNum == 0 {
		queueNum = runtime.NumCPU()
	}

	Logger.Info("InitQueue", zap.Int("queueNum", queueNum))

	for index := 0; index < queueNum; index++ {
		controller := NewController()
		controller.Init(65536, 65535, 1)
		controller.Start()
		gQueue[fmt.Sprintf("%d", index)] = controller
		gQueueHash.Add(fmt.Sprintf("%d", index))
	}
}

func GetQueue(acc []byte) (*Controller, error) {
	index, err := gQueueHash.Get(talents.Bytes2String(acc))
	if err != nil {
		return nil, err
	}
	Logger.Info("Hash", zap.String("Acc", talents.Bytes2String(acc)), zap.String("index", index))
	c, ok := gQueue[index]
	if !ok {
		return nil, fmt.Errorf("unfind index %s in map ", index)
	}

	return c, nil
}

func CloseQueue() {
	for index, controller := range gQueue {
		controller.Start()
		delete(gQueue, index)
	}
}

type CacheTask struct {
	MsgTy   int
	RetChan chan *CacheRet
	// Acc     []byte
	// Topic   []byte
	// @TODO
	// 这里要区分来源Acc和目标Acc, msgid里面存放的为这条消息的具体信息, 完美
	FAcc    []byte            //来源Acc
	FTopic  []byte            //来源Topic
	TAcc    []byte            //目标Acc
	TTopic  []byte            //目标topic
	Msg     *proto.PubTextMsg // []byte
	JsonMsg *proto.PubJsonMsg // json推送
	MsgIDs  [][]byte
}

type CacheRet struct {
	MsgTy  int
	Acc    []byte
	Topic  []byte
	MsgIDs *TopicIDMap
	Data   []byte
}

type BufPool struct {
	Data CacheTask
}

type Controller struct {
	controller   disruptor.Disruptor
	ring         []CacheTask
	bufferMask   int64
	reservations int64
	msgIDManger  *MsgIdManger // 消息msgid缓存
	msgCache     *MsgCache    // 消息缓存
}

func NewController() *Controller {
	c := &Controller{
		msgIDManger: NewMsgIdManger(),
		msgCache:    NewMsgCache(),
	}
	return c
}

func (c *Controller) Init(bufferSize int64, bufferMask int64, reservations int64) {
	c.controller = disruptor.Configure(bufferSize).
		WithConsumerGroup(Writer{queue: c, bufPool: &sync.Pool{}}).Build()
	c.ring = make([]CacheTask, bufferSize)
	c.bufferMask = bufferMask
	c.reservations = reservations
}

func (c *Controller) Start() {
	c.controller.Start()
}

func (c *Controller) Close() error {
	c.controller.Stop()
	return nil
}

func (c *Controller) Publish(data CacheTask) {
	sequence := disruptor.InitialSequenceValue
	writer := c.controller.Writer()

	sequence = writer.Reserve(c.reservations)
	for lower := sequence - c.reservations + 1; lower <= sequence; lower++ {
		c.ring[lower&c.bufferMask] = data
	}
	// 提交
	writer.Commit(sequence-c.reservations+1, sequence)
}

// ///////////

type Writer struct {
	queue   *Controller
	bufPool *sync.Pool
}

func (this Writer) Consume(lower, upper int64) {
	for lower <= upper {
		var bufPool *BufPool
		buf := this.bufPool.Get()
		if buf == nil {
			bufPool = &BufPool{}
		} else {
			bufPool = buf.(*BufPool)
		}
		bufPool.Data = this.queue.ring[lower&this.queue.bufferMask]
		// 消费
		switch bufPool.Data.MsgTy {
		case CACHE_TEXT_INSERT:
			// 插入数据
			this.queue.msgCache.TextInsert(bufPool.Data.Msg.Mid, bufPool.Data.Msg.Msg)
			this.queue.msgIDManger.InsertMsgID(bufPool.Data.FAcc, bufPool.Data.FTopic, bufPool.Data.Msg)
			Logger.Info("CACHE_TEXT_INSERT", zap.String("ToAcc", talents.Bytes2String(bufPool.Data.Msg.ToAcc)), zap.String("Topic", talents.Bytes2String(bufPool.Data.Msg.Ttp)), zap.String("Msg", talents.Bytes2String(bufPool.Data.Msg.Msg)))
			break
		case CACHE_JSON_INSERT:
			this.queue.msgCache.JsonInsert(bufPool.Data.JsonMsg.Mid, bufPool.Data.JsonMsg.Msg)
			this.queue.msgIDManger.InsertJsonMsgID(bufPool.Data.FAcc, bufPool.Data.FTopic, bufPool.Data.JsonMsg)
			Logger.Info("CACHE_JSON_INSERT", zap.String("ToAcc", talents.Bytes2String(bufPool.Data.JsonMsg.ToAcc)),
				zap.String("Topic", talents.Bytes2String(bufPool.Data.JsonMsg.Ttp)), zap.String("Msg", talents.Bytes2String(bufPool.Data.JsonMsg.Msg)))
			break
		case CACHE_TEXT_GET:
			Logger.Info("CACHE_TEXT_GET", zap.String("Msgid", talents.Bytes2String(bufPool.Data.MsgIDs[0])))
			ret := &CacheRet{}
			data, ok := this.queue.msgCache.TextGet(bufPool.Data.MsgIDs[0])
			if ok {
				ret.Data = data
			}
			bufPool.Data.RetChan <- ret
			break
		case CACHE_TEXT_SELECT:
			Logger.Info("Text_Select", zap.String("TAcc", talents.Bytes2String(bufPool.Data.TAcc)), zap.String("TTopic", talents.Bytes2String(bufPool.Data.TTopic)))
			// 查询返回msgid信息
			tm := this.queue.msgIDManger.GetMsgIDs(bufPool.Data.TAcc, bufPool.Data.TTopic)
			ret := &CacheRet{
				MsgIDs: tm,
			}
			bufPool.Data.RetChan <- ret
			break
		case CACHE_JSON_GET:
			ret := &CacheRet{}
			Logger.Info("CACHE_JSON_GET", zap.String("Msgid", talents.Bytes2String(bufPool.Data.MsgIDs[0])))
			data, ok := this.queue.msgCache.JsonGet(bufPool.Data.MsgIDs[0])
			if ok {
				ret.Data = data
			}
			bufPool.Data.RetChan <- ret
			break
		case CACHE_JSON_SELECT:
			Logger.Info("Json_Select", zap.String("TAcc", talents.Bytes2String(bufPool.Data.TAcc)), zap.String("TTopic", talents.Bytes2String(bufPool.Data.TTopic)))
			// 查询返回msgid信息
			tm := this.queue.msgIDManger.GetMsgIDs(bufPool.Data.TAcc, bufPool.Data.TTopic)
			ret := &CacheRet{
				MsgIDs: tm,
			}
			bufPool.Data.RetChan <- ret
			break
		case CACHE_TEXT_DELETE:
			this.queue.msgCache.TextDelete(bufPool.Data.MsgIDs)
			for _, msgid := range bufPool.Data.MsgIDs {
				this.queue.msgIDManger.MsgAck(bufPool.Data.TAcc, bufPool.Data.TTopic, msgid)
			}
			break
		case CACHE_JSON_DELETE:
			this.queue.msgCache.JsonDelete(bufPool.Data.MsgIDs)
			for _, msgid := range bufPool.Data.MsgIDs {
				this.queue.msgIDManger.MsgAck(bufPool.Data.TAcc, bufPool.Data.TTopic, msgid)
			}
			break
		}
		this.bufPool.Put(buf)
		lower++
	}
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
