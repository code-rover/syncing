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
	"strings"
	"sync"
	"syncing/comm"
	"syncing/gproto"
	"time"

	"github.com/golang/protobuf/proto"
)

var (
	step           = 10
	conn           *comm.Connection
	fidPathMap     = make(map[int32]string)
	localBasePath  string
	remoteBasePath string
	goMaxNum       = 5
)

type Params struct {
	Step           int
	ExecPath       string
	Delete         bool
	LocalBasePath  string
	RemoteBasePath string
	Port           int
}

func Start(params *Params) error {
	step = params.Step
	localBasePath = params.LocalBasePath

	conns := strings.Split(params.RemoteBasePath, ":")
	remoteBasePath = conns[1]

	var stdout io.Reader
	var stdin io.Writer

	subProcessDone := make(chan bool)
	pipeDone := make(chan bool)

	go func() {
		sshPort := fmt.Sprintf("-p %d", params.Port)
		cmd := exec.Command("ssh", sshPort, conns[0], params.ExecPath+" --server")
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

	t1 := time.Now()
	//初始化参数
	param := gproto.InitParam{}
	param.BasePath = remoteBasePath
	param.Step = int32(step)
	param.Delete = params.Delete

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
	elapsed := time.Since(t1)
	fmt.Printf("Done  %s\n", elapsed)
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
		fmt.Printf("recv cmd:%d\n", cmd)

		if cmd == gproto.MSG_B_SUMLIST {
			fileSumList := st.(*gproto.FileSumList)

			for _, sumList := range fileSumList.List {
				fPath := fidPathMap[sumList.Fid]
				fInfo, err := os.Lstat(fPath)
				if err != nil {
					fmt.Printf("not found path: %s\n", fPath)
					return err
				}
				if fInfo.Mode()&os.ModeSymlink != 0 {
					fmt.Printf("symlink file skip : %s\n", fPath)
					continue
				}

				waitGroup.Add(1)
				goLimit <- true

				go func(path string, sumList *gproto.SumList) {
					defer waitGroup.Done()
					defer func() {
						<-goLimit
					}()

					// fmt.Printf("Readfile %s  size: %d\n", fidMap[sumList.Fid], len(fdata))
					patchList := MakePatch(path, sumList)
					patchList.Fid = sumList.Fid
					// fmt.Printf(">patch size: %d\n", len(patchList.List))

					patchListBytes, err := proto.Marshal(patchList)
					if err != nil {
						fmt.Printf("Marshal error patchList: %s\n", err.Error())
						panic("Marshal error: " + err.Error())
						//return err
					}
					//fmt.Printf(">patch after Marsha1size:%s   %d\n", fidMap[sumList.Fid], len(patchListBytes))

					mutex.Lock()
					defer mutex.Unlock()
					_, err = conn.Send(gproto.MSG_A_PATCHLIST, patchListBytes)
					if err != nil {
						fmt.Printf("send error MSG_A_PATCHLIST: %s\n", err.Error())
						// return err
					}

				}(fPath, sumList)
			}

			waitGroup.Wait()
			conn.Send(gproto.MSG_A_END, []byte{})

		} else if cmd == gproto.MSG_B_END {
			syncResult := st.(*gproto.SyncResult)
			time.Sleep(100 * time.Millisecond)
			fmt.Println("sync done")
			fmt.Println("==========================================================")
			fmt.Printf("success count: %d\n", syncResult.SuccNum)
			fmt.Printf("failed count: %d\n", len(syncResult.FailedList))
			if len(syncResult.FailedList) > 0 {
				if len(syncResult.FailedList) < 10 {
					for i, fid := range syncResult.FailedList {
						fmt.Printf("%d: %s\n", i, fidPathMap[fid])
					}
				}
			}
			fmt.Printf("removed count: %d\n", len(syncResult.RemovedList))
			if len(syncResult.RemovedList) > 0 {
				if len(syncResult.RemovedList) < 10 {
					for i, path := range syncResult.RemovedList {
						fmt.Printf("%d: %s\n", i, path)
					}
				}
			}
			fmt.Printf("totalSend: %d bytes\n", conn.TotalSend)
			fmt.Printf("totalRecv: %d bytes\n", conn.TotalRecv)
			fmt.Println("==========================================================")

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
				// if f.Name() == ".git" /* || f.Name() == "pkg"*/ {
				// 	continue
				// }
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
