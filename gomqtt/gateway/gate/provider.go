package gate

/* 多协议容器管理模块 */
type Provider interface {
	Start()
	Close() error
}

func providersStart() {
	for _, v := range Conf.Provider.Invoked {
		switch v {
		case "tcp":
			tp := &TcpProvider{}
			go tp.Start()
		case "websocket":
			wp := &WsProvider{}
			go wp.Start()
		default:
			Logger.Fatal("invalid provider,please check your configuration")
		}
	}
}
