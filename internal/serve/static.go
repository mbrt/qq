package serve

import (
	"io/fs"
	"net/http"
	"os"
)

func staticFSHandler(fsys fs.FS) http.Handler {
	return http.FileServer(justFilesFS{http.FS(fsys)})
}

type justFilesFS struct {
	fs http.FileSystem
}

func (fs justFilesFS) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return justFilesFile{f}, nil
}

type justFilesFile struct {
	http.File
}

func (f justFilesFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, nil
}
