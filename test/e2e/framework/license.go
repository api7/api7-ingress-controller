package framework

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"time"

	"github.com/api7/gorsa"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed manifests/license_root_ca.crt
	licenseRootCa []byte

	//go:embed manifests/license_root_private_key.key
	licenseRootPrivateKey []byte

	BeginLicense = "-----BEGIN LICENSE-----\n"
	EndLicense   = "\n-----END LICENSE-----"

	MaxEncryptBlockSize = 245

	FeatureGateway = "API7 Gateway"
	FeaturePortal  = "API7 Portal"

	SegmentEncryptSeparator = "\n"
)

var (
	tenyearsLicense = `-----BEGIN CERTIFICATE-----
MIICwjCCAaqgAwIBAgIEZl6CqjANBgkqhkiG9w0BAQsFADAeMRwwGgYDVQQDDBNB
UEk3IExpY2Vuc2UgU2VydmVyMB4XDTI0MDYwNDAyNTc0NloXDTM0MDUwNTAzMTQx
OVowFDESMBAGA1UEAxMJUHJpdmF0ZSBBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAzQOhXlXmqHFqoylKqatI0Lx+oCmF2D+1tvS4VhoSOcO9Fr48Bp6/
pLeBnmgRIAXEJryMSpljvonQJKBuUuCl83loH4Ak3payNaXInv9GAyGvzgx0Ktkb
m/8iThlnibFXGNFEjM2bRSmJa2arJB8DsZBU20n5B86ZHXYCFKzGxJig536wGyhR
FjIjD6CWOgA6d9+hybr+AhSXPLSr22isnO63TpPM2x84qePZ4u6TyiVQcvw9l5rS
9n7EKskETKBXMrLJnt2aizedBgfSxnY//XLktpCjeMjm7xH9UNQyBXiJWH+3BsXy
ThJw0mDtDpL5T1Akn0Ws4ERvIYjjWH8N3QIDAQABoxIwEDAOBgNVHQ8BAf8EBAMC
B4AwDQYJKoZIhvcNAQELBQADggEBABNg3/22QlL+z3NjsZ8qaeSpwaZSvDmn659b
AZ/JMyjym72MWc+hxSeNBKkdhuvW5Vfp3itudO4Se+UtmbdxHa2BjrjNc15kNI9E
hxPPYKs2euSRvrJltO3ZHrWcUyacdd3m26PeNIGmbQo6O2HYEpHCaqPgP4mPX24b
T1c4DJ20/vqRK7kxdRiHJuO1tgtErnkWxt3cZ0jNaNRjjtWF3toNDCzTwB8GO0y6
qZOXAx8ZONUxZLA+mPJ/+GmdtZLXot8lGccS7wS/H4lC14ClOC3BwclCWK5YisAU
/swPfbsDquG3zTFexciHBsOLefmRhRMNDuNSw5R85qnklgoKWCQ=
-----END CERTIFICATE-----
-----BEGIN LICENSE-----
ebsDHWXMfP8NYc8W8YFn0lcanxqgRhWhzzTGW7qU5kjT8xQALDVKGZB52L08Ey5qiZdQQu8ihJyA9oH5nJq_77dHq0xo9HiNfuE6g6uQ4IVOSXi8dZgTFzyyjHlwJXHL67O6c3M2bCeI6646i8eTDGPVTrMcbK-v2q0ZaZzxqNMZMu738hO3dkSwAEVp6fK298MYRviOIWdpfxPAdZu0d-csfhd0pSX7VclDUog7QRo66zCoPocMK-a3Zrp8ButyNcApmGWulG9egr1Nj3gZSNt9mU5yV-3xFtfbFl_YLJA6ll7Gk2UjamOrORoak5nLu7d0hcgCj9wxF3agN39LAQ
Gi96E_osNcD_liROu0NtvYZNrSGfDTT_yLwC7Jlsopjf5QLwPlE9yhw-FbOuHiAfCwGG0IQWLJWcROPW--8_HQKU8ujqOTsu2b5abqpx1MFyS9T35P2dqxicP9Li_XB-8dZ1jxm_Gii7uQkEUmhtCYB4EL5m9VKWwjbNmWCOfItYaHyT_87aiQHndH4wliyIAy2BpDwV-t9s7LbdHf2ZVaKFin6v0eRBqy-J7FdKhvgn-IWvm3PsUxcf2EJhXVjhBoiyIk_VCOmx1XkK9rxVqPRQOwCvpDkTtIM4vysb6nSwi5qGvYnZK1AbGqdJE0m7ydIyu3C3PT0jOrNczzvnEg
-----END LICENSE-----
`
	expiredLicense = `-----BEGIN CERTIFICATE-----
MIIC2TCCAcGgAwIBAgIEZl57hDANBgkqhkiG9w0BAQsFADAeMRwwGgYDVQQDDBNB
UEk3IExpY2Vuc2UgU2VydmVyMB4XDTI0MDYwNDAyMjcxNloXDTM0MDUwNTAzMTQx
OVowFDESMBAGA1UEAxMJUHJpdmF0ZSBBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAtLylHmfntWmrLYoTMPEgbAi3ZdDT+FNrrlIW1iAq8bxYfmJizkR8
4cgxgsfggvYbF/YOAqKZ8CYJs/eWKLJT6yR/udb+oT8Z7CM2tanl8Qax6zL/skYw
qwcaESaK7fp9T5lxHGwt45AIHBME8LPLYbieFdGY18Oz1bzYUJt7Ju9TKdAIlxye
PST3nzIpOTacBa93acQQIGBwARhx2K/juIHRlOTReEqJG/QEvcysRM5Rq0kzEIEU
N1w8yedLASIbIGm4CPW8/2Iw1Mn7cJfx1JJNDbnpK448R5bqIVH9oX+ukNIRuD30
uYTqmdv2QFNiIaTlDAYAZV0IV4wTIhi6OQIDAQABoykwJzAOBgNVHQ8BAf8EBAMC
B4AwFQYDVR0RBA4wDIIKZnJlZV90cmlhbDANBgkqhkiG9w0BAQsFAAOCAQEAmU7t
D+/0PVg33lhAU5fHqK5umwYVY4Zb9Akx55c0HGyRkKJhxyyCmU4tF9aemSfnPMzT
XvP0eHbyZr+j47IaFbUJPaDQALNedFMggte2sPTQ794dfmCwXIooBCUnOPz2X1hw
0spUtjsjATuNpiQTD/GJ7Z8RfwIfYRs0/u1/OlnLVehtAPB8cFshAqw06aSTE5y8
o9ZLwebgLhZUe4rkzDyKKnYb0cQripV7H3sQa3VGYOBr9xLTTGFQXz+kRszyE3B9
c+LaoRtPzQqjQo4uHDLTmxbsJDLEOfX5qiVKaTbQHNSM4lPrNpIqtUupF/enU8kk
JNy+O8imzazjMUuGZg==
-----END CERTIFICATE-----
-----BEGIN LICENSE-----
Ryzj-ccB_ppjuUs8svPzcbUjfrvGMft06x9EG4qmlI9800l5Uf4K4RuMtvaqmbtcaqmQ5OXUmpxB5OOA58FMqz0xDxc_LGPRJa0ImFKw3friQYjUKym6kAeD27EHAe__cxTEFkk0KI_RKzvX9PgmatP62z5n8pSxNN53lJ4hU4zJi-68cskzIrb4k9WwMzk4R0Kb7hp3jV9Qna_8pIouFM18IsnUplzBiJBRXe4sTph5GLELrm7crusy8XMQCy3g5mz2gB8gs8lj4VLICvOh7LPhjeKb232Vfc5MStVXjPpA5PzZ6q2oNAbo1xRLVi5Q33cnDv-qS7s6GnY8yut37A
IrE4crXKsoIdRA3ByDC33C2-1wThOAxykgqQogfdw6XBv0Mo6lvrbFcf8SASK4iodp_GhyEoUXBYssmJFxwTHEGxjgSJhHgbpPRml-9OvdkeKD5nfwclEWj9H90b20dNk5wcr7pi7fQP6KLEAyLv8yKsZp5A6LNwEZk-WxqG_e9LusZxWksVdZJbhEdc7ZMDjaDzrYGnap36CIHw0khH1j8H0wdU0fAdyemFEm3jj2Qvg9sT69yCpvR1nxkrBI2fGkVeryVqAIXrM30RJZz-LmMPo8V_BLzYRKqhpBPocjxto_kyP-iRjfnlFZcKhbaUzkx6WCRwmsVWXnWmC83BwA
-----END LICENSE-----
`
)

// License info for API7 Enterprise
type License struct {
	// LicenseSN The serial number of the license
	LicenseSN string `json:"license_sn"`
	// Customer The name of consumer
	Customer string `json:"customer"`
	// Secret Used for encryption and decryption of license-related data
	Secret string `json:"secret"`
	// CreatedAt The time the license was created
	EffectiveAt int64 `json:"effective_at"`
	// ExpiredAt The time of the license expired
	ExpiredAt int64 `json:"expired_at"`
	// MaxDPCores is the highest number of dp cores
	MaxDPCores         int64 `json:"max_dp_cores"`
	ETCDProxyWriteable bool  `json:"etcd_proxy_writeable"`
	// FeatureList is the feature list of license
	FeatureList []string `json:"feature_list"`
	// DeploymentId dashboard deployment id
	DeploymentID string `json:"deployment_id"`
	// FreeTrial is the flag that license is free trial or not
	FreeTrial bool `json:"free_trial"`
	// IsTestEnv is the flag that license is for test env or not
	IsTestEnv bool `json:"is_test_env"`
	// IssuanceDate is the license issuance date
	IssuanceDate int64 `json:"issuance_date,omitempty"`

	cert *x509.Certificate
}

// Validate validate license basic info
func (l *License) Validate() bool {
	return len(l.LicenseSN) > 0 &&
		len(l.Secret) > 0 &&
		len(l.Customer) > 0 &&
		l.EffectiveAt > 0 &&
		l.ExpiredAt > 0 &&
		l.EffectiveAt <= l.ExpiredAt
}

func (l *License) Cert() *x509.Certificate {
	return l.cert
}

func (f *Framework) UploadLicense() {
	payload := map[string]any{"data": tenyearsLicense}
	payloadBytes, err := json.Marshal(payload)
	assert.Nil(f.GinkgoT, err)

	respExpect := f.DashboardHTTPClient().PUT("/api/license").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payloadBytes).
		Expect()

	body := respExpect.Body().Raw()
	f.Logger.Logf(f.GinkgoT, "request /api/license, response body: %s", body)

	respExpect.Status(200)
}

func (f *Framework) GenerateDeprecatedFormatLicense(license License) (string, error) {
	if license.LicenseSN == "" {
		license.LicenseSN = uuid.New().String()
	}
	license.Secret = "secret"
	if len(license.FeatureList) == 0 {
		license.FeatureList = []string{FeatureGateway, FeaturePortal}
	}
	content, err := json.Marshal(license)
	if err != nil {
		return "", err
	}

	return segmentEncryptLicense(content, f.PrivateKey)
}

func (f *Framework) GenerateLicense(license License) (string, error) {
	return f.generateLicense(license, GetLicenseRootCaContent(), GetLicenseRootPrivateKey())
}

func (f *Framework) generateLicenseWithInvalidRootCa(license License) (string, error) {
	// generate a new root cert and private key instead of the correct root ca and private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	template := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "Root Cert"},
		SerialNumber: big.NewInt(time.Now().Unix()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 90),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	rootCert, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return "", err
	}
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCert,
	})
	privateKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return f.generateLicense(license, certPem, privateKeyPem)
}

func (f *Framework) generateLicenseWithRevokedL2Cert(license License) (string, error) {
	return f.generateLicense(license, GetLicenseRootCaContent(), GetLicenseRootPrivateKey(), certificateOptionsSetSN(big.NewInt(1257894000)))
}

func LoadCrtFromBytes(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}

	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse L2 certificate")
	}

	return crt, nil
}

func LoadPrivateKeyFromBytes(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func (f *Framework) generateLicense(license License, certPem, privatePem []byte, options ...certificateOption) (string, error) {
	parentPrivateKey, err := LoadPrivateKeyFromBytes(privatePem)
	if err != nil {
		return "", errors.Wrap(err, "failed to LoadPrivateKeyFromBytes")
	}
	parentCert, err := LoadCrtFromBytes(certPem)
	if err != nil {
		return "", errors.Wrap(err, "failed to LoadCrtFromBytes")
	}
	l2PrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	template := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "L2 Cert"},
		SerialNumber: big.NewInt(time.Now().Unix()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 90),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	if license.FreeTrial {
		template.DNSNames = append(template.DNSNames, "free_trial")
	}
	for _, opt := range options {
		opt(template)
	}
	l2Cert, err := x509.CreateCertificate(rand.Reader, template, parentCert, &l2PrivateKey.PublicKey, parentPrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to x509.CreateCertificate")
	}
	l2CertPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: l2Cert,
	})

	// adjust license
	if license.LicenseSN == "" {
		license.LicenseSN = uuid.New().String()
	}
	license.Secret = "secret"
	if len(license.FeatureList) == 0 {
		license.FeatureList = append(license.FeatureList, FeatureGateway, FeaturePortal)
	}
	license.IssuanceDate = time.Now().Unix()
	content, err := json.Marshal(license)
	if err != nil {
		return "", errors.Wrap(err, "failed to json.Unmarshal license")
	}
	encryptLicense, err := segmentEncryptLicense(content, l2PrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to segment encrypt license")
	}

	// encode license: cert + license
	var buf = bytes.NewBuffer(l2CertPem)
	for _, s := range []string{BeginLicense, encryptLicense, EndLicense} {
		if _, err := buf.WriteString(s); err != nil {
			return "", errors.Wrap(err, "failed to write string to buffer")
		}
	}
	return buf.String(), nil
}

func GetLicenseRootCaContent() []byte {
	return licenseRootCa
}

func GetLicenseRootPrivateKey() []byte {
	return licenseRootPrivateKey
}

func segmentEncryptLicense(content []byte, privateKey *rsa.PrivateKey) (string, error) {
	// If the content that needs to be encrypted does not exceed BlockSize, there is no need for multipart encryption.
	if len(content) <= MaxEncryptBlockSize {
		encryptedContent, err := gorsa.PrivateEncrypt(privateKey, content)
		if err != nil {
			return "", err
		}
		base64EncryptedContent := base64.RawURLEncoding.EncodeToString(encryptedContent)
		return base64EncryptedContent, nil
	}

	// If the content to be encrypted exceeds the BlockSize, press BlockSize to encrypt it in segments.
	var encryptedChunks []string
	for i := 0; i < len(content); i += MaxEncryptBlockSize {
		end := i + MaxEncryptBlockSize
		if end > len(content) {
			end = len(content)
		}
		encryptedChunk, err := gorsa.PrivateEncrypt(privateKey, content[i:end])
		if err != nil {
			return "", err
		}
		// We need base64 encode the content which after encrypting, else the content may include the separator.
		base64EncryptedChunk := base64.RawURLEncoding.EncodeToString(encryptedChunk)
		encryptedChunks = append(encryptedChunks, base64EncryptedChunk)
	}

	return strings.Join(encryptedChunks, SegmentEncryptSeparator), nil
}

type certificateOption func(certificate *x509.Certificate)

func certificateOptionsSetSN(sn *big.Int) certificateOption {
	return func(cert *x509.Certificate) {
		cert.SerialNumber = sn
	}
}
