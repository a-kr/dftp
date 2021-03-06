package httpface

/*
* Public HTTP interface to distributed file system.
* Functions:
*   - directory browser
*   - file downloader
 */

import (
	"dftp/cluster"
	"dftp/dfsfat"
	"dftp/httputils"
	"dftp/utils"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	DfsRoot *dfsfat.TreeNode
	Cluster *cluster.Cluster
	mux     *http.ServeMux
}

func (s *Server) ServeHttp(addr string) {
	s.mux = http.NewServeMux()
	httputils.HandleFunc(s.mux, "/", s.Index)
	httputils.HandleFunc(s.mux, "/fs/", s.Fs)
	httputils.HandleFunc(s.mux, "/find/", s.Find)
	log.Printf("HTTP public interface listening on %s...", addr)
	if err := http.ListenAndServe(addr, s.mux); err != nil {
		log.Fatalf("http: %s", err)
	}
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `Hi! See /fs/ for filesystem browser.`, 200)
}

// Display full distributed filesystem listing as plain text
func (s *Server) Find(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	s.DfsRoot.Walk(func(path string, info os.FileInfo, _ error) error {
		fmt.Fprintf(w, "/%s\r\n", path)
		return nil
	})
}

// Display directory listing or serve a single file
func (s *Server) Fs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/fs/")
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")
	entry := s.DfsRoot.Seek(path)
	if entry == nil {
		http.Error(w, fmt.Sprintf("`%s` not found in DFS", path), 404)
		return
	}
	ro := entry.GetReadonly()

	if !ro.IsDir() {
		s.ServeFile(w, r, path, ro)
		return
	}

	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!DOCTYPE html>")
	fmt.Fprintf(w, `<html><head><meta charset="utf-8"><title>Index of /%s</title></head>`, path)
	fmt.Fprintf(w, `<body><h1>Index of /%s</h1><hr/>`, path)
	fmt.Fprintf(w, `<pre>`)
	if path != "" {
		fmt.Fprintf(w, `<a href="../">../</a>`+"\r\n")
	}

	// sort entries: directories first, then alphabetically
	type E struct {
		Name     string
		Entry    *dfsfat.TreeNode
		Sortname string
	}
	entries := []*E{}

	for name, entry := range ro.ChildNodes {
		sortname := "B" + name
		if entry.IsDir() {
			sortname = "A" + name
		}
		eee := &E{name, entry, sortname}
		entries = append(entries, eee)
	}
	utils.SortSlice(entries, func(li, ri interface{}) bool {
		l, r := li.(*E), ri.(*E)
		return l.Sortname < r.Sortname
	})

	for _, eee := range entries {
		name := eee.Name
		entry := eee.Entry
		entryStat := entry.GetFilestat()
		if entryStat.IsDeleted() { // file was removed
			continue
		}
		if entryStat.IsDir() {
			name += "/"
		}
		displayName := name
		spaces := ""
		if len(displayName) > 50 {
			displayName = displayName[:49] + "…"
		} else {
			for len(spaces) < 50-len(displayName) {
				spaces += " "
			}
		}
		dt := time.Unix(entryStat.LastModified, 0).Format("2006-01-02 15:04")
		sz := ""
		txtLink := ""
		if !entryStat.IsDir() {
			sz = fmt.Sprintf("%d", entryStat.SizeInBytes)
			txtLink = fmt.Sprintf(`<a href="%s?format=txt" title="view as plain text">txt</a>`, name)
		} else {
			sz = "[DIR]"
			txtLink = "   "
		}

		fmt.Fprintf(w, `<a href="%s">%s</a>%s%20s%20s %s   %s`+"\r\n", name, displayName, spaces, dt, sz, txtLink, entryStat.OwnerNode)
	}
	fmt.Fprintf(w, `</pre><hr/></body></html>`)
}

func (s *Server) ServeFile(w http.ResponseWriter, r *http.Request, path string, entry *dfsfat.TreeNodeReadonly) {
	redirN, err := strconv.Atoi(r.FormValue("redirN"))
	if err != nil {
		redirN = 0
	}

	f, err := s.Cluster.Proxy.OpenRead(path, entry, redirN)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()

	ctype := mime.TypeByExtension(filepath.Ext(path))
	if r.FormValue("format") == "txt" {
		ctype = "text/plain; charset=utf-8"
	}
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	return
}
