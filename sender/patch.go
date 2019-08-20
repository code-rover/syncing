package sender

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"syncing/gproto"
)

func MakePatch(f2 []byte, sumList *gproto.SumList) *gproto.PatchList {
	blockMap := make(map[uint32][]*gproto.SumPos, len(sumList.List))
	for i := 0; i < len(sumList.List); i++ {
		blockMap[sumList.List[i].Sum1] = sumList.List[i].Sum2List
	}
	fmt.Printf("sum1 size: %d\n", len(sumList.List))

	dataLen := len(f2)

	var patchList gproto.PatchList
	var backItem *gproto.Patch

	var bufA = -1 //差异开始段位置
	var bufB = -1 //差异结束段位置
	i := 0

	var sum1 uint32

	for i = 0; i+step <= dataLen; {
		backItem = nil
		if len(patchList.List) > 0 {
			backItem = patchList.List[len(patchList.List)-1]
		}

		// if bufA != -1 && i > step && sum1 > 0 {
		// 	sum1 = Alder32SumBasedOnPrev(f2, i, sum1) //根据上一结果增量计算,bufA不等于-1意味着上一步是连续差异数据，可以借用上次结果增量计算本次alder32值
		// } else {
		sum1 = Alder32Sum(f2[i : i+step])
		// }

		sum2List, isSum1Exist := blockMap[sum1]
		sumPos := int32(-1)
		if isSum1Exist { //需要继续检查sum2
			//fmt.Printf("sum1 exist %d\n", sum1)
			for _, sum2Pos := range sum2List {
				if sum2Pos.Sum == md5sum(f2[i:i+step]) {
					sumPos = sum2Pos.Pos
					// fmt.Printf("sumPos: %d  str: %s\n", sumPos, f2[i:i+step])
					break
				}
			}
		}

		if isSum1Exist && sumPos > -1 {
			if bufA != -1 {
				buf := bytes.NewBuffer(backItem.Data)
				buf.Write(f2[bufA:bufB])
				backItem.Data = buf.Bytes()
				bufA = -1
				bufB = -1
			}
			//fmt.Printf("find: %d   %d   %s\n", sum1, sumPos, f2[i:i+step])

			//优化 队列上一个元素不是字符串 或  间断块
			if backItem == nil || backItem.Pos == -1 || sumPos != backItem.Pos+backItem.Len {
				backItem = &gproto.Patch{
					Pos: sumPos,
					Len: int32(step),
				}
				patchList.List = append(patchList.List, backItem)
			} else {
				backItem.Len += int32(step)
			}

			i += step

		} else { //差异部分
			if backItem == nil || backItem.Pos > -1 {
				backItem = &gproto.Patch{
					Pos: -1,
				}
				patchList.List = append(patchList.List, backItem)
			}

			if bufA == -1 {
				bufA = i
				bufB = i
			}
			bufB++
			i++
		}
		//println(i)
	}

	//剩余差异内容
	if bufA > -1 {
		buf := bytes.NewBuffer(backItem.Data)
		buf.Write(f2[bufA:bufB])
		backItem.Data = buf.Bytes()
	}

	//剩余block处理
	if i+step > dataLen { //不足一个block的剩余
		if backItem == nil || backItem.Pos > -1 {
			backItem = &gproto.Patch{Pos: -1}
			patchList.List = append(patchList.List, backItem)
		}
		buf := bytes.NewBuffer(backItem.Data)
		buf.Write(f2[i:len(f2)])
		backItem.Data = buf.Bytes()
	}

	return &patchList
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

//根据之前结果增量计算
func Alder32SumBasedOnPrev(data []byte, curPos int, prev uint32) uint32 {
	d1 := uint32(data[curPos-step])
	d2 := uint32(data[curPos])
	prevA := prev & 0xffff
	prevB := (prev >> 16) & 0xffff
	prevA -= d1
	prevA += d2
	prevB -= uint32(step) * d1
	prevB--
	prevB += prevA
	prevA %= 65521
	prevB %= 65521
	return prevB<<16 | prevA&0xffff
}

var h = md5.New()

func md5sum(input []byte) string {
	h.Reset()
	h.Write(input)
	//return *(*string)(unsafe.Pointer(&h))
	return hex.EncodeToString(h.Sum(nil))
	//return string(h.Sum(nil))
}
