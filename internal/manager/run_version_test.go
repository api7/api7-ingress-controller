// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package manager

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/version"
)

func TestReleaseVersionAtLeast(t *testing.T) {
	minV := version.MustParseSemantic("1.31.0")

	for _, tc := range []struct {
		gitVersion  string
		wantAtLeast bool
	}{
		// Managed distributions tag the release; that is not a prerelease.
		{"v1.31.0-eks-a737599", true},
		{"v1.31.0-gke.1000000", true},
		{"v1.31.0+rke2r1", true},
		{"v1.30.5-eks-a737599", false},

		// Real prereleases of the minimum still sort below it.
		{"v1.31.0-alpha.1", false},
		{"v1.31.0-beta.0", false},
		{"v1.31.0-rc.1", false},

		{"v1.31.0", true},
		{"v1.32.1", true},
		{"v1.30.0", false},
	} {
		t.Run(tc.gitVersion, func(t *testing.T) {
			parsed, err := version.ParseSemantic(tc.gitVersion)
			if err != nil {
				t.Fatalf("parsing %q: %v", tc.gitVersion, err)
			}
			if got := releaseVersion(parsed).AtLeast(minV); got != tc.wantAtLeast {
				t.Errorf("releaseVersion(%q).AtLeast(%s) = %v, want %v",
					tc.gitVersion, minV, got, tc.wantAtLeast)
			}
		})
	}
}
