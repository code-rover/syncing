package receiver

import (
	"crypto/md5"
	"encoding/hex"
	"syncing/gproto"
)

func MakeSumList(data []byte) *gproto.SumList {
	var sumList gproto.SumList
	if len(data) < step {
		return &sumList
	}

	sumMap := make(map[uint32]*gproto.SumInfo)

	for i := 0; i <= len(data)-step; i += step {
		sum1 := Alder32Sum(data[i : i+step])
		sum2 := md5sum(data[i : i+step])

		//fmt.Fprintf(os.Stderr, "msg sum1: %d   %s\n", sum1, data[i:i+step])
		if _, ok := sumMap[sum1]; !ok {
			sumMap[sum1] = &gproto.SumInfo{
				Sum1:     sum1,
				Sum2List: []*gproto.SumPos{{Sum: sum2, Pos: int32(i)}},
			}
		} else {
			sumMap[sum1].Sum2List = append(sumMap[sum1].Sum2List, &gproto.SumPos{Sum: sum2, Pos: int32(i)})
		}

	}

	for _, v := range sumMap {
		sumList.List = append(sumList.List, v)
	}

	return &sumList
}

func Alder32Sum(data []byte) uint32 {
	a := 1
	b := 0
	for i := 0; i < len(data); i++ {
		a += int(data[i])
		b += a
	}
	a %= 65521
	b %= 65521
	return uint32(b<<16 | a&0xffff)
}

var h = md5.New()

func md5sum(input []byte) string {
	h.Reset()
	h.Write(input)
	//return *(*string)(unsafe.Pointer(&h))
	return hex.EncodeToString(h.Sum(nil))
	//return string(h.Sum(nil))
}
