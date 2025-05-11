package workspaces

import (
	"fmt"
	"os"
	"path"
	"time"
)

type Workspace struct {
	Parent   string
	DirEntry os.DirEntry
}

func (w *Workspace) Path() string {
	return path.Join(w.Parent, w.DirEntry.Name())
}

func (w *Workspace) TruncatedName(maxlen int, marker string) string {
	if len(w.DirEntry.Name()) <= maxlen {
		return w.DirEntry.Name()
	}
	return fmt.Sprintf("%s%s", w.DirEntry.Name()[:maxlen-len(marker)], marker)
}

func (w *Workspace) ModTime() time.Time {
	info, err := w.DirEntry.Info()
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
func isIgnored(d os.DirEntry) bool {
	return d.Name() == ".DS_Store" || !d.IsDir()
}

func Load(path string) ([]Workspace, error) {
	o, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	w := make([]Workspace, 0, len(o))
	for i := range o {
		if isIgnored(o[i]) {
			continue
		}
		w = append(w, Workspace{DirEntry: o[i], Parent: path})
	}
	return w, nil
}
