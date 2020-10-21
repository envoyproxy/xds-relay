package handler

import (
	"net/http"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/xds-relay/internal/app/orchestrator"
	"github.com/envoyproxy/xds-relay/internal/pkg/util/stringify"
	"github.com/envoyproxy/xds-relay/pkg/marshallable"
)

func edsDumpHandler(o *orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		edsKey := getParam(req.URL.Path)
		if edsKey == "" {
			w.WriteHeader(http.StatusBadRequest)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: "Empty key",
			})
			_, _ = w.Write([]byte(s))
		}

		c := orchestrator.Orchestrator.GetReadOnlyCache(*o)
		resp, err := c.FetchReadOnly(edsKey)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			s, _ := stringify.InterfaceToString(&marshallable.Error{
				Message: err.Error(),
			})
			_, _ = w.Write([]byte(s))
			return
		}

		versionedResponse := resp.Resp.Get()
		m := &marshallable.EDS{
			Key:       edsKey,
			Version:   resp.Resp.GetPayloadVersion(),
			Endpoints: make([]string, 0),
		}
		if versionedResponse.V2 != nil {
			r2 := marshalResources(versionedResponse.V2.Resources)
			for _, r := range r2.Endpoints {
				endpoint, _ := r.(*v2.ClusterLoadAssignment)
				if endpoint == nil {
					continue
				}
				for _, e := range endpoint.Endpoints {
					for _, lbe := range e.LbEndpoints {
						if lbe.GetEndpoint() == nil {
							continue
						}
						m.Endpoints = append(m.Endpoints, lbe.GetEndpoint().Address.GetSocketAddress().Address)
					}
				}
			}
		} else if versionedResponse.V3 != nil {
			r3 := marshalResources(versionedResponse.V3.Resources)
			for _, r := range r3.Endpoints {
				endpoint, _ := r.(*endpointv3.ClusterLoadAssignment)
				if endpoint == nil {
					continue
				}
				for _, e := range endpoint.Endpoints {
					for _, lbe := range e.LbEndpoints {
						if lbe.GetEndpoint() == nil {
							continue
						}
						m.Endpoints = append(m.Endpoints, lbe.GetEndpoint().Address.GetSocketAddress().Address)
					}
				}
			}
		}

		x, e := stringify.InterfaceToString(m)
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(x))
	}
}
