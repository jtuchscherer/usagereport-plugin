package models_test

import (
	. "github.com/jtuchscherer/usagereport-plugin/models"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Models", func() {
	var report Report

	BeforeEach(func() {
		report = Report{
			Orgs: []Org{
				Org{
					Name:        "test-org",
					MemoryQuota: 4096,
					MemoryUsage: 256,
					Spaces: []Space{Space{
						Name: "test-space",
						Apps: []App{
							App{Ram: 128, Instances: 2, Running: true},
							App{Ram: 128, Instances: 1, Running: false},
						},
						ServiceInstances: []ServiceInstance{
							ServiceInstance{Name: "my-mysql"},
							ServiceInstance{Name: "my-redis"},
							ServiceInstance{Name: "extra-mysql"},
						},
					},
					},
				},
			},
		}
	})

	Describe("Report#CSV", func() {
		It("should return csv formated string", func() {
			expectedOutput, err := ioutil.ReadFile("fixtures/result.csv")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(report.CSV()).To(Equal(string(expectedOutput)))
		})
	})

	Describe("Report#String", func() {
		It("should return human readable formated string", func() {
			expectedOutput, err := ioutil.ReadFile("fixtures/result.txt")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(report.String()).To(Equal(string(expectedOutput)))
		})
	})
})
