package main

import (
	"code.cloudfoundry.org/cli/util/testhelpers/pluginbuilder"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUsagereportPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	pluginbuilder.BuildTestBinary(".", "usagereport")
	RunSpecs(t, "UsagereportPlugin Suite")
}
