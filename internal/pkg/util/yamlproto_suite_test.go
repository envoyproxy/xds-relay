package yamlproto_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestYamlProtoConverter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "YamlProtoConverter Suite")
}
