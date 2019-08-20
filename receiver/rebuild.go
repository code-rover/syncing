package receiver

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"syncing/gproto"
)

//
func RebuildFile(patchList *gproto.PatchList) error {
	path := fidPathMap[patchList.Fid]

	if path == "" {
		path = "./newfile"
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
	//var buf bytes.Buffer
	for _, patch := range patchList.List {
		if patch.Pos == -1 {
			//buf.Write(patch.Data)
			wbuf.Write(patch.Data)
			// println("data: " + string(patch.data))
		} else {
			// println("patch: " + string(origin[patch.pos:patch.pos+patch.len]))
			//buf.Write(fdata[patch.Pos : patch.Pos+patch.Len])
			wbuf.Write(fdata[patch.Pos : patch.Pos+patch.Len])
		}
	}
	wbuf.Flush()

	//ioutil.WriteFile(path+"_new", buf.Bytes(), 0666)
	return nil
}
