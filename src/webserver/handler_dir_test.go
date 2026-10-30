package webserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/webserver"
)

// fakeDirLibrary is a library implementation which only supports the
// methods needed by the directory browsing handler.
type fakeDirLibrary struct {
	dirs           []library.Dir
	tracks         []library.SearchResult
	lastDirPath    string
	lastSearchTerm string
}

func (f *fakeDirLibrary) AddLibraryPath(string)                      {}
func (f *fakeDirLibrary) Search(string) []library.SearchResult       { return nil }
func (f *fakeDirLibrary) GetFilePath(int64) string                   { return "" }
func (f *fakeDirLibrary) GetAlbumFiles(int64) []library.SearchResult { return nil }
func (f *fakeDirLibrary) Scan()                                      {}
func (f *fakeDirLibrary) AddMedia(string) error                      { return nil }
func (f *fakeDirLibrary) Initialize() error                          { return nil }
func (f *fakeDirLibrary) Truncate() error                            { return nil }
func (f *fakeDirLibrary) Close()                                     {}

func (f *fakeDirLibrary) ListDirs(searchTerm string) []library.Dir {
	f.lastSearchTerm = searchTerm
	return f.dirs
}

func (f *fakeDirLibrary) DirTracks(dirPath, searchTerm string) []library.SearchResult {
	f.lastDirPath = dirPath
	f.lastSearchTerm = searchTerm
	return f.tracks
}

func TestDirHandlerListDirs(t *testing.T) {
	fake := &fakeDirLibrary{
		dirs: []library.Dir{
			{Path: "/music/one", Count: 2},
			{Path: "/music/two", Count: 5},
		},
	}

	handler := webserver.NewDirHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/v1/dirs", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 but got %d", resp.Code)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %s", err)
	}

	var dirs []library.Dir
	if err := json.Unmarshal(body, &dirs); err != nil {
		t.Fatalf("parsing response: %s, body: %s", err, body)
	}

	if len(dirs) != len(fake.dirs) {
		t.Fatalf("expected %d dirs but got %d", len(fake.dirs), len(dirs))
	}

	for i := range fake.dirs {
		if dirs[i] != fake.dirs[i] {
			t.Errorf("expected %+v but got %+v", fake.dirs[i], dirs[i])
		}
	}
}

func TestDirHandlerEmptyList(t *testing.T) {
	handler := webserver.NewDirHandler(&fakeDirLibrary{})

	req := httptest.NewRequest(http.MethodGet, "/v1/dirs", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if body := resp.Body.String(); body != "[]\n" {
		t.Errorf("expected an empty array but got: %q", body)
	}
}

func TestDirHandlerTracks(t *testing.T) {
	fake := &fakeDirLibrary{
		tracks: []library.SearchResult{
			{ID: 1, Title: "One", Path: "/music/one/one.mp3"},
			{ID: 2, Title: "Two", Path: "/music/one/two.mp3"},
		},
	}

	handler := webserver.NewDirHandler(fake)

	dirPath := "/music/one"
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/v1/dir/tracks?path=%s", url.QueryEscape(dirPath)),
		nil,
	)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 but got %d", resp.Code)
	}

	if fake.lastDirPath != dirPath {
		t.Errorf("expected the library to be asked for %q but got %q",
			dirPath, fake.lastDirPath)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %s", err)
	}

	var tracks []library.SearchResult
	if err := json.Unmarshal(body, &tracks); err != nil {
		t.Fatalf("parsing response: %s, body: %s", err, body)
	}

	if len(tracks) != len(fake.tracks) {
		t.Fatalf("expected %d tracks but got %d", len(fake.tracks), len(tracks))
	}

	for i := range fake.tracks {
		if tracks[i].ID != fake.tracks[i].ID {
			t.Errorf("expected track %d but got %d", fake.tracks[i].ID, tracks[i].ID)
		}
		if tracks[i].Path != fake.tracks[i].Path {
			t.Errorf("expected path %q but got %q", fake.tracks[i].Path, tracks[i].Path)
		}
	}
}

func TestDirHandlerPassesSearchTerm(t *testing.T) {
	tests := []struct {
		desc string
		url  string
	}{
		{
			desc: "on directory listing",
			url:  "/v1/dirs?q=rock",
		},
		{
			desc: "on directory tracks",
			url:  "/v1/dir/tracks?path=%2Fmusic%2Fone&q=rock",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fake := &fakeDirLibrary{}
			handler := webserver.NewDirHandler(fake)

			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200 but got %d", resp.Code)
			}

			if fake.lastSearchTerm != "rock" {
				t.Errorf("expected the library to be asked for %q but got %q",
					"rock", fake.lastSearchTerm)
			}
		})
	}
}

func TestDirHandlerTracksWithoutPath(t *testing.T) {
	handler := webserver.NewDirHandler(&fakeDirLibrary{})

	req := httptest.NewRequest(http.MethodGet, "/v1/dir/tracks", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 but got %d", resp.Code)
	}
}
