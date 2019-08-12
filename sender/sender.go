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

	dirInfoBytes, err := ReadDirInfo2()
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

	<-done
	fmt.Println("over!")
	return nil
}

func ReadDirInfo2() ([]byte, error) {
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

	firstItem := &gproto.DirStruct{
		Name:  ".",
		Mtime: 0,
		Mode:  0,
	}
	stack.PushBack(MakeNode(firstItem, "./dir1/"))
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
				current.dirStruct.FileList = append(current.dirStruct.FileList, &gproto.FileStruct{
					Name:  f.Name(),
					Mtime: f.ModTime().Unix(),
					Mode:  0,
					Size:  f.Size(),
					//Hash:  hex.EncodeToString(md5hash.Sum(nil)),
				})

			}
		}
	}

	//printTree(rootDir, 0)
	rootDir.Name = "/home/darren/syncing/dir1"
	data, err := proto.Marshal(rootDir)
	if err != nil {
		//panic("Marsha1 error")
		return nil, errors.New("Marshl error: " + err.Error())
	}

	return data, nil
}

/*
 *  读取目录文件信息
 */
/*
func ReadDirectoryInfo() ([]byte, error) {
	var rootDir *gproto.DirStruct
	var stack = list.New()

	topDir := &gproto.DirStruct{
		Path:  "./",
		Name:  ".",
		Mtime: 0,
		Mode:  0,
	}
	stack.PushBack(topDir)
	rootDir = topDir

	var ds *gproto.DirStruct
	for stack.Back() != nil {
		ds = stack.Back().Value.(*gproto.DirStruct)
		stack.Remove(stack.Back())

		dir, err := ioutil.ReadDir(ds.Path)
		if err != nil {
			//panic("ReadDir error")
			return nil, errors.New("ReadDir error: " + err.Error())
		}
		for _, f := range dir {
			if f.IsDir() {
				if f.Name() == ".svn" || f.Name() == ".git" || f.Name() == "gproto" {
					continue
				}
				d := &gproto.DirStruct{
					Path:  ds.Path + f.Name() + "/",
					Name:  f.Name(),
					Mtime: 0,
					Mode:  0,
				}
				stack.PushBack(d)

				ds.DirList = append(ds.DirList, d)

			} else {
				//var fileSuffix string
				var fileSuffix string = path.Ext(f.Name()) //获取文件后缀
				if fileSuffix == ".o" || fileSuffix == ".exe" || fileSuffix == ".msi" {
					continue
				}

				ds.FileList = append(ds.FileList, &gproto.FileStruct{
					Name:  f.Name(),
					Mtime: f.ModTime().Unix(),
					Mode:  0,
					Size:  f.Size(),
					//Hash:  hex.EncodeToString(md5hash.Sum(nil)),
				})
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
*/
