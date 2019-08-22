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
	"sync"
	"syncing/comm"
	"syncing/gproto"

	"github.com/golang/protobuf/proto"
)

var (
	step           = 10
	conn           *comm.Connection
	fidPathMap     = make(map[int32]string)
	localBasePath  string
	remoteBasePath string
	goMaxNum       = 100
)

func Start(ip string, port string, user string, execPath string, lpath string, rpath string, myStep int) error {
	step = myStep
	localBasePath = lpath
	remoteBasePath = rpath
	var stdout io.Reader
	var stdin io.Writer

	subProcessDone := make(chan bool)
	pipeDone := make(chan bool)

	go func() {
		cmd := exec.Command("ssh", "-p"+port, user+"@"+ip, execPath+" --server")
		stdout, _ = cmd.StdoutPipe()
		stdin, _ = cmd.StdinPipe()
		cmd.Stderr = os.Stderr
		pipeDone <- true

		err := cmd.Run()
		if err != nil {
			panic(err)
		}
		subProcessDone <- true
	}()

	//初始化参数
	param := gproto.InitParam{}
	param.BasePath = remoteBasePath
	param.Step = int32(step)

	fmt.Printf("localBasePath: %s\n", localBasePath)
	fmt.Printf("remoteBasePath: %s\n", remoteBasePath)

	paramData, err := proto.Marshal(&param)
	if err != nil {
		return errors.New("Marshal InitParam error: " + err.Error())
	}

	dirTreeData, err := ReadDirInfo()
	if err != nil {
		return errors.New(err.Error())
	}

	<-pipeDone //等待管道建立完成
	rBuf := bufio.NewReader(stdout)
	wBuf := bufio.NewWriter(stdin)
	conn = comm.NewConn(rBuf, wBuf)

	_, err = conn.Send(gproto.MSG_A_INITPARAM, paramData)
	if err != nil {
		panic(err.Error())
	}

	_, err = conn.Send(gproto.MSG_A_DIR_INFO, dirTreeData)
	if err != nil {
		panic(err.Error())
	}

	ProcessMsg(conn)

	<-subProcessDone
	fmt.Println("over!")
	return nil
}

func ProcessMsg(conn *comm.Connection) error {
	var waitGroup sync.WaitGroup = sync.WaitGroup{}
	var mutex sync.Mutex = sync.Mutex{}
	goLimit := make(chan bool, goMaxNum)

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

				waitGroup.Add(1)
				goLimit <- true

				go func(path string, sumList *gproto.SumList) {
					defer waitGroup.Done()
					<-goLimit
					fdata, err := ioutil.ReadFile(path)
					if err != nil {
						fmt.Printf("ReadFile error: %s\n", err.Error())
						// return err
					}
					// fmt.Printf("Readfile %s  size: %d\n", fidMap[sumList.Fid], len(fdata))
					patchList := MakePatch(fdata, sumList)
					patchList.Fid = sumList.Fid
					// fmt.Printf(">patch size: %d\n", len(patchList.List))

					patchListBytes, err := proto.Marshal(patchList)
					if err != nil {
						fmt.Printf("Marshal error patchList: %s\n", err.Error())
						// return err
					}
					//fmt.Printf(">patch after Marsha1size:%s   %d\n", fidMap[sumList.Fid], len(patchListBytes))

					mutex.Lock()
					defer mutex.Unlock()
					_, err = conn.Send(gproto.MSG_A_PATCHLIST, patchListBytes)
					if err != nil {
						fmt.Printf("send error MSG_A_PATCHLIST: %s\n", err.Error())
						// return err
					}

				}(fidPathMap[sumList.Fid], sumList)
			}

			waitGroup.Wait()
			conn.Send(gproto.MSG_A_END, []byte{})

		} else if cmd == gproto.MSG_B_END {
			// fmt.Println("recv MSG_B_END")
			return nil
		}
	}
}

func ReadDirInfo() ([]byte, error) {
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

	firstItem := &gproto.DirStruct{
		Name:  "",
		Mtime: 0,
		Mode:  0,
	}
	stack.PushBack(MakeNode(firstItem, localBasePath))
	rootDir = firstItem

	var current *DNode
	for stack.Back() != nil {
		current = stack.Back().Value.(*DNode)
		stack.Remove(stack.Back())

		dir, err := ioutil.ReadDir(current.path)
		if err != nil {
			return nil, errors.New("ReadDir error: " + err.Error())
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
				fidPathMap[fid] = current.path + f.Name()
				fid++
			}
		}
	}

	//printTree(rootDir, 0)
	data, err := proto.Marshal(rootDir)
	if err != nil {
		//panic("Marsha1 error")
		return nil, errors.New("Marshl error: " + err.Error())
	}

	return data, nil
}
