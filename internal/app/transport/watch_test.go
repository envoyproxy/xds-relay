package transport

import (
	"sync"

	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
)

type version int

const (
	V2 version = iota
	V3
)

var discoveryResponsev2 = &discoveryv2.DiscoveryResponse{}
var discoveryResponsev3 = &discoveryv3.DiscoveryResponse{}
var discoveryRequestv2 = &gcp.Request{}
var discoveryRequestv3 = &gcpv3.Request{}

var _ = Describe("TestWatch", func() {
	table.DescribeTable("TestGetChannel", func(w Watch, v version) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			more := true
			switch v {
			case V2:
				_, more = <-w.GetChannel().V2
			case V3:
				_, more = <-w.GetChannel().V3
			}
			gomega.Expect(more).To(gomega.BeFalse())
			wg.Done()
		}()
		w.Close()
		wg.Wait()
	}, []table.TableEntry{
		table.Entry("V2", newWatchV2(), V2),
		table.Entry("V3", newWatchV3(), V3),
	}...)

	table.DescribeTable("TestSendSuccessful", func(w Watch, r Response, expected interface{}, v version) {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			var got interface{}
			more := false
			switch v {
			case V2:
				got, more = <-w.GetChannel().V2
			case V3:
				got, more = <-w.GetChannel().V3
			}

			gomega.Expect(more).To(gomega.BeTrue())
			gomega.Expect(got).To(gomega.Equal(expected))
			wg.Done()
		}()

		go func() {
			ok := w.Send(r)
			gomega.Expect(ok).To(gomega.BeTrue())
			wg.Done()
		}()
		wg.Wait()
	}, []table.TableEntry{
		table.Entry(
			"V2",
			newWatchV2(),
			NewResponseV2(discoveryRequestv2, discoveryResponsev2),
			cachev2.PassthroughResponse{DiscoveryResponse: discoveryResponsev2, Request: *discoveryRequestv2},
			V2),
		table.Entry(
			"V3",
			newWatchV3(),
			NewResponseV3(discoveryRequestv3, discoveryResponsev3),
			cachev3.PassthroughResponse{DiscoveryResponse: discoveryResponsev3, Request: *discoveryRequestv3},
			V3),
	}...)

	table.DescribeTable("TestSendFalseWhenBlocked", func(w Watch, resp Response) {
		var wg sync.WaitGroup
		wg.Add(2)
		// We perform 2 sends with no receive on w.Out .
		// One of the send gets blocked because of no recipient.
		// The second send goes goes to default case due to channel full.
		// The second send closes the channel when blocked.
		// The closed channel terminates the blocked send to exit the test case.
		go sendWithCloseChannelOnFailure(w, &wg, resp)
		go sendWithCloseChannelOnFailure(w, &wg, resp)
		wg.Wait()
	}, []table.TableEntry{
		table.Entry("V2", newWatchV2(), NewResponseV2(discoveryRequestv2, discoveryResponsev2)),
		table.Entry("V3", newWatchV3(), NewResponseV3(discoveryRequestv3, discoveryResponsev3)),
	}...)
})

func sendWithCloseChannelOnFailure(w Watch, wg *sync.WaitGroup, r Response) {
	ok := w.Send(r)
	if !ok {
		w.Close()
	}
	wg.Done()
}
