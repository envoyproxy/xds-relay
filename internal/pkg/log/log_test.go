// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

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
