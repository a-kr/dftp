package dfsfat

/*
* File structures representing distributed filesystem tree.
*/

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	MultipleNodeOwners = "(multiple)"
)

type TreeNode struct {
	sync.RWMutex
	fileStat   FileStat
	childNodes map[string]*TreeNode
}

func NewRootNode() *TreeNode {
	n := &TreeNode{}
	n.fileStat.Dir = true
	return n
}

type TreeNodeReadonly struct {
	FileStat
	ChildNodes map[string]*TreeNode
}

type FileStat struct {
	Basename        string
	Dir             bool
	LastModified    int64
	LastInfoUpdated int64
	SizeInBytes     int64
	FileMode        os.FileMode
	OwnerNode       string
}

func (n *TreeNode) GetReadonly() *TreeNodeReadonly {
	n.RLock()
	defer n.RUnlock()
	ro := TreeNodeReadonly{
		FileStat:   n.fileStat,
		ChildNodes: nil,
	}
	if n.childNodes != nil {
		ro.ChildNodes = make(map[string]*TreeNode, len(n.childNodes))
		for k, v := range n.childNodes {
			ro.ChildNodes[k] = v
		}
	}
	return &ro
}

func (n *TreeNode) GetFilestat() *FileStat {
	n.RLock()
	defer n.RUnlock()
	s := n.fileStat // copy
	return &s
}

func (n *TreeNode) IsDir() bool {
	n.RLock()
	defer n.RUnlock()
	return n.fileStat.IsDir()
}

func (n *TreeNode) Seek(path string) *TreeNode {
	if path == "" {
		return n
	}
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	return n.seek(parts)
}

func (n *TreeNode) seek(path []string) *TreeNode {
	part0 := path[0]
	n.RLock()
	entry, ok := n.childNodes[part0]
	n.RUnlock()
	if !ok {
		return nil
	}
	if len(path) == 1 {
		return entry
	}
	return entry.seek(path[1:])
}

func (n *TreeNode) Walk(callback filepath.WalkFunc) {
	n.walk(callback, "")
}

func (n *TreeNode) walk(callback filepath.WalkFunc, basepath string) {
	ro := n.GetReadonly()
	if basepath != "" {
		err := callback(basepath, &ro.FileStat, nil)
		if err == filepath.SkipDir {
			return
		}
	}
	for name, entry := range ro.ChildNodes {
		entry.walk(callback, filepath.Join(basepath, name))
	}
}
