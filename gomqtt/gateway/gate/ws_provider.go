package gate

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/uber-go/zap"
)

/* Websocket协议容器管理模块*/
type WsProvider struct {
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (tp *WsProvider) Start() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		//websocket.Conn
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			Logger.Info("update websocket error", zap.Error(err))
			return
		}

		ci := &connInfo{}
		ci.tp = 2
		ci.wsC = conn

		serve(ci)
	})

	Logger.Debug("websocket provider startted", zap.String("addr", Conf.Provider.WsAddr))

	err := http.ListenAndServe(Conf.Provider.WsAddr, nil)
	if err != nil {
		Logger.Fatal("Websocket ListenAndServe error ", zap.Error(err))
	}
}

func (tp *WsProvider) Close() error {
	return nil
}
