package zap

import (
	"fmt"
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
	var opts *options

	BeforeEach(func() {
		opts = &options{}
	})

	It("should set a custom writer", func() {
		var w mockWriter
		WriteTo(&w)(opts)
		Expect(opts.outputDest).To(Equal(&w))
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

var _ = Describe("Parse log level", func() {
	It("should set specified log level", func() {
		level, err := ParseLogLevel("error")
		Expect(err).To(BeNil())
		Expect(level).To(Equal(zap.NewAtomicLevelAt(zap.ErrorLevel)))
	})

	It("should set info level if invalid string is provided", func() {
		level, err := ParseLogLevel("not a valid log level")
		Expect(err).To(Equal(fmt.Errorf(`unrecognized level: "not a valid log level"`)))
		Expect(level).To(Equal(zap.NewAtomicLevelAt(zap.InfoLevel)))
	})
})
