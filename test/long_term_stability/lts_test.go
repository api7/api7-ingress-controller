/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package long_term_stability

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/api7/api7-ingress-controller/test/e2e/framework"
	_ "github.com/api7/api7-ingress-controller/test/long_term_stability/spec_subjects"
)

// Run long-term-stability tests using Ginkgo runner.
func TestLongTermStability(t *testing.T) {
	RegisterFailHandler(Fail)
	var f = framework.NewFramework()

	BeforeSuite(f.BeforeSuite)
	AfterSuite(f.AfterSuite)

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting long-term-stability suite\n")
	RunSpecs(t, "long-term-stability suite")
}
