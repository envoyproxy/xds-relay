package log

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mockWriter bool

func (w *mockWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

var _ = Describe("Zap options setup", func() {
	var opts *Options

	BeforeEach(func() {
		opts = &Options{}
	})

	It("should set a custom writer", func() {
		var w mockWriter
		WriteTo(&w)(opts)
		Expect(opts.OutputDest).To(Equal(&w))
	})
})

var _ = Describe("Initializing new Zap logger", func() {
	Context("with the default options", func() {
		It("shouldn't fail", func() {
			Expect(New()).NotTo(BeNil())
		})
	})

	Context("with custom options", func() {
		It("shouldn't fail", func() {
			cfg := zap.NewProductionEncoderConfig()
			encoder := zapcore.NewConsoleEncoder(cfg)
			Expect(New(WriteTo(ioutil.Discard), Encoder(encoder))).NotTo(BeNil())
		})
	})
})
