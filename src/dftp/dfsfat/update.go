package dfsfat

/*
* Filesystem tree update algorithm
 */

import (
	"log"
	"strings"
)

type FileAnnouncement struct {
	FullName string
	FileStat
	fullNameParts []string
	Deletion      bool
}

func (fa *FileAnnouncement) ensureInit() {
	if fa.fullNameParts == nil {
		fa.fullNameParts = strings.Split(strings.TrimPrefix(fa.FullName, "/"), "/")
	}
	if fa.Deletion {
		fa.FileStat.SizeInBytes = -1
	}
}

func (fa *FileAnnouncement) namePart() string {
	return fa.fullNameParts[0]
}

func (fa *FileAnnouncement) isLeaf() bool {
	return len(fa.fullNameParts) == 1
}

func (fa *FileAnnouncement) nameShift() {
	fa.fullNameParts = fa.fullNameParts[1:]
}

// Update() is called either by local filesystem scanner or upon receiving messages
// from DFS peers.
func (n *TreeNode) Update(files []*FileAnnouncement) {
	log.Printf("FAT: update starting (%d items)", len(files))
	// TODO sort files by fullname
	for _, fa := range files {
		fa.ensureInit()
	}
	n.update(files)
	log.Printf("FAT: update finished (%d items)", len(files))
}

type fileGroup struct {
	name  string
	files []*FileAnnouncement
}

func groupAnnouncementsByFilepart(files []*FileAnnouncement) []fileGroup {
	// assumes files are sorted by namePart()
	groups := make([]fileGroup, 0, 1)
	prevName := ""
	prevGroup := fileGroup{}
	for _, fa := range files {
		curName := fa.namePart()
		if curName != prevName && len(prevGroup.files) > 0 {
			groups = append(groups, prevGroup)
			prevGroup = fileGroup{}
		}
		prevGroup.name = curName
		prevGroup.files = append(prevGroup.files, fa)
		prevName = curName
	}
	if len(prevGroup.files) > 0 {
		groups = append(groups, prevGroup)
	}
	return groups
}

func (n *TreeNode) update(files []*FileAnnouncement) {
	for _, faGroup := range groupAnnouncementsByFilepart(files) {
		n.RLock()
		entry, ok := n.childNodes[faGroup.name]
		n.RUnlock()
		if !ok {
			n.Lock()
			entry, ok = n.childNodes[faGroup.name]
			if !ok {
				if n.childNodes == nil {
					n.childNodes = make(map[string]*TreeNode)
				}
				entry = &TreeNode{}
				entry.fileStat.Basename = faGroup.name
				n.childNodes[faGroup.name] = entry
			}
			n.Unlock()
		}

		nestedFiles := make([]*FileAnnouncement, 0, len(faGroup.files))

		for _, fa := range faGroup.files {
			if !fa.isLeaf() && !fa.Deletion {
				fa.nameShift()
				nestedFiles = append(nestedFiles, fa)
			} else {
				entry.Lock()
				// TODO check if file actually changed
				if fa.LastInfoUpdated > entry.fileStat.LastInfoUpdated {
					prevOwner := entry.fileStat.OwnerNode
					entry.fileStat = fa.FileStat
					if fa.FileStat.Dir && prevOwner != "" && prevOwner != fa.FileStat.OwnerNode {
						entry.fileStat.OwnerNode = MultipleNodeOwners
						// TODO: maybe file moved from one node to another? Need periodic ownership recalculation.
					}
				}
				entry.Unlock()
			}
		}

		if len(nestedFiles) > 0 {
			entry.setAsDir()
			entry.update(nestedFiles)
			entry.recalculateOwner()
		}
	}
}

func (n *TreeNode) setAsDir() {
	n.Lock()
	defer n.Unlock()
	n.fileStat.Dir = true
	n.fileStat.SizeInBytes = 0
	if n.childNodes == nil {
		n.childNodes = make(map[string]*TreeNode)
	}
}

func (n *TreeNode) recalculateOwner() {
	owner := ""
	n.Lock()
	defer n.Unlock()
	if !n.fileStat.Dir {
		return
	}
	for _, e := range n.childNodes {
		stat := e.GetFilestat()
		if owner == "" {
			owner = stat.OwnerNode
		} else if owner != stat.OwnerNode {
			owner = MultipleNodeOwners
		}
	}
	n.fileStat.OwnerNode = owner
}
