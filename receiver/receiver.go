package receiver

import (
	"bufio"
	"container/list"
	"encoding/binary"
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

	//writer := bufio.NewWriter(os.Stdout)
	//basePath := "/home/darren/syncing/dir1/"

	reader := bufio.NewReader(os.Stdin)
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(reader, lenBuf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ReadFull error: %s\n", err)
		os.Exit(-1)
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	errwriter.WriteString("msg recv len: " + strconv.Itoa(int(length)) + "\n")
	errwriter.Flush()

	dataBuf := make([]byte, length)
	n, err := io.ReadFull(reader, dataBuf)
	if err != nil {
		errwriter.WriteString("msg err " + err.Error())
		os.Exit(-2)
	}
	errwriter.WriteString("msg recv data len: " + strconv.Itoa(int(n)) + "\n")
	errwriter.Flush()

	// bstr := fmt.Sprintf("%X", dataBuf)
	// errwriter.WriteString("msg recv data bin: " + bstr + "\n")
	// errwriter.Flush()

	var ds gproto.DirStruct
	err = proto.Unmarshal(dataBuf, &ds)
	if err != nil {
		errwriter.WriteString("msg err " + err.Error())
		os.Exit(-3)
	}
	errwriter.WriteString("msg Unmarshal data success! getname: " + ds.GetName() + "\n")
	errwriter.Flush()

	pathStack := list.New()
	cntMap := make(map[*gproto.DirStruct]int)

	stack := list.New()
	stack.PushBack(&ds)

	for stack.Back() != nil {
		ds := stack.Back().Value.(*gproto.DirStruct)
		stack.Remove(stack.Back())

		if _, ok := cntMap[ds]; !ok {
			cntMap[ds] = 0
		}
		//if len(ds.DirList) > 0 {
		cntMap[ds]++
		//}
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
			errwriter.WriteString("msg file name: " + filePath + "\n")
			errwriter.Flush()

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					errwriter.WriteString("msg missing " + filePath + "\n")
					errwriter.Flush()
					continue
				}
				errwriter.WriteString("msg err " + err.Error() + "\n")
				errwriter.Flush()
				continue
			}

			if file.Mtime != fileInfo.ModTime().Unix() || file.Size != fileInfo.Size() {
				errwriter.WriteString("msg diff " + filePath + "\n")
				errwriter.Flush()
			}
		}

		//update pathStack
		if len(ds.GetDirList()) == 0 {
			for e := pathStack.Back(); e != nil; {
				item := e.Value.(*gproto.DirStruct)
				fmt.Fprintf(os.Stderr, "%s   ", fullPath)

				childVisited := true
				for _, child := range item.DirList {
					if cntMap[child] == 0 {
						childVisited = false
						fmt.Fprintf(os.Stderr, "\n")
						break
					}
				}

				if childVisited {
					preEle := e.Prev()
					fmt.Fprintf(os.Stderr, "  remove: %s\n", item.Name)
					pathStack.Remove(e)
					e = preEle
				} else {
					break
				}

				//更新fullPath  just for log
				fullPath = ""
				for e := pathStack.Front(); e != nil; e = e.Next() {
					fullPath += ((e.Value.(*gproto.DirStruct).Name) + "/")
				}
			}
		}

		//errwriter.WriteString("msg dir size: " + strconv.Itoa(len(ds.GetDirList())) + "\n")
		for _, dir := range ds.GetDirList() {
			//errwriter.WriteString("msg dir name: " + dir.Path + dir.Name + "\n")
			errwriter.Flush()
			stack.PushBack(dir)
		}

	}
	errwriter.Flush()
}
