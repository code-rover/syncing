package sender

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syncing/comm"
	"syncing/gproto"

	"github.com/golang/protobuf/proto"
)

var step = 10

var conn *comm.Connection
var fidMap map[int32]string
var remoteBasePath string

func Send(ip string, port string, user string, execPath string, rpath string) error {

	var cmd *exec.Cmd
	var stdout io.Reader
	var stdin io.Writer

	done := make(chan bool)
	go func() {
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

	remoteBasePath = rpath
	var dirInfoBytes []byte
	var err error
	dirInfoBytes, fidMap, err = ReadDirInfo()
	if err != nil {
		return errors.New(err.Error())
	}

	println("data len: " + strconv.Itoa(len(dirInfoBytes)))
	// bstr := fmt.Sprintf("%X", dd)
	// println("bstr: " + bstr)
	// fmt.Println(len(bstr))

	bufWriter := bufio.NewWriter(stdin)
	bufReader := bufio.NewReader(stdout)
	conn = comm.NewConn(bufReader, bufWriter)

	n, err := conn.Send(gproto.MSG_A_DIR_INFO, dirInfoBytes)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("write: %d\n", n)
	bufWriter.Flush()

	ProcessMsg(conn)

	<-done
	fmt.Println("over!")
	return nil
}

func ProcessMsg(conn *comm.Connection) error {
	for {
		cmd, st, err := conn.Recv()
		if err != nil {
			fmt.Printf("Recv error: %s\n", err.Error())
			return err
		}

		if cmd == gproto.MSG_B_SUMLIST {
			fileSumList := st.(*gproto.FileSumList)

			for _, sumList := range fileSumList.List {
				// fmt.Printf(">%d  %d: %s\n", i, sumList.Fid, fidMap[sumList.Fid])
				fdata, err := ioutil.ReadFile(fidMap[sumList.Fid])
				if err != nil {
					fmt.Printf("ReadFile error: %s\n", err.Error())
					return err
				}
				// fmt.Printf("Readfile %s  size: %d\n", fidMap[sumList.Fid], len(fdata))
				patchList := MakePatch(fdata, sumList)
				patchList.Fid = sumList.Fid
				// fmt.Printf(">patch size: %d\n", len(patchList.List))

				patchListBytes, err := proto.Marshal(patchList)
				if err != nil {
					fmt.Printf("Marshal error: %s\n", err.Error())
					return err
				}
				//fmt.Printf(">patch after Marsha1size:%s   %d\n", fidMap[sumList.Fid], len(patchListBytes))

				_, err = conn.Send(gproto.MSG_A_PATCHLIST, patchListBytes)
				if err != nil {
					return err
				}
			}
			conn.Send(gproto.MSG_A_END, []byte{})

		} else if cmd == gproto.MSG_B_END {
			// fmt.Println("recv MSG_B_END")
			return nil
		}
	}
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
	fid := int32(1) //分配标识id
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
	rootDir.Name = remoteBasePath //"/home/darren/syncing"
	data, err := proto.Marshal(rootDir)
	if err != nil {
		//panic("Marsha1 error")
		return nil, nil, errors.New("Marshl error: " + err.Error())
	}

	return data, fidMap, nil
}
