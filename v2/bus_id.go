package bus

type BusIdType = uint32

//平台：4 bit //目前不用，预留吧
//进程类型：5 bit
//区服：17 bit
//同类型进程实例编号：6
type BusId BusIdType

func BusIdFromUint32(id uint32) BusId {
	return BusId(id)
}

func GetBusId(plat, apptype, zone, inst uint32) BusId {
	return BusIdFromUint32(plat<<28 | apptype<<23 | zone<<6 | inst)
}

func (b BusId) Uint32() uint32 {
	return uint32(b)
}

func (b BusId) PlatId() BusIdType {
	return BusIdType(b >> 28)
}

func (b BusId) AppType() BusIdType {
	return BusIdType((b >> 23) & 0x0000001f)
}

func (b BusId) ZoneId() BusIdType {
	return BusIdType((b >> 6) & 0x0001ffff)
}

func (b BusId) InstId() BusIdType {
	return BusIdType(b & 0x0000003f)
}

func (b BusId) Split() (BusIdType, BusIdType, BusIdType, BusIdType) {
	return b.PlatId(), b.AppType(), b.ZoneId(), b.InstId()
}

func (b BusId) SetPlatId(v BusIdType) BusId {
	u := b.Uint32()
	return BusIdFromUint32(u&0x0fffffff | (v << 28))
}

func (b BusId) SetAppType(v BusIdType) BusId {
	u := b.Uint32()
	return BusIdFromUint32(u&0xf07fffff | (v << 23))
}

func (b BusId) SetZoneId(v BusIdType) BusId {
	u := b.Uint32()
	return BusIdFromUint32(u&0xff80003f | (v << 6))
}

func (b BusId) SetInstId(v BusIdType) BusId {
	u := b.Uint32()
	return BusIdFromUint32(u&0xffffffc0 | v)
}

func (b BusId) IsIn(ids []BusId) bool {
	for _, v := range ids {
		if b == v {
			return true
		}
	}
	return false
}

func (b BusId) IsInU32(ids []BusIdType) bool {
	id := b.Uint32()
	for _, v := range ids {
		if id == v {
			return true
		}
	}
	return false
}

func U32IsIn(id BusIdType, ids []BusId) bool {
	i := BusIdFromUint32(id)
	for _, v := range ids {
		if i == v {
			return true
		}
	}
	return false
}
