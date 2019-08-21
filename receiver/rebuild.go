package receiver

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"syncing/gproto"
	"time"
)

func RebuildFile(i int, patchList *gproto.PatchList, waitGroup *sync.WaitGroup) error {
	defer waitGroup.Done()
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
	_ = i
	//fmt.Fprintf(os.Stderr, "write file %d fid:%d   %s   %d  %s\n", i, patchList.Fid, path, size, newHash)
	return nil
}
