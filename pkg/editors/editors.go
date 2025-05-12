package editors

import "os"

type Editor interface {
	Command() string
	CreateTemp() (*os.File, error)
	OpenFileArgs(f string) []string
}

type Helix struct{}

func (e Helix) Command() string {
	return "hx"
}

func (e Helix) CreateTemp() (*os.File, error) {
	f, err := os.CreateTemp("", "*workspacescli")
	if err != nil {
		return nil, err
	}
	return f, f.Close()
}

func (e Helix) OpenFileArgs(f string) []string {
	return []string{f}
}
