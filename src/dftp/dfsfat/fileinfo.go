package dfsfat

import (
	"os"
	"time"
)

// implementation of os.FileInfo

func (fs *FileStat) Name() string {
	return fs.Basename
}

func (fs *FileStat) Size() int64 {
	return fs.SizeInBytes
}

func (fs *FileStat) Mode() os.FileMode {
	return fs.FileMode
}

func (fs *FileStat) ModTime() time.Time {
	return time.Unix(fs.LastModified, 0)
}

func (fs *FileStat) IsDir() bool {
	return fs.Dir
}

func (fs *FileStat) Sys() interface{} {
	return fs
}

func (fs *FileStat) IsDeleted() bool {
	return fs.SizeInBytes < 0
}

// Additional methods to implement goftp.FileInfo

func (fs *FileStat) Owner() string {
	return "dftp"
}

func (fs *FileStat) Group() string {
	return "dftp"
}
