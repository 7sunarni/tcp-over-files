package multiplexer

import (
	"encoding/binary"
)

const (
	IDIndex     = 4
	StatusIndex = IDIndex + 4
	SizeIndex   = StatusIndex + 4
	DataLength  = 500
)

type Frame struct {
	ID     uint32 // 4
	Status uint32 // 4
	Size   uint32 // 4
	Data   []byte // 500
}

func NewFrame(d []byte) *Frame {
	// TODO: check size. use read?
	f := &Frame{}
	f.ID = binary.LittleEndian.Uint32(d[0:IDIndex])
	f.Status = binary.LittleEndian.Uint32(d[IDIndex:StatusIndex])
	f.Size = binary.LittleEndian.Uint32(d[StatusIndex:SizeIndex])
	f.Data = d[SizeIndex : SizeIndex+f.Size]
	return f
}

func (f *Frame) bytes() []byte {
	ret := make([]byte, SizeIndex+DataLength)
	if len(f.Data) > DataLength {
		if f.Size == 0 {
			f.Size = DataLength
		}
		copy(ret[SizeIndex:], f.Data[0:DataLength])
	} else {
		if f.Size == 0 {
			f.Size = uint32(len(f.Data))
		}
		copy(ret[SizeIndex:], f.Data)
	}
	binary.LittleEndian.PutUint32(ret[0:IDIndex], f.ID)
	binary.LittleEndian.PutUint32(ret[IDIndex:StatusIndex], f.Status)
	binary.LittleEndian.PutUint32(ret[StatusIndex:SizeIndex], f.Size)
	return ret
}
