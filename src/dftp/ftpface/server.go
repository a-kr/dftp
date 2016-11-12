package ftpface

/* Public FTP interface to distributed file system.
*
* Read-only.
 */

import (
	"dftp/cluster"
	"dftp/dfsfat"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	goftp "github.com/goftp/server"
)

var (
	ReadOnlyAccessError = fmt.Errorf("read-only access")
	NotImplementedError = fmt.Errorf("not implemented")
	NotFoundError       = fmt.Errorf("not found")
	NotAFileError       = fmt.Errorf("not a file")
	NotADirectoryError  = fmt.Errorf("not a firectory")
)

type Server struct {
	DfsRoot   *dfsfat.TreeNode
	Cluster   *cluster.Cluster
	ftpserver goftp.Server
}

var _ goftp.DriverFactory = &Server{}

func (s *Server) ServeFtp(addr string) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		log.Fatalf("invalid ftp address (should be of form host:port)")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Fatalf("invalid ftp port: %s", err)
	}

	ftp := goftp.NewServer(&goftp.ServerOpts{
		Factory: s,
		Port:    port,
		Auth:    &Auth{},
	})
	log.Printf("FTP public interface listening on %s...", addr)
	err = ftp.ListenAndServe()
	if err != nil {
		log.Fatalf("ftp: %s", err)
	}
}

func (s *Server) NewDriver() (goftp.Driver, error) {
	d := &Driver{
		Server: s,
	}
	return d, nil
}

type Auth struct {
}

func (a *Auth) CheckPasswd(login, pass string) (bool, error) {
	return true, nil
}

type Driver struct {
	Server *Server
}

var _ goftp.Driver = &Driver{}


func (d *Driver) normalizePath(path string) string {
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")
	return path
}

func (d *Driver) Init(conn *goftp.Conn) {
}

func (d *Driver) Stat(path string) (goftp.FileInfo, error) {
	path = d.normalizePath(path)
	entry := d.Server.DfsRoot.Seek(path)
	if entry == nil {
		return nil, NotFoundError
	}
	ro := entry.GetReadonly()
	return &ro.FileStat, nil
}

func (d *Driver) ChangeDir(path string) error {
	path = d.normalizePath(path)
	entry := d.Server.DfsRoot.Seek(path)
	if entry == nil {
		return NotFoundError
	}
	ro := entry.GetReadonly()
	if !ro.IsDir() {
		return NotADirectoryError
	}
	return nil
}

func (d *Driver) ListDir(path string, callback func(goftp.FileInfo) error) error {
	path = d.normalizePath(path)
	entry := d.Server.DfsRoot.Seek(path)
	if entry == nil {
		return NotFoundError
	}
	ro := entry.GetReadonly()
	if !ro.IsDir() {
		return NotADirectoryError
	}

	for _, entry := range ro.ChildNodes {
		entryStat := entry.GetFilestat()
		if entryStat.IsDeleted() { // file was removed
			continue
		}
		err := callback(entryStat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) DeleteDir(path string) error {
	path = d.normalizePath(path)
	return ReadOnlyAccessError
}

func (d *Driver) DeleteFile(path string) error {
	path = d.normalizePath(path)
	return ReadOnlyAccessError
}

func (d *Driver) Rename(pathFrom, pathTo string) error {
	pathFrom = d.normalizePath(pathFrom)
	pathTo = d.normalizePath(pathTo)
	return ReadOnlyAccessError
}

func (d *Driver) MakeDir(path string) error {
	path = d.normalizePath(path)
	return ReadOnlyAccessError
}

func (d *Driver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	path = d.normalizePath(path)
	if offset > 0 {
		return 0, nil, NotImplementedError
	}
	entry := d.Server.DfsRoot.Seek(path)
	if entry == nil {
		return 0, nil, NotFoundError
	}
	ro := entry.GetReadonly()
	if ro.IsDir() {
		return 0, nil, NotAFileError
	}
	f, err := d.Server.Cluster.Proxy.OpenRead(path, ro, 0)
	return ro.FileStat.SizeInBytes, f, err
}

func (d *Driver) PutFile(path string, data io.Reader, appendData bool) (int64, error) {
	path = d.normalizePath(path)
	return 0, NotImplementedError
}
