package localfs

/*
* Local file system manager and scanner
*/

import (
	"dftp/dfsfat"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalFs struct {
	LocalRoot     string
	DfsMountPoint string
	DfsRoot       *dfsfat.TreeNode
	MyNodeName    string
}

func NewLocalFs(localRoot string, dfsMountPoint string, dfsRoot *dfsfat.TreeNode, myNodeName string) *LocalFs {
	s := &LocalFs{
		LocalRoot:     localRoot,
		DfsMountPoint: dfsMountPoint,
		DfsRoot:       dfsRoot,
		MyNodeName:    myNodeName,
	}
	if strings.HasPrefix(s.DfsMountPoint, "/") {
		s.DfsMountPoint = strings.TrimPrefix(s.DfsMountPoint, "/")
	}
	if !strings.HasSuffix(s.LocalRoot, "/") {
		s.LocalRoot += "/"
	}
	return s
}

var (
	LocalFileNotFoundError = fmt.Errorf("local file not found")
)

func (fs *LocalFs) OpenRead(dfsPath string) (io.ReadCloser, error) {
	if fs.DfsMountPoint != "" {
		if !strings.HasPrefix(dfsPath, fs.DfsMountPoint) {
			return nil, LocalFileNotFoundError
		}
		dfsPath = strings.TrimPrefix(dfsPath, fs.DfsMountPoint)
	}
	localFilename := filepath.Join(fs.LocalRoot, dfsPath)
	f, err := os.Open(localFilename)
	return f, err
}
