package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

// DirHandler is a http.Handler responsible for directory browsing requests.
// It exposes the file system structure of the media library so the web
// interface could offer a directory based view of the library.
type DirHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (dh DirHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if req.URL.Path == APIv1EndpointDirTracks {
		dh.dirTracks(writer, req)
		return
	}

	dh.listDirs(writer, req)
}

// listDirs returns the list of directories which contain media files,
// together with the number of files in each one.
func (dh DirHandler) listDirs(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	dirs := dh.library.ListDirs(req.URL.Query().Get("q"))
	if dirs == nil {
		dirs = []library.Dir{}
	}

	enc := json.NewEncoder(writer)
	_ = enc.Encode(dirs)
}

// dirTracks returns all the media files in the directory given with the
// "path" query argument and all of its sub-directories.
func (dh DirHandler) dirTracks(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	dirPath := req.URL.Query().Get("path")
	if dirPath == "" {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error": "missing 'path' parameter"}`))
		return
	}

	results := dh.library.DirTracks(dirPath, req.URL.Query().Get("q"))
	if results == nil {
		results = []library.SearchResult{}
	}

	enc := json.NewEncoder(writer)
	_ = enc.Encode(results)
}

// NewDirHandler returns a new DirHandler. It needs a library.Library to
// browse the directories of.
func NewDirHandler(lib library.Library) *DirHandler {
	return &DirHandler{
		library: lib,
	}
}
