package receiver

import (
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func RemoveFiles() error {
	defer taskWaitGroup.Done()

	var stack = list.New()
	type DNode struct {
		path string
	}

	MakeNode := func(p string) *DNode {
		return &DNode{
			path: p,
		}
	}

	stack.PushBack(MakeNode(basePath + "/"))

	var current *DNode
	for stack.Back() != nil {
		current = stack.Back().Value.(*DNode)
		stack.Remove(stack.Back())

		dir, err := ioutil.ReadDir(current.path)
		if err != nil {
			return errors.New("ReadDir error: " + err.Error())
		}
		for _, f := range dir {
			if f.IsDir() {
				// if f.Name() == ".git" /* || f.Name() == "pkg"*/ {
				// 	continue
				// }
				newNode := MakeNode(current.path + f.Name() + "/")
				stack.PushBack(newNode)

			} else {
				fullPath := current.path + f.Name()
				if _, ok := pathMap[fullPath]; !ok {
					err := os.Remove(fullPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "delete path failed: %s\n", fullPath)
						continue
					}

					syncResultMutex.Lock()
					syncResult.RemovedList = append(syncResult.RemovedList, fullPath)
					syncResultMutex.Unlock()
					for dir := path.Dir(fullPath); len(dir) > 5; dir = path.Dir(dir) {
						fileInfo, _ := ioutil.ReadDir(dir)

						hasFile := false
						for _, f := range fileInfo {
							if !f.IsDir() {
								hasFile = true
								break
							}
						}
						if hasFile {
							break
						}
						os.Remove(dir)
					}
					fmt.Fprintf(os.Stderr, "delete path: %s\n", fullPath)
				}
			}
		}
	}

	//printTree(rootDir, 0)

	return nil
}
