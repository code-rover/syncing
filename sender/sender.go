package sender

import (
	"bufio"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syncing/gproto"
	"time"

	"github.com/golang/protobuf/proto"
)

func Send(ip string, port string, user string, execPath string) error {

	var cmd *exec.Cmd
	var stdout io.Reader
	var stdin io.Writer

	done := make(chan bool)
	go func() {
		//cmd = exec.Command("ssh", "-p36000", "p_guangdfan@10.85.4.218", "/data/home/p_guangdfan/mydata/gosync/gsync --server")
		cmd = exec.Command("ssh", "-p"+port, user+"@"+ip, execPath+" --server")
		stdout, _ = cmd.StdoutPipe()
		stdin, _ = cmd.StdinPipe()

		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			panic(err)
		}
		done <- true
	}()

	dirInfoBytes, fidMap, err := ReadDirInfo()
	if err != nil {
		return errors.New(err.Error())
	}

	println("data len: " + strconv.Itoa(len(dirInfoBytes)))
	// bstr := fmt.Sprintf("%X", dd)
	// println("bstr: " + bstr)
	// fmt.Println(len(bstr))

	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(dirInfoBytes)))

	bufWriter := bufio.NewWriter(stdin)
	bufReader := bufio.NewReader(stdout)

	n, err := bufWriter.Write(lenBuf)
	print("Write n: ")
	println(n)
	if err != nil {
		println("Write error")
		return errors.New(err.Error())
	}

	n, err = bufWriter.Write(dirInfoBytes)
	if err != nil {
		return errors.New(err.Error())
	}

	fmt.Printf("write: %d\n", n)
	bufWriter.Flush()

	time.Sleep(1 * time.Second)

	lenBuf = make([]byte, 4)
	n, err = io.ReadFull(bufReader, lenBuf)
	if err != nil {
		println(n)
		println("error: " + err.Error())
		return errors.New(err.Error())
	}
	length := binary.LittleEndian.Uint32(lenBuf)
	dataBuf := make([]byte, length)
	n, err = io.ReadFull(bufReader, dataBuf)
	if err != nil {
		return errors.New(err.Error())
	}

	var fileIdStruct gproto.FileIdStruct
	err = proto.Unmarshal(dataBuf, &fileIdStruct)
	if err != nil {
		return errors.New(err.Error())
	}

	println("idlist size: " + strconv.Itoa(len(fileIdStruct.IdList)) + "\n")
	for _, fid := range fileIdStruct.IdList {
		println(fidMap[fid])
	}

	//just for test
	n, err = bufWriter.Write(lenBuf)
	bufWriter.Flush()

	<-done
	fmt.Println("over!")
	return nil
}

func ReadDirInfo() ([]byte, map[int32]string, error) {
	var rootDir *gproto.DirStruct
	var stack = list.New()

	type DNode struct {
		dirStruct *gproto.DirStruct
		path      string
	}

	MakeNode := func(ds *gproto.DirStruct, p string) *DNode {
		return &DNode{
			dirStruct: ds,
			path:      p,
		}
	}
	fid := int32(1)
	fidMap := make(map[int32]string)

	firstItem := &gproto.DirStruct{
		Name:  ".",
		Mtime: 0,
		Mode:  0,
	}
	stack.PushBack(MakeNode(firstItem, "./"))
	rootDir = firstItem

	var current *DNode
	for stack.Back() != nil {
		current = stack.Back().Value.(*DNode)
		stack.Remove(stack.Back())

		dir, err := ioutil.ReadDir(current.path)
		if err != nil {
			return nil, nil, errors.New("ReadDir error: " + err.Error())
		}
		for _, f := range dir {
			if f.IsDir() {
				if f.Name() == ".git" /* || f.Name() == "pkg"*/ {
					continue
				}
				newNode := MakeNode(&gproto.DirStruct{
					Name:  f.Name(),
					Mode:  0,
					Mtime: f.ModTime().Unix(),
				}, current.path+f.Name()+"/")
				stack.PushBack(newNode)
				current.dirStruct.DirList = append(current.dirStruct.DirList, newNode.dirStruct)
			} else {
				newFileStruct := gproto.FileStruct{
					Name:  f.Name(),
					Fid:   fid,
					Mtime: f.ModTime().Unix(),
					Mode:  0,
					Size:  f.Size(),
					//Hash:  hex.EncodeToString(md5hash.Sum(nil)),
				}
				current.dirStruct.FileList = append(current.dirStruct.FileList, &newFileStruct)
				fidMap[fid] = current.path + f.Name()
				fid++
			}
		}
	}

	//printTree(rootDir, 0)
	rootDir.Name = "/home/darren/syncing"
	data, err := proto.Marshal(rootDir)
	if err != nil {
		//panic("Marsha1 error")
		return nil, nil, errors.New("Marshl error: " + err.Error())
	}

	return data, fidMap, nil
}
