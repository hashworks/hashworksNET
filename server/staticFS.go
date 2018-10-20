package server

import (
	"github.com/hashworks/hashworksNET/server/bindata"
	"net/http"
	"os"
)

type prefixHTTPFS struct {
	prefix string
}

func (phfs *prefixHTTPFS) Open(path string) (http.File, error) {
	f, err := bindata.FS.OpenFile(bindata.CTX, phfs.prefix+path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	return f, nil
}
