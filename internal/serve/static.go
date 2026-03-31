package serve

import (
	"log/slog"
	"net/http"
	"os"
)

func staticFSHandler(rootPath string) http.Handler {
	if _, err := os.Stat(rootPath); err != nil {
		slog.Warn("Static assets path not found", "path", rootPath, "err", err)
	}
	fs := os.DirFS(rootPath)
	return http.FileServer(justFilesFS{http.FS(fs)})
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
