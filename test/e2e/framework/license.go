package framework

import (
	_ "embed"
	"encoding/json"

	"github.com/stretchr/testify/assert"
)

func (f *Framework) UploadLicense() {
	payload := map[string]any{"data": API7EELicense}
	payloadBytes, err := json.Marshal(payload)
	assert.Nil(f.GinkgoT, err)

	respExpect := f.DashboardHTTPClient().PUT("/api/license").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payloadBytes).
		Expect()

	body := respExpect.Body().Raw()
	f.Logf("request /api/license, response body: %s", body)

	respExpect.Status(200)
}
