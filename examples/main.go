package main

import (
	"flag"
	"fmt"

	lbus "github.com/Lei2050/lei-bus/v2"
	api "github.com/Lei2050/lei-net/api"
	pkt "github.com/Lei2050/lei-net/packet/v2"
)

var busId lbus.BusId
var pBusId *int64
var ListenAddr *string //是否作为服务器
var ServerAddr *string

func init() {
	pBusId = flag.Int64("bus_id", 0, "bus id")
	ListenAddr = flag.String("listen_addr", "", "listen_addr")
	ServerAddr = flag.String("server_addr", "", "server addr")
}

func main() {
	flag.Parse()

	busId = lbus.BusId(*pBusId)
	busConfig := &lbus.BusConfig{}
	if len(*ListenAddr) > 0 {
		busConfig.ServerAddr = []string{*ListenAddr}
		busConfig.ServerMaxConn = 100
		busConfig.ServerIdleTime = 120000
	}
	if len(*ServerAddr) > 0 {
		busConfig.Clients = []lbus.ClientXml{{ClientAddr: *ServerAddr, ClientIdleTime: 180000, HeartBeat: 1000}}
	}

	bus := lbus.NewBusServer([]lbus.BusId{busId}, busConfig, &PacketHandler{})
	bus.Start()

	go func() {
		for {
			select {
			case rawMsg := <-bus.C():
				rawMsg.Process()
			case <-bus.CloseUtil.C():
				fmt.Println("bus is closed, going to exit...")
				return
			}
		}
	}()

	var input string
	var toBusId uint32
	for {
		input = ""
		fmt.Println("Please input:")
		//将信息input发送给toBusId
		fmt.Scanf("%d:%s\n", &toBusId, &input)
		if input == "exit" {
			fmt.Println("done !")
			bus.Close(nil)
			break
		}

		if len(input) <= 0 {
			continue
		}

		packet := pkt.NewPacket()
		packet.WriteInt32(1)
		packet.WriteUint32(uint32(busId))
		packet.WriteVarStrH(input)
		bus.SendMsgToBusId(lbus.BusId(toBusId), packet)
	}

	fmt.Println("Press any key to continue...")
	fmt.Scanf("%c", &input)
}

type PacketHandler struct{}

func (ph *PacketHandler) Process(conn api.TcpConnectioner, packet *pkt.Packet) {
	msgId := packet.ReadInt32()
	peerBusId := packet.ReadUint32()
	content := packet.ReadVarStrH()
	switch msgId {
	case 1:
		fmt.Printf("\"%s\" from %d\n", content, peerBusId)
		//ack
		ackPacket := pkt.NewPacket()
		ackPacket.WriteInt32(2)
		ackPacket.WriteUint32(uint32(busId))
		ackPacket.WriteVarStrH("got")
		conn.Write(ackPacket)
	case 2:
		fmt.Printf("ack \"%s\" from %d\n", content, peerBusId)
	}
}
