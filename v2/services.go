package bus

import (
	api "github.com/Lei2050/lei-net/api"
	pkt "github.com/Lei2050/lei-net/packet/v2"
	log "github.com/Lei2050/lei-utils/log"
)

// //////////////// impliments pkt.PacketHandler - start //////////////////
func (b *Bus) Process(conn api.TcpConnectioner, packet *pkt.Packet) {
	if packet.GetPayloadLen() <= 0 {
		return
	}

	msgId := packet.ReadUint32()
	switch msgId {
	case MsgIdHeartbeat:
		conn.Write(getHeartbeatAckPacket())
		return
	case MsgIdHeartbeatAck:
		return
	case MsgIdRegisterBusId:
		b.onRegisterBusId(conn, packet)
		return
	case MsgIdRegisterBusIdAck:
		b.onRegisterBusIdAck(conn, packet)
		return
	}

	packet.SetReadPos(0)

	//传递给上层
	b.c <- RawMsg{packetHandler: b.packetHandler, conn: conn, packet: packet}
}

// //////////////// impliments pkt.PacketHandler - end //////////////////

// //////////////// impliments tcp.Protocoler - start //////////////////
func (b *Bus) OnConnect(conn api.TcpConnectioner) {
	b.Broker.OnConnect(conn)

	if len(b.Config.BusId) <= 0 {
		return
	}
	packet := getRegisterBusIdPacket(b.Config.BusId)
	conn.Write(packet)
}

func (b *Bus) HeartBeatMsg() any {
	return getHeartbeatPacket()
}

// //////////////// impliments tcp.Protocoler - end //////////////////

const (
	MsgIdHeartbeat        uint32 = 0xA1DB4BF6
	MsgIdHeartbeatAck     uint32 = 0xA1DB4BF7
	MsgIdRegisterBusId    uint32 = 0xA1DB4BF8
	MsgIdRegisterBusIdAck uint32 = MsgIdRegisterBusId + 1
)

func getHeartbeatPacket() *pkt.Packet {
	packet := pkt.NewPacket()
	packet.WriteUint32(MsgIdHeartbeat)
	return packet
}

func getHeartbeatAckPacket() *pkt.Packet {
	packet := pkt.NewPacket()
	packet.WriteUint32(MsgIdHeartbeatAck)
	return packet
}

func getRegisterBusIdPacket(busIds []uint32) *pkt.Packet {
	packet := pkt.NewPacket()
	packet.WriteUint32(MsgIdRegisterBusId)
	packet.WriteInt16(int16(len(busIds)))
	for _, v := range busIds {
		packet.WriteUint32(v)
	}
	return packet
}

func getRegisterBusIdAckPacket(busIds []uint32) *pkt.Packet {
	packet := pkt.NewPacket()
	packet.WriteUint32(MsgIdRegisterBusIdAck)
	packet.WriteInt16(int16(len(busIds)))
	for _, v := range busIds {
		packet.WriteUint32(v)
	}
	return packet
}

// //////////////// services - start //////////////////

func (b *Bus) onRegisterBusId(conn api.TcpConnectioner, packet *pkt.Packet) {
	peerBusIdCnt := packet.ReadInt16()
	log.Tracef("onRegisterBusId %d, %d", conn.Id(), peerBusIdCnt)
	for i := 0; i < int(peerBusIdCnt); i++ {
		peerBusId := packet.ReadUint32()
		b.RegisterBusId(BusId(peerBusId), conn)
		log.Tracef("onRegisterBusId %d, %d", conn.Id(), peerBusId)
	}

	if len(b.Config.BusId) <= 0 {
		return
	}

	conn.Write(getRegisterBusIdAckPacket(b.Config.BusId))
}

func (b *Bus) onRegisterBusIdAck(conn api.TcpConnectioner, packet *pkt.Packet) {
	peerBusIdCnt := packet.ReadInt16()
	log.Tracef("onRegisterBusIdAck %d, %d", conn.Id(), peerBusIdCnt)
	for i := 0; i < int(peerBusIdCnt); i++ {
		peerBusId := packet.ReadUint32()
		log.Tracef("onRegisterBusIdAck %d, %d", conn.Id(), peerBusId)
	}
}

// //////////////// services - end //////////////////
