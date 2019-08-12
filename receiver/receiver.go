package receiver

import (
	"bufio"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"syncing/gproto"

	"github.com/golang/protobuf/proto"
)

func RunServer() {
	errwriter := bufio.NewWriter(os.Stderr)
	errwriter.WriteString("msg welcome!\n")
	errwriter.Flush()

	//basePath := "/home/darren/syncing/dir1/"

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(reader, lenBuf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ReadFull error: %s\n", err)
		return
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	errwriter.WriteString("msg recv len: " + strconv.Itoa(int(length)) + "\n")

	dataBuf := make([]byte, length)
	n, err := io.ReadFull(reader, dataBuf)
	if err != nil {
		errwriter.WriteString("msg err " + err.Error())
		return
	}
	errwriter.WriteString("msg recv data len: " + strconv.Itoa(int(n)) + "\n")

	// bstr := fmt.Sprintf("%X", dataBuf)
	// errwriter.WriteString("msg recv data bin: " + bstr + "\n")
	// errwriter.Flush()

	fileIdStruct, err := FileListCheck(dataBuf)
	errwriter.WriteString("msg fileIdStruct: " + strconv.Itoa(len(fileIdStruct.IdList)) + "\n")

	if err != nil {
		errwriter.WriteString("msg err FileListCheck: " + err.Error())
		return
	}
	fidBytes, err := proto.Marshal(fileIdStruct)
	if err != nil {
		errwriter.WriteString("msg err Marshal: " + err.Error())
		return
	}
	lenBuf = make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(fidBytes)))
	n, err = writer.Write(lenBuf)
	if err != nil {
		errwriter.WriteString("msg err Writer lenbuf: " + err.Error())
		return
	}

	n, err = writer.Write(fidBytes)
	if err != nil {
		errwriter.WriteString("msg err Writer bidBytes: " + err.Error())
		return
	}
	errwriter.WriteString("msg bidBytes len: " + strconv.Itoa(n) + "\n")
	errwriter.Flush()
	writer.Flush()

	//just waiting for sender msg
	lenBuf = make([]byte, 4)
	io.ReadFull(reader, lenBuf)
}

func FileListCheck(dataBuf []byte) (*gproto.FileIdStruct, error) {
	errwriter := bufio.NewWriter(os.Stderr)

	var ds gproto.DirStruct
	err := proto.Unmarshal(dataBuf, &ds)
	if err != nil {
		errwriter.WriteString("msg err " + err.Error())
		errwriter.Flush()
		return nil, errors.New(err.Error())
	}
	errwriter.WriteString("msg Unmarshal data success! getname: " + ds.GetName() + "\n")

	var fileIdStruct gproto.FileIdStruct

	pathStack := list.New()                        //用于计算全路径
	visitedMap := make(map[*gproto.DirStruct]bool) //记录节点是否访问过

	stack := list.New()
	stack.PushBack(&ds)

	for stack.Back() != nil {
		ds := stack.Back().Value.(*gproto.DirStruct)
		stack.Remove(stack.Back())

		if _, ok := visitedMap[ds]; !ok {
			visitedMap[ds] = true
		}
		pathStack.PushBack(ds)

		fileList := ds.GetFileList()
		var fullPath string
		if len(fileList) > 0 {
			for e := pathStack.Front(); e != nil; e = e.Next() {
				fullPath += ((e.Value.(*gproto.DirStruct).Name) + "/")
			}
		}

		for _, file := range fileList {
			filePath := /*basePath + */ fullPath + file.Name
			// errwriter.WriteString("msg file name: " + filePath + "\n")
			// errwriter.Flush()

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					errwriter.WriteString("msg missing " + filePath + "\n")
					fileIdStruct.IdList = append(fileIdStruct.IdList, file.GetFid())
					continue
				}
				errwriter.WriteString("msg err " + err.Error() + "\n")
				continue
			}

			if file.Mtime != fileInfo.ModTime().Unix() || file.Size != fileInfo.Size() {
				errwriter.WriteString("msg diff " + filePath + "\n")
				fileIdStruct.IdList = append(fileIdStruct.IdList, file.GetFid())
			}
		}

		//update pathStack
		if len(ds.GetDirList()) == 0 {
			for e := pathStack.Back(); e != nil; {
				item := e.Value.(*gproto.DirStruct)
				// fmt.Fprintf(os.Stderr, "%s   ", fullPath)

				childVisited := true
				for _, child := range item.DirList {
					if !visitedMap[child] {
						childVisited = false
						break
					}
				}

				if childVisited {
					preEle := e.Prev()
					// fmt.Fprintf(os.Stderr, "  remove: %s\n", item.Name)
					pathStack.Remove(e)
					e = preEle
				} else {
					break
				}

				//更新fullPath  just for log
				// fullPath = ""
				// for e := pathStack.Front(); e != nil; e = e.Next() {
				// 	fullPath += ((e.Value.(*gproto.DirStruct).Name) + "/")
				// }
			}
		}

		for _, dir := range ds.GetDirList() {
			stack.PushBack(dir)
		}
	}
	errwriter.Flush()
	return &fileIdStruct, nil
}
