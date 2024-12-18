package bus

import (
	"fmt"
	"sync"
	"time"

	netapi "github.com/Lei2050/lei-net/api"
	pkt "github.com/Lei2050/lei-net/packet/v2"
	tcp "github.com/Lei2050/lei-net/tcp/v2"
	cls "github.com/Lei2050/lei-utils/cls"
	log "github.com/Lei2050/lei-utils/log"
)

var _ tcp.Protocoler = &Bus{}
var _ pkt.PacketHandler = &Bus{}

type Bus struct {
	Config *BusConfig

	tcpSers []netapi.TcpServerer
	tcpClis map[string]netapi.TcpClienter //key = server_addr

	tcpConns map[BusId]netapi.TcpConnectioner

	packetHandler pkt.PacketHandler

	c chan RawMsg

	*pkt.Broker
	cls.CloseUtil
	sync.RWMutex
}

func NewBusServer(busId []BusId, config *BusConfig, packetHandler pkt.PacketHandler) *Bus {
	config.BusId = make([]uint32, len(busId))
	for k := range config.BusId {
		config.BusId[k] = busId[k].Uint32()
	}
	b := &Bus{
		Config:        config,
		tcpClis:       make(map[string]netapi.TcpClienter),
		tcpConns:      make(map[BusId]netapi.TcpConnectioner),
		packetHandler: packetHandler,
		c:             make(chan RawMsg, 1024),
		CloseUtil:     cls.MakeCloseUtil(),
	}
	b.Broker = pkt.NewBroker(b)
	return b
}

func (b *Bus) getClientCloseCb(addr string) func() {
	s := addr
	return func() {
		//log.Debug("tcpClis:%+v", b.tcpClis)
		b.Lock()
		delete(b.tcpClis, s)
		b.Unlock()
		//log.Debug("after tcpClis:%+v", b.tcpClis)
	}
}

func (b *Bus) getConnCloseCb(busId BusId) func() {
	busid := busId
	return func() {
		//log.Debug("tcpConns:%+v", b.tcpConns)
		b.Lock()
		delete(b.tcpConns, busid)
		b.Unlock()
		//log.Debug("after tcpConns:%+v", b.tcpConns)
	}
}

func (b *Bus) reconncectAndClear() {
	var (
		reconnects []string
		dials      []ClientXml
	)
	b.RLock()
	for _, v := range b.Config.Clients {
		if v.AutoReconnect <= 0 {
			continue
		}

		tcpcli := b.tcpClis[v.ClientAddr]
		if tcpcli != nil {
			if tcpcli.IsClosed() {
				reconnects = append(reconnects, v.ClientAddr)
			}
			continue //连接存在并有效
		}

		dials = append(dials, v)
	}
	b.RUnlock()

	for _, v := range reconnects {
		tcpcli := b.tcpClis[v]
		if tcpcli == nil {
			continue
		}
		log.Infof("try reconnect %s", v)
		err := tcpcli.Reconnect()
		if err != nil { //重连失败，等下次再继续
			log.Errorf("Reconnect %s failed:%+v", v, err)
			continue
		}
		b.Lock()
		b.tcpClis[v] = tcpcli
		b.Unlock()
	}

	for _, v := range dials {
		log.Infof("Bus dial %s", v.ClientAddr)
		tcpCli, err := tcp.Dial(v.ClientAddr, b,
			tcp.IdleTime(v.ClientIdleTime),
			//tcp.InBuffSize(b.Config.InBufSize),
			//tcp.OutBuffSize(b.Config.OutBufSize),
			tcp.ReadMaxSize(b.Config.InMaxSize),
			tcp.WriteMaxSize(b.Config.OutMaxSize),
		)
		if err != nil {
			log.Errorf("dial tcpclient on %+v failed", v)
			continue
		}

		b.Lock()
		b.tcpClis[v.ClientAddr] = tcpCli
		b.Unlock()
		//if v.HeartBeat > 0 {
		//	go b.heartBeatLoop(*v, tcpCli)
		//}
	}
}

// 定时尝试重连并且清理
func (b *Bus) reconncectAndClearLoop() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			b.reconncectAndClear()
		case <-b.C():
			return
		}
	}
}

func (b *Bus) Start() error {
	log.Infof("BusConfig:%+v", b.Config)

	b.tcpSers = make([]netapi.TcpServerer, len(b.Config.ServerAddr))
	for k, v := range b.Config.ServerAddr {
		tcpSer, err := tcp.NewServer(
			b,
			tcp.Address(v),
			tcp.MaxConn(b.Config.ServerMaxConn),
			tcp.IdleTime(b.Config.ServerIdleTime),
		)
		if err != nil {
			log.Errorf("create tcpserver on %s failed", v)
			return err
		}

		log.Infof("Bus listening on %s", v)

		go tcpSer.Start()
		b.tcpSers[k] = tcpSer
	}

	for _, v := range b.Config.Clients {
		log.Infof("Bus dial %s", v.ClientAddr)
		tcpCli, err := tcp.Dial(v.ClientAddr, b,
			tcp.IdleTime(v.ClientIdleTime),
			//tcp.InBuffSize(b.Config.InBufSize),
			//tcp.OutBuffSize(b.Config.OutBufSize),
			tcp.ReadMaxSize(b.Config.InMaxSize),
			tcp.WriteMaxSize(b.Config.OutMaxSize),
			tcp.HeartBeat(v.HeartBeat),
		)
		if err != nil {
			log.Errorf("dial tcpclient on %+v failed", v)
			if v.AutoReconnect > 0 {
				continue
			}
			return err
		}

		b.tcpClis[v.ClientAddr] = tcpCli
		tcpCli.RegisterCloseCb(b.getClientCloseCb(v.ClientAddr))
		//if v.HeartBeat > 0 {
		//	go b.heartBeatLoop(*v, tcpCli)
		//}
	}

	go b.reconncectAndClearLoop()

	return nil
}

func (b *Bus) RegisterBusId(id BusId, conn netapi.TcpConnectioner) {
	log.Infof("Bus registered:%v", id)
	//log.Debug("tcpConns:%+v", b.tcpConns)
	b.Lock()
	b.tcpConns[id] = conn
	conn.RegisterCloseCb(b.getConnCloseCb(id))
	b.Unlock()
	//log.Debug("after tcpConns:%+v", b.tcpConns)
}

func (b *Bus) C() <-chan RawMsg {
	return b.c
}

func (b *Bus) GetConnByBusId(id BusId) netapi.TcpConnectioner {
	b.RLock()
	defer b.RUnlock()
	conn := b.tcpConns[id]
	return conn
}

func (b *Bus) SendMsgToBusId(id BusId, msg any) error {
	conn := b.GetConnByBusId(id)
	if conn == nil {
		return fmt.Errorf("not find busid:%+v", id)
	}

	conn.Write(msg)

	return nil
}

func (b *Bus) Exit() {
	b.Lock()
	defer b.Unlock()
	for _, v := range b.tcpConns {
		v.Close()
	}

	b.Close(nil)
}

func (b *Bus) GetBusIdByAppType(apptype BusIdType) BusId {
	b.RLock()
	defer b.RUnlock()
	for k := range b.tcpConns {
		if k.AppType() == apptype {
			return k
		}
	}
	return 0
}

func (b *Bus) GetBusIdByAppTypeAndZoneId(apptype BusIdType, zoneId BusIdType) BusId {
	b.RLock()
	defer b.RUnlock()
	for k := range b.tcpConns {
		if k.AppType() == apptype && k.ZoneId() == zoneId {
			return k
		}
	}
	return 0
}

func (b *Bus) GetBusIdsByAppType(apptype BusIdType) []BusId {
	busIds := make([]BusId, 0)
	b.RLock()
	defer b.RUnlock()
	for k := range b.tcpConns {
		if k.AppType() == apptype {
			busIds = append(busIds, k)
		}
	}
	return busIds
}

type RawMsg struct {
	packetHandler pkt.PacketHandler
	conn          netapi.TcpConnectioner
	packet        *pkt.Packet
}

func (r *RawMsg) Process() {
	r.packetHandler.Process(r.conn, r.packet)
	r.packet.Release()
}
