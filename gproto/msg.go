package gproto

const MSG_A_INITPARAM = 0x01 //同步初始化参数
const MSG_A_DIR_INFO = 0x02  //同步目录树信息
const MSG_B_SUMLIST = 0x03
const MSG_A_PATCHLIST = 0x04
const MSG_A_END = 0x05
const MSG_B_END = 0x06

type Header struct {
	Id  int8
	Len int32
}
