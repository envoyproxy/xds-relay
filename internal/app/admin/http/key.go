package handler

import (
	"fmt"
	"net/http"

	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
)

func keyDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		allKeys, err := orchestrator.Orchestrator.GetDownstreamAggregatedKeys(*o)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: fmt.Sprintf("error in getting cache keys: %s", err.Error()),
			})
			_, _ = w.Write([]byte(errMessage))
			return
		}

		keys := make([]string, 0)
		for k := range allKeys {
			keys = append(keys, k)
		}

		response := &marshallable.Key{
			Names: keys,
		}
		marshalledKeys, err := stringify.InterfaceToString(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errMessage, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: fmt.Sprintf("error in marshalling keys: %s", err.Error()),
			})
			_, _ = w.Write([]byte(errMessage))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(marshalledKeys))
	}
}
