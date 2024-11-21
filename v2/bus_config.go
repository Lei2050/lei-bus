package bus

type ClientXml struct {
	ClientAddr     string
	ClientIdleTime int //ms
	HeartBeat      int //ms //心跳由客户端连接维持，一般设置为服务器等待数据事件的三分之一
	AutoReconnect  int //是否自动断线重连
}

type BusConfig struct {
	BusId []uint32 //数组，支持一个bus拥有多个busId的情况。上层可能会修改，或上层可能会不用

	ServerAddr     []string
	ServerMaxConn  int
	ServerIdleTime int //ms

	Clients []ClientXml

	InBufSize  int //读缓冲初始大小
	OutBufSize int //写缓冲初始大小
	InMaxSize  int //读数据长度硬上限
	OutMaxSize int //写数据长度硬上限
}
