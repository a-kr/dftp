package localfs

import (
	"dftp/dfsfat"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *LocalFs) ScanOnce() {
	files := make([]*dfsfat.FileAnnouncement, 0)

	scanT := time.Now().Unix()

	log.Printf("Scanner: starting local scan...")

	err := filepath.Walk(s.LocalRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		fa := &dfsfat.FileAnnouncement{
			FullName: path,
			Deletion: false,
		}
		fa.OwnerNode = s.MyNodeName
		fa.Dir = info.IsDir()
		fa.FileMode = info.Mode()
		fa.Basename = info.Name()
		fa.LastModified = info.ModTime().Unix()
		if !info.IsDir() {
			fa.SizeInBytes = info.Size()
		}
		fa.LastInfoUpdated = scanT
		fa.FullName = strings.TrimPrefix(path, s.LocalRoot)
		fa.FullName = filepath.Join(s.DfsMountPoint, fa.FullName)
		if fa.FullName != "" {
			files = append(files, fa)
		}
		return nil
	})
	log.Printf("Scanner: local scan finished, %d file(s) found", len(files))
	if err != nil {
		log.Fatalf("Scanner: scan error: %s", err)
	} else {
		s.lastScanMutex.Lock()
		s.LastFullScan = files
		s.LastFullScanTime = scanT
		s.lastScanMutex.Unlock()

		s.DfsRoot.Update(files)

		// TODO: notify peers if partial update is available
	}
}
