package receiver

//
import (
	"bufio"
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syncing/comm"
	"syncing/gproto"
	"time"

	"github.com/golang/protobuf/proto"
)

var (
	step        = 0
	fidPathMap  = make(map[int32]string)
	fidMtimeMap = make(map[int32]int64)
	conn        *comm.Connection
	basePath    string
	goMaxNum    = 100
)

func RunServer() {
	errwriter := bufio.NewWriter(os.Stderr)
	errwriter.WriteString("msg welcome!\n")
	errwriter.Flush()

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	conn = comm.NewConn(reader, writer)

	ProcessMsg(conn)

	fmt.Fprintf(os.Stderr, "msg server stoped\n")
	conn.Send(gproto.MSG_B_END, []byte{})
}

var rebuildWaitGroup = sync.WaitGroup{}
var rebuildGoLimit = make(chan bool, goMaxNum)

func ProcessMsg(conn *comm.Connection) error {
	for {
		cmd, st, err := conn.Recv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "msg recv err: %s\n", err.Error())
			return err
		}

		if cmd == gproto.MSG_A_INITPARAM {
			initParam := st.(*gproto.InitParam)
			step = int(initParam.Step)
			if strings.HasPrefix(initParam.BasePath, "~/") {
				initParam.BasePath = initParam.BasePath[2:]
			}
			basePath, err = filepath.Abs(initParam.BasePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "msg basePath error %s\n", err.Error())
				return err
			}

		} else if cmd == gproto.MSG_A_DIR_INFO {
			ds := st.(*gproto.DirStruct)
			fileSumList, err := FileListCheck(ds)
			if err != nil {
				fmt.Fprintf(os.Stderr, "msg FileListCheck error: %s\n", err.Error())
				return err
			}
			fidBytes, err := proto.Marshal(fileSumList)
			if err != nil {
				fmt.Fprintf(os.Stderr, "msg Marsha1 fileSumList error: %s\n", err.Error())
				return err
			}

			_, err = conn.Send(gproto.MSG_B_SUMLIST, fidBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "msg Send fidBytes  error: %s\n", err.Error())
				return err
			}

		} else if cmd == gproto.MSG_A_PATCHLIST {
			patchList := st.(*gproto.PatchList)
			rebuildWaitGroup.Add(1)
			rebuildGoLimit <- true
			go RebuildFile(patchList)

		} else if cmd == gproto.MSG_A_END {
			fmt.Fprintf(os.Stderr, "msg recv end\n")

			break
		} else {
			break
		}
	}
	rebuildWaitGroup.Wait()
	return nil
}

func RebuildFile(patchList *gproto.PatchList) error {
	defer rebuildWaitGroup.Done()
	defer func() {
		<-rebuildGoLimit
	}()

	path := fidPathMap[patchList.Fid]

	if path == "" {
		return errors.New("not found file in map")
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return errors.New(err.Error())
	}
	defer f.Close()

	wbuf := bufio.NewWriter(f)

	fdata, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "msg error "+err.Error())
		return errors.New(err.Error())
	}

	var size int64
	buf := bytes.Buffer{}
	for _, patch := range patchList.List {
		if patch.Pos == -1 {
			wbuf.Write(patch.Data)
			buf.Write(patch.Data)
			size += int64(len(patch.Data))
		} else {
			wbuf.Write(fdata[patch.Pos : patch.Pos+patch.Len])
			buf.Write(fdata[patch.Pos : patch.Pos+patch.Len])
			//fmt.Fprintf(os.Stderr, "msg patch.Data: %s\n", fdata[patch.Pos:patch.Pos+patch.Len])
			size += int64((patch.Pos + patch.Len - patch.Pos))
		}
	}
	wbuf.Flush()
	newHash := md5sum(buf.Bytes())
	if newHash != patchList.Hash {
		fmt.Fprintf(os.Stderr, "md5sum error: %s   %s\n", patchList.Hash, newHash)
	}

	f.Truncate(int64(size)) //去掉多余数据

	mtime := fidMtimeMap[patchList.Fid]
	tm := time.Unix(mtime, 0)
	err = os.Chtimes(path, tm, tm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chtimes err: %s\n", err.Error())
	}
	fmt.Fprintf(os.Stderr, "rebuild file fid:%d  patchSize:%d  %s   %d  %s\n", patchList.Fid, len(patchList.List), path, size, newHash)
	return nil
}

func FileListCheck(ds *gproto.DirStruct) (*gproto.FileSumList, error) {
	errwriter := bufio.NewWriter(os.Stderr)
	errwriter.WriteString("msg Unmarshal data success! getname: " + ds.GetName() + "\n")

	var fileSumList gproto.FileSumList

	pathStack := list.New()                        //用于计算全路径
	visitedMap := make(map[*gproto.DirStruct]bool) //记录节点是否访问过

	var waitGroup sync.WaitGroup = sync.WaitGroup{}
	var mutex sync.Mutex = sync.Mutex{}

	stack := list.New()
	stack.PushBack(ds)

	for stack.Back() != nil {
		ds := stack.Back().Value.(*gproto.DirStruct)
		stack.Remove(stack.Back())

		if _, ok := visitedMap[ds]; !ok {
			visitedMap[ds] = true
		}
		pathStack.PushBack(ds)

		fileList := ds.GetFileList()
		var pathBuf bytes.Buffer
		if len(fileList) > 0 {
			for e := pathStack.Front(); e != nil; e = e.Next() {
				pathBuf.WriteString(e.Value.(*gproto.DirStruct).Name)
				pathBuf.WriteByte(os.PathSeparator)
			}
		}

		for _, file := range fileList {
			path := strings.Join([]string{basePath, pathBuf.String(), file.Name}, "")

			fileInfo, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					//fmt.Fprintf(os.Stderr, "msg missing %s\n", path)
					fidMtimeMap[file.Fid] = file.Mtime
					fidPathMap[file.Fid] = path

					err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Mkdir err %s %s\n", filepath.Dir(path), err)
						continue
					}
					f, err := os.Create(path) //不存在 先创建
					if err != nil {
						fmt.Fprintf(os.Stderr, "Createfile err %s %s\n", path, err)
						continue
					}
					f.Close()

					var sumList gproto.SumList
					sumList.Fid = file.Fid
					mutex.Lock()
					fileSumList.List = append(fileSumList.List, &sumList)
					mutex.Unlock()
					continue

				} else {
					errwriter.WriteString("msg err " + err.Error() + "\n")
					continue
				}
			}

			if file.Mtime != fileInfo.ModTime().Unix() || file.Size != fileInfo.Size() {
				//fmt.Fprintf(os.Stderr, "msg diff %s\n", path)
				fidMtimeMap[file.Fid] = file.Mtime
				fidPathMap[file.Fid] = path

				waitGroup.Add(1)
				go func(fid int32, mypath string) {
					defer waitGroup.Done()

					sumList := MakeSumList(mypath)
					sumList.Fid = fid

					//fmt.Fprintf(os.Stderr, "msg sumList %s  %d\n", path, len(sumList.List))
					mutex.Lock()
					fileSumList.List = append(fileSumList.List, sumList)
					mutex.Unlock()
				}(file.Fid, path)
			}
		}

		//update pathStack
		if len(ds.GetDirList()) == 0 {
			for e := pathStack.Back(); e != nil; {
				item := e.Value.(*gproto.DirStruct)

				childVisited := true
				for _, child := range item.DirList {
					if !visitedMap[child] {
						childVisited = false
						break
					}
				}

				if childVisited {
					preEle := e.Prev()
					pathStack.Remove(e)
					e = preEle
				} else {
					break
				}
			}
		}

		for _, dir := range ds.GetDirList() {
			stack.PushBack(dir)
		}
	}
	errwriter.Flush()

	waitGroup.Wait()
	return &fileSumList, nil
}
