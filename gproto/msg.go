package gproto

const MSG_A_DIR_INFO = 0x01
const MSG_B_SUMLIST = 0x02
const MSG_A_PATCHLIST = 0x03
const MSG_A_END = 0x04
const MSG_B_END = 0x05

type Header struct {
	Id  int8
	Len int32
}
