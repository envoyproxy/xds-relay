package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
)

func readyHandler() http.HandlerFunc {
	ready := true
	var mu sync.Mutex
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			mu.Lock()
			defer mu.Unlock()
			if ready {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		case http.MethodPost:
			desiredFromURL := filepath.Base(req.URL.Path)
			desired, err := strconv.ParseBool(desiredFromURL)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				s, _ := stringify.InterfaceToString(&marshallable.Error{
					Message: "Only true/false values accepted for ready endpoint",
				})
				_, _ = w.Write([]byte(s))
				return
			}
			mu.Lock()
			defer mu.Unlock()
			ready = desired
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
