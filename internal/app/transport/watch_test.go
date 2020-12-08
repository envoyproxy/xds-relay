package transport

import (
	"sync"

	discoveryv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cachev2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	gcp "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	gcpv3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/envoyproxy/xds-relay/internal/pkg/stats"
)

type version int

const (
	V2 version = iota
	V3
)

var discoveryResponsev2 = &discoveryv2.DiscoveryResponse{}
var discoveryResponsev3 = &discoveryv3.DiscoveryResponse{}
var discoveryResponse2v2 = &discoveryv2.DiscoveryResponse{TypeUrl: resourcev2.ListenerType}
var discoveryResponse2v3 = &discoveryv3.DiscoveryResponse{TypeUrl: resourcev3.ListenerType}
var discoveryRequestv2 = &gcp.Request{}
var discoveryRequestv3 = &gcpv3.Request{}
var discoveryRequest2v2 = &gcp.Request{TypeUrl: resourcev2.ListenerType}
var discoveryRequest2v3 = &gcpv3.Request{TypeUrl: resourcev3.ListenerType}
var mockScope = stats.NewMockScope("mockwatch")

var _ = Describe("TestWatch", func() {
	DescribeTable("TestGetChannel", func(w Watch, v version) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			verifyChannelState(v, nil, false, w)
			wg.Done()
		}()
		w.Close()
		wg.Wait()
	}, []TableEntry{
		Entry("V2", newWatchV2(mockScope), V2),
		Entry("V3", newWatchV3(mockScope), V3),
	}...)

	DescribeTable("TestSendSuccessful", func(w Watch, r Response, expected interface{}, v version) {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			verifyChannelState(v, expected, true, w)
			wg.Done()
		}()

		go func() {
			err := w.Send(r)
			Expect(err).To(BeNil())
			wg.Done()
		}()
		wg.Wait()
	}, []TableEntry{
		Entry(
			"V2",
			newWatchV2(mockScope),
			NewResponseV2(discoveryRequestv2, discoveryResponsev2),
			&cachev2.PassthroughResponse{DiscoveryResponse: discoveryResponsev2, Request: discoveryRequestv2},
			V2),
		Entry(
			"V3",
			newWatchV3(mockScope),
			NewResponseV3(discoveryRequestv3, discoveryResponsev3),
			&cachev3.PassthroughResponse{DiscoveryResponse: discoveryResponsev3, Request: discoveryRequestv3},
			V3),
	}...)

	DescribeTable("TestSendFalseWhenBlocked", func(w Watch, resp Response) {
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
	}, []TableEntry{
		Entry("V2", newWatchV2(mockScope), NewResponseV2(discoveryRequestv2, discoveryResponsev2)),
		Entry("V3", newWatchV3(mockScope), NewResponseV3(discoveryRequestv3, discoveryResponsev3)),
	}...)

	DescribeTable("TestNoPanicOnSendAfterClose", func(w Watch, r Response, expected interface{}, v version) {
		err := w.Send(r)
		Expect(err).To(BeNil())
		w.Close()
		err = w.Send(r)
		Expect(err).To(BeNil())

		verifyChannelState(v, expected, true, w)
		verifyChannelState(v, nil, false, w)
	}, []TableEntry{
		Entry(
			"V2",
			newWatchV2(mockScope),
			NewResponseV2(discoveryRequestv2, discoveryResponsev2),
			&cachev2.PassthroughResponse{DiscoveryResponse: discoveryResponsev2, Request: discoveryRequestv2},
			V2),
		Entry(
			"V3",
			newWatchV3(mockScope),
			NewResponseV3(discoveryRequestv3, discoveryResponsev3),
			&cachev3.PassthroughResponse{DiscoveryResponse: discoveryResponsev3, Request: discoveryRequestv3},
			V3),
	}...)

	DescribeTable("TestSendDequeue", func(w Watch, r, r2 Response, expected interface{}, v version) {
		// Send two responses in sequence. The second one should be dequeued.
		var wg sync.WaitGroup
		wg.Add(2)
		err := w.Send(r)
		Expect(err).To(BeNil())
		wg.Done()
		err = w.Send(r2)
		Expect(err).To(BeNil())
		wg.Done()
		wg.Wait()

		verifyChannelState(v, expected, true, w)
	}, []TableEntry{
		Entry(
			"V2",
			newWatchV2(mockScope),
			NewResponseV2(discoveryRequestv2, discoveryResponsev2),
			NewResponseV2(discoveryRequest2v2, discoveryResponse2v2),
			&cachev2.PassthroughResponse{DiscoveryResponse: discoveryResponse2v2, Request: discoveryRequest2v2},
			V2),
		Entry(
			"V3",
			newWatchV3(mockScope),
			NewResponseV3(discoveryRequestv3, discoveryResponsev3),
			NewResponseV3(discoveryRequest2v3, discoveryResponse2v3),
			&cachev3.PassthroughResponse{DiscoveryResponse: discoveryResponse2v3, Request: discoveryRequest2v3},
			V3),
	}...)
})

func sendWithCloseChannelOnFailure(w Watch, wg *sync.WaitGroup, r Response) {
	err := w.Send(r)
	if err != nil {
		w.Close()
	}
	wg.Done()
}

func verifyChannelState(v version, expectedResponse interface{}, expectedMore bool, w Watch) {
	var got interface{}
	more := false

	switch v {
	case V2:
		got, more = <-w.GetChannel().V2
	case V3:
		got, more = <-w.GetChannel().V3
	}
	Expect(more).To(Equal(expectedMore))
	if expectedResponse == nil {
		Expect(got).To(BeNil())
	} else {
		Expect(got).To(Equal(expectedResponse))
	}
}
