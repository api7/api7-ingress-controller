// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ingress

// import (
// 	"fmt"
// 	"net/http"
// 	"time"

// 	ginkgo "github.com/onsi/ginkgo/v2"
// 	"github.com/stretchr/testify/assert"

// 	"github.com/api7/api7-ingress-controller/test/e2e/scaffold"
// )

// var _ = ginkgo.Describe("suite-ingress-features: secret controller", func() {
// 	apisixTlsSuites := func(s *scaffold.Scaffold) {
// 		ginkgo.It("should create SSL if secret name referenced by TLS is corrected later", func() {
// 			secretName := "test-apisix-tls"
// 			// create secret later than ApisixTls
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			// key compare
// 			keyCompare := "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"
// 			// create secret
// 			err := s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)

// 			// create ApisixTls resource
// 			tlsName := "tls-name"
// 			host := "api6.com"
// 			//First attempt
// 			err = s.NewApisixTls(tlsName, host, "wrong-secret")
// 			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")

// 			// verify that no SSL resource is created so far
// 			tls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")

// 			//Second attempt
// 			err = s.NewApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")

// 			// verify SSL resource
// 			tls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")
// 			//cleanup
// 			s.DeleteApisixTls("sample-tls", "httpbin.org", "test-apisix-tls")
// 		})
// 		ginkgo.It("should update SSL if secret referenced by ApisixTls is created later", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - api6.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			secretName := "test-apisix-tls"
// 			// create ApisixTls resource
// 			tlsName := "tls-name"
// 			host := "api6.com"
// 			err := s.NewApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
// 			time.Sleep(10 * time.Second)

// 			// create secret later than ApisixTls
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			// key compare
// 			keyCompare := "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"
// 			// create secret
// 			err = s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)

// 			// verify SSL resource
// 			tls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()
// 		})

// 		ginkgo.It("should update SSL if secret referenced by ApisixTls is updated", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - api6.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			secretName := "test-apisix-tls"
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			// key compare
// 			keyCompare := "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"
// 			// create secret
// 			err := s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// create ApisixTls resource
// 			tlsName := "tls-name"
// 			host := "api6.com"
// 			err = s.NewApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			certUpdate := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJAM7zkxmhGdNEMA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzE1MzNaGA8yMDUxMDQwMjA3MTUzM1owZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZ
// NbA0INfLdBq3B5avL0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7E
// IVDKcCKF7JQUi5za9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQ
// PtG9e8pVYTbNhRublGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHh
// NwRyrWOb3zhdDeKlTm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTY
// AlXzwsbVYSOW4gMW39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY
// 5FN1KsTIA5fdDbgqCTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ
// 1kzVx6r70UMq+NJbs7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw1
// 7sUcH4DNHCxnzLK0L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1Gm
// cgaTC92nZsafA5d8r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4t
// A/8si8qUHh/hHKt71dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7
// Y7j4V6li2mkCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAGmueLf8FFxVCgKKPQbHgPJ8/bTBQfBXi4cG
// eg46QT3j4r8cw05CM64NXQs51q8kWa4aIhGzXweZPZZcnLppIZX31HguAsaOTWEv
// 2T3v9nqacMdujdyB+ll/bkwBoVudF47/JqQvHkevE0W43EZGgtmaPWki6psXK0C7
// Fwx95FeUizYrbsDDKbPKQ2iv70oES2bhhKi6K3o6uV6cFuUTxoNsjv9mjMM93xL/
// r5HhOAg7/voXwkvYAkOoehGRF4pwsfmjCJMF9k3BqDOZM+8087EHZFiWCPOvseA/
// FNxtEiDeUt86BGO9wzTv4ZN0gkJOrxATIw15wRemxnXJ8GUweiZh+U2nvDSklZq4
// Z4Wj2tWfa/yIqRBOZyqcAOS6jCnz/pYEGRniO6DMZSN1VX8A5pH8HNMnREW8Sze+
// c9cNZwquESqGFfKMAHOzuyxJhqEvuwwT/JNCLUgtQICPtdAQmNJQEwDAIdmS8VrX
// XNsBIYSloIopKd7IY3V7Y8yASs7jKLrtJN4AE+SpssWuTa4p2LSO08sW3NP8r86G
// 1cs5R6Mckmqaqk5ES9gRbKmhm4goamb2fe2HJ/PTFyOrWtucA6OU3AdrQNXY+qbK
// FskpOMlyl8WZo6JMsbOjd+tVygf9QuhvH9MxYDvppfeHxS0B7mYGZDOV/4G2vn2A
// C1K1xkvo
// -----END CERTIFICATE-----`
// 			keyUpdate := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKQIBAAKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZNbA0INfLdBq3B5av
// L0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7EIVDKcCKF7JQUi5za
// 9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQPtG9e8pVYTbNhRub
// lGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHhNwRyrWOb3zhdDeKl
// Tm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTYAlXzwsbVYSOW4gMW
// 39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY5FN1KsTIA5fdDbgq
// CTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ1kzVx6r70UMq+NJb
// s7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw17sUcH4DNHCxnzLK0
// L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1GmcgaTC92nZsafA5d8
// r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4tA/8si8qUHh/hHKt7
// 1dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7Y7j4V6li2mkCAwEA
// AQKCAgEAipLCQdUdHWfbS6JoUUtR6TnnbWDOaQV9Qt1q2PGBBN4oze6Z7zXyTbCK
// w04fiNy1cnoJidvn5UZfc/Pqk/I/KboV4TLWCBVqRIJH1QcOLDt7eaIvHZNAyjUY
// zmpj7ijnElsnQayw8gjDrzgS8g6FkMXQsFaoHRkXRsjx7THHTeelV0IsV8bTkPFz
// nDCKjveeRXACmBOHqFGAMBMh0rZqR9SUHn168HqGPZ/wPntY8/mtv8xkAIGp1tQ8
// lqn7Pe7CqA2d7IuPaUbJDCcL2FyUGfT+jfKQpAsGKdRNvlTlU7fOlzDKKzTKP/G5
// ZgrNv9NGUj7erwU9Nfb90r0RtqYIac2CQnBDNx6snsSLx+QlO5wEQ8+xZ3zjKkGi
// VazSaSqpkK2MV7KwyxJD5WEMqyHuUufQGU8TVywi4RkH21VXiwxwCbRLpF/gdBMi
// 2015JF1qd/t4GSHc6CtxqKslSxvNpr2cG6c/5Tx2gC3X6ARMr01YDFdKKI/FrsTQ
// w3/E/WNKoPNgAXy361oaT5QVGOyBbL8h9mmDW6vD3R0aLLzdZKsQU+DNs6tcAl10
// mboKMBDVyg2Qew5QsumD89EpA3Da9VkyMxcPmOJ7oOJqscNF3Ejyx8ePlF+jS3dR
// 467uaXERVCGep2WDuYcv5Y14uUYAVj+P9aH85YjozA2KzG0aiqECggEBAMZvZwwd
// fgub7RrU/0sHWw50JupVaRIjVIgopTmUw6MWDcpjn6BiCsMCJlabHxeN8XJYn3+E
// aro8TZMvMYvjQQ0yrdKbPw8VLoTNfu++Bn/HPQ8gHFV+wzk33xfKRlRqxXic+MlG
// SQ33IV+xr7JoW4fiMjSvOpp3kJj459mSOBLjhcXW7N4sMPhY6O72g2pqa8ncmOJT
// ZU94VlssAxL3B1P1wsH8HsjhIluDHT+9qwsHhq/prIGLs4ydQSNB0m1oDT21zK/R
// jC7fDbTT4oTZzy8QYiLDwW4ugct53HMQCnJfOdX4F6aOxt7k4d8VwoGuliKyjHii
// VX4C+LZfT/64XiUCggEBAMDLyfqY8PK4ULNPrpSh92IOX5iwCcwGmOYDwGIeIbO6
// S1Pp7WGhA6Ryap/+F5gIdhB0qfnhAslMrQNRFfo3EcHF86UiCsfIRbPh7fwC2s0F
// 4G+tST6TsPyCddjWiQsvsmT1eYPNDgPj6JDl3d0mzboVsVge46RORTAhm4k2pobZ
// EUDl+bljWM4LTf+pjN4DTRptwwueSqLVfxdkMhrHSG/NAYFerUKozxJGZeavnUcL
// c+WUCtPJvyQrx2CBhn8LQ7xsPJJPYjNwSf6joMNXPyGvfPYQaxaK1NM0y/7KfuYN
// N8DoBwwnpZIvcznjHDWY0P/cIZFKC2mmq6N062z/bfUCggEAey2oQAMGvVobgy55
// EzALvBsqFQjT4miADs18UxQfpVsJUHsrGboCiC8LcXN1h3+bQ6nzyIqAXf8VAKqp
// DPcS6IhvEm9AY7J4YAPYKiZBjow1QPBj5kZ8FUaze+caZUiqMEbwwLCapMqlsutv
// 70WMm/szwzSLIlvaLLtF4O89U6xc3ASgoQG5nFBEuCHaTfKl2nbPiJ7QItbGdG4L
// sngZ2mqSbSx+R6BJXZk0TN8GECCp4QUjCn+YA0+Sobo4T6Xpokb6OqHPbUEVFwz4
// bhNu4v4+jOoLZsQD2jVZPSvV8E1gb4xD0iaLGM3n0D2HskyX8g332OKcQ07A6SSd
// WbdE6QKCAQEAiC2pzgNHdfowrmcjBkNdLHrAlWYKlX03dIjD08o6vethl7UNAj+s
// BfT3UWk1myKm2jq9cQ25XRx2vHgC0Qki1r8OuN5RxQm2CjgUVERj7hsvi1JYAQZr
// JgC0YuQuSqN3G460NR+ava62r9pdmv70o3L9ICQ5YO4UOsoSRZo/h9I9OJz4hjUh
// HfCoOGS3Zn3ocTmEYml9iITKz2fraDTI+odQf+Oy9/mqwdrN0WLL8cmqJEgsWaoQ
// A+mUW5tBt+zp/GZrZmECGRlAesdzH2c55X5CAsBYE8UeTMznJmI7vh0p+20oxTIf
// 5iDz/7hmTYlSXtdLMoedhhO++qb0P7owHQKCAQBbjTZSMRiqevr3JlEouxioZ+XD
// yol9hMGQEquLWzqWlvP/LvgNT0PNRcE/qrMlPo7kzoq0r8PbL22tBl2C7C0e+BIv
// UnBVSIGJ/c0AhVSDuOAJiF36pvsDysTZXMTFE/9i5bkGOiwtzRNe4Hym/SEZUCpn
// 4hL3iPXCYEWz5hDeoucgdZQjHv2JeQQ3NR9OokwQOwvzlx8uYwgV/6ZT9dJJ3AZL
// 8Z1U5/a5LhrDtIKAsCCpRx99P++Eqt2M2YV7jZcTfEbEvxP4XBYcdh30nbq1uEhs
// 4zEnK1pMx5PnEljN1mcgmL2TPsMVN5DN9zXHW5eNQ6wfXR8rCfHwVIVcUuaB
// -----END RSA PRIVATE KEY-----`
// 			keyCompareUpdate := "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"
// 			// key update compare
// 			err = s.NewSecret(secretName, certUpdate, keyUpdate)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tlsUpdate, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// delete ApisixTls
// 			err = s.DeleteApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
// 		})

// 		ginkgo.It("should be able to handle a kube style SSL secret", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			apisixRoute := fmt.Sprintf(`
// apiVersion: apisix.apache.org/v2
// kind: ApisixRoute
// metadata:
//   name: httpbin-route
// spec:
//   http:
//   - name: rule1
//     match:
//       hosts:
//       - api6.com
//       paths:
//       - /ip
//     backends:
//     - serviceName: %s
//       servicePort: %d
// `, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

// 			secretName := "test-apisix-tls"
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			// key compare
// 			keyCompare := "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"
// 			// create kube secret
// 			err := s.NewKubeTlsSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// create ApisixTls resource
// 			tlsName := "tls-name"
// 			host := "api6.com"
// 			err = s.NewApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			certUpdate := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJAM7zkxmhGdNEMA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzE1MzNaGA8yMDUxMDQwMjA3MTUzM1owZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZ
// NbA0INfLdBq3B5avL0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7E
// IVDKcCKF7JQUi5za9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQ
// PtG9e8pVYTbNhRublGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHh
// NwRyrWOb3zhdDeKlTm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTY
// AlXzwsbVYSOW4gMW39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY
// 5FN1KsTIA5fdDbgqCTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ
// 1kzVx6r70UMq+NJbs7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw1
// 7sUcH4DNHCxnzLK0L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1Gm
// cgaTC92nZsafA5d8r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4t
// A/8si8qUHh/hHKt71dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7
// Y7j4V6li2mkCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAGmueLf8FFxVCgKKPQbHgPJ8/bTBQfBXi4cG
// eg46QT3j4r8cw05CM64NXQs51q8kWa4aIhGzXweZPZZcnLppIZX31HguAsaOTWEv
// 2T3v9nqacMdujdyB+ll/bkwBoVudF47/JqQvHkevE0W43EZGgtmaPWki6psXK0C7
// Fwx95FeUizYrbsDDKbPKQ2iv70oES2bhhKi6K3o6uV6cFuUTxoNsjv9mjMM93xL/
// r5HhOAg7/voXwkvYAkOoehGRF4pwsfmjCJMF9k3BqDOZM+8087EHZFiWCPOvseA/
// FNxtEiDeUt86BGO9wzTv4ZN0gkJOrxATIw15wRemxnXJ8GUweiZh+U2nvDSklZq4
// Z4Wj2tWfa/yIqRBOZyqcAOS6jCnz/pYEGRniO6DMZSN1VX8A5pH8HNMnREW8Sze+
// c9cNZwquESqGFfKMAHOzuyxJhqEvuwwT/JNCLUgtQICPtdAQmNJQEwDAIdmS8VrX
// XNsBIYSloIopKd7IY3V7Y8yASs7jKLrtJN4AE+SpssWuTa4p2LSO08sW3NP8r86G
// 1cs5R6Mckmqaqk5ES9gRbKmhm4goamb2fe2HJ/PTFyOrWtucA6OU3AdrQNXY+qbK
// FskpOMlyl8WZo6JMsbOjd+tVygf9QuhvH9MxYDvppfeHxS0B7mYGZDOV/4G2vn2A
// C1K1xkvo
// -----END CERTIFICATE-----`
// 			keyUpdate := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKQIBAAKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZNbA0INfLdBq3B5av
// L0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7EIVDKcCKF7JQUi5za
// 9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQPtG9e8pVYTbNhRub
// lGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHhNwRyrWOb3zhdDeKl
// Tm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTYAlXzwsbVYSOW4gMW
// 39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY5FN1KsTIA5fdDbgq
// CTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ1kzVx6r70UMq+NJb
// s7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw17sUcH4DNHCxnzLK0
// L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1GmcgaTC92nZsafA5d8
// r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4tA/8si8qUHh/hHKt7
// 1dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7Y7j4V6li2mkCAwEA
// AQKCAgEAipLCQdUdHWfbS6JoUUtR6TnnbWDOaQV9Qt1q2PGBBN4oze6Z7zXyTbCK
// w04fiNy1cnoJidvn5UZfc/Pqk/I/KboV4TLWCBVqRIJH1QcOLDt7eaIvHZNAyjUY
// zmpj7ijnElsnQayw8gjDrzgS8g6FkMXQsFaoHRkXRsjx7THHTeelV0IsV8bTkPFz
// nDCKjveeRXACmBOHqFGAMBMh0rZqR9SUHn168HqGPZ/wPntY8/mtv8xkAIGp1tQ8
// lqn7Pe7CqA2d7IuPaUbJDCcL2FyUGfT+jfKQpAsGKdRNvlTlU7fOlzDKKzTKP/G5
// ZgrNv9NGUj7erwU9Nfb90r0RtqYIac2CQnBDNx6snsSLx+QlO5wEQ8+xZ3zjKkGi
// VazSaSqpkK2MV7KwyxJD5WEMqyHuUufQGU8TVywi4RkH21VXiwxwCbRLpF/gdBMi
// 2015JF1qd/t4GSHc6CtxqKslSxvNpr2cG6c/5Tx2gC3X6ARMr01YDFdKKI/FrsTQ
// w3/E/WNKoPNgAXy361oaT5QVGOyBbL8h9mmDW6vD3R0aLLzdZKsQU+DNs6tcAl10
// mboKMBDVyg2Qew5QsumD89EpA3Da9VkyMxcPmOJ7oOJqscNF3Ejyx8ePlF+jS3dR
// 467uaXERVCGep2WDuYcv5Y14uUYAVj+P9aH85YjozA2KzG0aiqECggEBAMZvZwwd
// fgub7RrU/0sHWw50JupVaRIjVIgopTmUw6MWDcpjn6BiCsMCJlabHxeN8XJYn3+E
// aro8TZMvMYvjQQ0yrdKbPw8VLoTNfu++Bn/HPQ8gHFV+wzk33xfKRlRqxXic+MlG
// SQ33IV+xr7JoW4fiMjSvOpp3kJj459mSOBLjhcXW7N4sMPhY6O72g2pqa8ncmOJT
// ZU94VlssAxL3B1P1wsH8HsjhIluDHT+9qwsHhq/prIGLs4ydQSNB0m1oDT21zK/R
// jC7fDbTT4oTZzy8QYiLDwW4ugct53HMQCnJfOdX4F6aOxt7k4d8VwoGuliKyjHii
// VX4C+LZfT/64XiUCggEBAMDLyfqY8PK4ULNPrpSh92IOX5iwCcwGmOYDwGIeIbO6
// S1Pp7WGhA6Ryap/+F5gIdhB0qfnhAslMrQNRFfo3EcHF86UiCsfIRbPh7fwC2s0F
// 4G+tST6TsPyCddjWiQsvsmT1eYPNDgPj6JDl3d0mzboVsVge46RORTAhm4k2pobZ
// EUDl+bljWM4LTf+pjN4DTRptwwueSqLVfxdkMhrHSG/NAYFerUKozxJGZeavnUcL
// c+WUCtPJvyQrx2CBhn8LQ7xsPJJPYjNwSf6joMNXPyGvfPYQaxaK1NM0y/7KfuYN
// N8DoBwwnpZIvcznjHDWY0P/cIZFKC2mmq6N062z/bfUCggEAey2oQAMGvVobgy55
// EzALvBsqFQjT4miADs18UxQfpVsJUHsrGboCiC8LcXN1h3+bQ6nzyIqAXf8VAKqp
// DPcS6IhvEm9AY7J4YAPYKiZBjow1QPBj5kZ8FUaze+caZUiqMEbwwLCapMqlsutv
// 70WMm/szwzSLIlvaLLtF4O89U6xc3ASgoQG5nFBEuCHaTfKl2nbPiJ7QItbGdG4L
// sngZ2mqSbSx+R6BJXZk0TN8GECCp4QUjCn+YA0+Sobo4T6Xpokb6OqHPbUEVFwz4
// bhNu4v4+jOoLZsQD2jVZPSvV8E1gb4xD0iaLGM3n0D2HskyX8g332OKcQ07A6SSd
// WbdE6QKCAQEAiC2pzgNHdfowrmcjBkNdLHrAlWYKlX03dIjD08o6vethl7UNAj+s
// BfT3UWk1myKm2jq9cQ25XRx2vHgC0Qki1r8OuN5RxQm2CjgUVERj7hsvi1JYAQZr
// JgC0YuQuSqN3G460NR+ava62r9pdmv70o3L9ICQ5YO4UOsoSRZo/h9I9OJz4hjUh
// HfCoOGS3Zn3ocTmEYml9iITKz2fraDTI+odQf+Oy9/mqwdrN0WLL8cmqJEgsWaoQ
// A+mUW5tBt+zp/GZrZmECGRlAesdzH2c55X5CAsBYE8UeTMznJmI7vh0p+20oxTIf
// 5iDz/7hmTYlSXtdLMoedhhO++qb0P7owHQKCAQBbjTZSMRiqevr3JlEouxioZ+XD
// yol9hMGQEquLWzqWlvP/LvgNT0PNRcE/qrMlPo7kzoq0r8PbL22tBl2C7C0e+BIv
// UnBVSIGJ/c0AhVSDuOAJiF36pvsDysTZXMTFE/9i5bkGOiwtzRNe4Hym/SEZUCpn
// 4hL3iPXCYEWz5hDeoucgdZQjHv2JeQQ3NR9OokwQOwvzlx8uYwgV/6ZT9dJJ3AZL
// 8Z1U5/a5LhrDtIKAsCCpRx99P++Eqt2M2YV7jZcTfEbEvxP4XBYcdh30nbq1uEhs
// 4zEnK1pMx5PnEljN1mcgmL2TPsMVN5DN9zXHW5eNQ6wfXR8rCfHwVIVcUuaB
// -----END RSA PRIVATE KEY-----`
// 			keyCompareUpdate := "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"
// 			// key update compare
// 			err = s.NewKubeTlsSecret(secretName, certUpdate, keyUpdate)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tlsUpdate, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// delete ApisixTls
// 			err = s.DeleteApisixTls(tlsName, host, secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
// 		})
// 	}

// 	ingressSuites := func(s *scaffold.Scaffold) {
// 		ginkgo.It("should update SSL if secret referenced by Ingress is updated", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			secretName := "test-apisix-tls"

// 			ingress := fmt.Sprintf(`
// apiVersion: networking.k8s.io/v1
// kind: Ingress
// metadata:
//   annotations:
//     kubernetes.io/ingress.class: apisix
//   name: ingress-route
// spec:
//   tls:
//   - hosts:
//     - api6.com
//     secretName: %s
//   rules:
//   - host: api6.com
//     http:
//       paths:
//       - path: /ip
//         pathType: Exact
//         backend:
//           service:
//             name: %s
//             port:
//               number: %d
// `, secretName, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ingress))
// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 0, "routes number not expect")

// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			// key compare
// 			keyCompare := "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"
// 			// create secret
// 			err = s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			host := "api6.com"
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			routes, err = s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1, "routes number not expect")
// 			tls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			certUpdate := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJAM7zkxmhGdNEMA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzE1MzNaGA8yMDUxMDQwMjA3MTUzM1owZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZ
// NbA0INfLdBq3B5avL0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7E
// IVDKcCKF7JQUi5za9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQ
// PtG9e8pVYTbNhRublGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHh
// NwRyrWOb3zhdDeKlTm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTY
// AlXzwsbVYSOW4gMW39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY
// 5FN1KsTIA5fdDbgqCTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ
// 1kzVx6r70UMq+NJbs7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw1
// 7sUcH4DNHCxnzLK0L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1Gm
// cgaTC92nZsafA5d8r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4t
// A/8si8qUHh/hHKt71dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7
// Y7j4V6li2mkCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAGmueLf8FFxVCgKKPQbHgPJ8/bTBQfBXi4cG
// eg46QT3j4r8cw05CM64NXQs51q8kWa4aIhGzXweZPZZcnLppIZX31HguAsaOTWEv
// 2T3v9nqacMdujdyB+ll/bkwBoVudF47/JqQvHkevE0W43EZGgtmaPWki6psXK0C7
// Fwx95FeUizYrbsDDKbPKQ2iv70oES2bhhKi6K3o6uV6cFuUTxoNsjv9mjMM93xL/
// r5HhOAg7/voXwkvYAkOoehGRF4pwsfmjCJMF9k3BqDOZM+8087EHZFiWCPOvseA/
// FNxtEiDeUt86BGO9wzTv4ZN0gkJOrxATIw15wRemxnXJ8GUweiZh+U2nvDSklZq4
// Z4Wj2tWfa/yIqRBOZyqcAOS6jCnz/pYEGRniO6DMZSN1VX8A5pH8HNMnREW8Sze+
// c9cNZwquESqGFfKMAHOzuyxJhqEvuwwT/JNCLUgtQICPtdAQmNJQEwDAIdmS8VrX
// XNsBIYSloIopKd7IY3V7Y8yASs7jKLrtJN4AE+SpssWuTa4p2LSO08sW3NP8r86G
// 1cs5R6Mckmqaqk5ES9gRbKmhm4goamb2fe2HJ/PTFyOrWtucA6OU3AdrQNXY+qbK
// FskpOMlyl8WZo6JMsbOjd+tVygf9QuhvH9MxYDvppfeHxS0B7mYGZDOV/4G2vn2A
// C1K1xkvo
// -----END CERTIFICATE-----`
// 			keyUpdate := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKQIBAAKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZNbA0INfLdBq3B5av
// L0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7EIVDKcCKF7JQUi5za
// 9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQPtG9e8pVYTbNhRub
// lGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHhNwRyrWOb3zhdDeKl
// Tm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTYAlXzwsbVYSOW4gMW
// 39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY5FN1KsTIA5fdDbgq
// CTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ1kzVx6r70UMq+NJb
// s7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw17sUcH4DNHCxnzLK0
// L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1GmcgaTC92nZsafA5d8
// r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4tA/8si8qUHh/hHKt7
// 1dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7Y7j4V6li2mkCAwEA
// AQKCAgEAipLCQdUdHWfbS6JoUUtR6TnnbWDOaQV9Qt1q2PGBBN4oze6Z7zXyTbCK
// w04fiNy1cnoJidvn5UZfc/Pqk/I/KboV4TLWCBVqRIJH1QcOLDt7eaIvHZNAyjUY
// zmpj7ijnElsnQayw8gjDrzgS8g6FkMXQsFaoHRkXRsjx7THHTeelV0IsV8bTkPFz
// nDCKjveeRXACmBOHqFGAMBMh0rZqR9SUHn168HqGPZ/wPntY8/mtv8xkAIGp1tQ8
// lqn7Pe7CqA2d7IuPaUbJDCcL2FyUGfT+jfKQpAsGKdRNvlTlU7fOlzDKKzTKP/G5
// ZgrNv9NGUj7erwU9Nfb90r0RtqYIac2CQnBDNx6snsSLx+QlO5wEQ8+xZ3zjKkGi
// VazSaSqpkK2MV7KwyxJD5WEMqyHuUufQGU8TVywi4RkH21VXiwxwCbRLpF/gdBMi
// 2015JF1qd/t4GSHc6CtxqKslSxvNpr2cG6c/5Tx2gC3X6ARMr01YDFdKKI/FrsTQ
// w3/E/WNKoPNgAXy361oaT5QVGOyBbL8h9mmDW6vD3R0aLLzdZKsQU+DNs6tcAl10
// mboKMBDVyg2Qew5QsumD89EpA3Da9VkyMxcPmOJ7oOJqscNF3Ejyx8ePlF+jS3dR
// 467uaXERVCGep2WDuYcv5Y14uUYAVj+P9aH85YjozA2KzG0aiqECggEBAMZvZwwd
// fgub7RrU/0sHWw50JupVaRIjVIgopTmUw6MWDcpjn6BiCsMCJlabHxeN8XJYn3+E
// aro8TZMvMYvjQQ0yrdKbPw8VLoTNfu++Bn/HPQ8gHFV+wzk33xfKRlRqxXic+MlG
// SQ33IV+xr7JoW4fiMjSvOpp3kJj459mSOBLjhcXW7N4sMPhY6O72g2pqa8ncmOJT
// ZU94VlssAxL3B1P1wsH8HsjhIluDHT+9qwsHhq/prIGLs4ydQSNB0m1oDT21zK/R
// jC7fDbTT4oTZzy8QYiLDwW4ugct53HMQCnJfOdX4F6aOxt7k4d8VwoGuliKyjHii
// VX4C+LZfT/64XiUCggEBAMDLyfqY8PK4ULNPrpSh92IOX5iwCcwGmOYDwGIeIbO6
// S1Pp7WGhA6Ryap/+F5gIdhB0qfnhAslMrQNRFfo3EcHF86UiCsfIRbPh7fwC2s0F
// 4G+tST6TsPyCddjWiQsvsmT1eYPNDgPj6JDl3d0mzboVsVge46RORTAhm4k2pobZ
// EUDl+bljWM4LTf+pjN4DTRptwwueSqLVfxdkMhrHSG/NAYFerUKozxJGZeavnUcL
// c+WUCtPJvyQrx2CBhn8LQ7xsPJJPYjNwSf6joMNXPyGvfPYQaxaK1NM0y/7KfuYN
// N8DoBwwnpZIvcznjHDWY0P/cIZFKC2mmq6N062z/bfUCggEAey2oQAMGvVobgy55
// EzALvBsqFQjT4miADs18UxQfpVsJUHsrGboCiC8LcXN1h3+bQ6nzyIqAXf8VAKqp
// DPcS6IhvEm9AY7J4YAPYKiZBjow1QPBj5kZ8FUaze+caZUiqMEbwwLCapMqlsutv
// 70WMm/szwzSLIlvaLLtF4O89U6xc3ASgoQG5nFBEuCHaTfKl2nbPiJ7QItbGdG4L
// sngZ2mqSbSx+R6BJXZk0TN8GECCp4QUjCn+YA0+Sobo4T6Xpokb6OqHPbUEVFwz4
// bhNu4v4+jOoLZsQD2jVZPSvV8E1gb4xD0iaLGM3n0D2HskyX8g332OKcQ07A6SSd
// WbdE6QKCAQEAiC2pzgNHdfowrmcjBkNdLHrAlWYKlX03dIjD08o6vethl7UNAj+s
// BfT3UWk1myKm2jq9cQ25XRx2vHgC0Qki1r8OuN5RxQm2CjgUVERj7hsvi1JYAQZr
// JgC0YuQuSqN3G460NR+ava62r9pdmv70o3L9ICQ5YO4UOsoSRZo/h9I9OJz4hjUh
// HfCoOGS3Zn3ocTmEYml9iITKz2fraDTI+odQf+Oy9/mqwdrN0WLL8cmqJEgsWaoQ
// A+mUW5tBt+zp/GZrZmECGRlAesdzH2c55X5CAsBYE8UeTMznJmI7vh0p+20oxTIf
// 5iDz/7hmTYlSXtdLMoedhhO++qb0P7owHQKCAQBbjTZSMRiqevr3JlEouxioZ+XD
// yol9hMGQEquLWzqWlvP/LvgNT0PNRcE/qrMlPo7kzoq0r8PbL22tBl2C7C0e+BIv
// UnBVSIGJ/c0AhVSDuOAJiF36pvsDysTZXMTFE/9i5bkGOiwtzRNe4Hym/SEZUCpn
// 4hL3iPXCYEWz5hDeoucgdZQjHv2JeQQ3NR9OokwQOwvzlx8uYwgV/6ZT9dJJ3AZL
// 8Z1U5/a5LhrDtIKAsCCpRx99P++Eqt2M2YV7jZcTfEbEvxP4XBYcdh30nbq1uEhs
// 4zEnK1pMx5PnEljN1mcgmL2TPsMVN5DN9zXHW5eNQ6wfXR8rCfHwVIVcUuaB
// -----END RSA PRIVATE KEY-----`
// 			keyCompareUpdate := "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"
// 			// key update compare
// 			err = s.NewSecret(secretName, certUpdate, keyUpdate)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tlsUpdate, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()
// 		})

// 		ginkgo.It("should delete SSL if secret and Ingress are deleted", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			secretName := "test-apisix-tls"
// 			// create secret
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			err := s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			host := "api6.com"

// 			// create ingress
// 			ingress := fmt.Sprintf(`
// apiVersion: networking.k8s.io/v1
// kind: Ingress
// metadata:
//   annotations:
//     kubernetes.io/ingress.class: apisix
//   name: ingress-route
// spec:
//   tls:
//   - hosts:
//     - api6.com
//     secretName: %s
//   rules:
//   - host: api6.com
//     http:
//       paths:
//       - path: /ip
//         pathType: Exact
//         backend:
//           service:
//             name: %s
//             port:
//               number: %d
// `, secretName, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ingress))

// 			// key compare
// 			keyCompare := "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"

// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1, "routes number not expect")
// 			ssls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), ssls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, ssls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, ssls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// delete secret
// 			err = s.DeleteResource("secret", secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "delete secret error")
// 			// check routes
// 			routes, err = s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1, "routes number not expect")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			ssls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), ssls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, ssls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, ssls[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// delete ingress
// 			err = s.DeleteResource("ingress", "ingress-route")
// 			assert.Nil(ginkgo.GinkgoT(), err, "delete ingress error")
// 			// check routes
// 			routes, err = s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 0, "routes number not expect")
// 			// check ssl in APISIX
// 			time.Sleep(20 * time.Second)
// 			ssls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), ssls, 0, "tls number not expect")
// 		})

// 		ginkgo.It("should update SSL if secret is deleted then re-create", func() {
// 			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
// 			secretName := "test-apisix-tls"
// 			// create secret
// 			cert := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJALDqPppBVXQ3MA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzEyMDBaGA8yMDUxMDQwMjA3MTIwMFowZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwY
// Y6sVLGtWoR8gKFSZImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV
// 0npk/TpZfaCx7zobsfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG
// 3Fhr0AC067GVYvdwp1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFl
// itFFPZkeYG89O/7Ca1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaW
// v+xauWnm4hxOzBK7ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415h
// M2jMK69aAkQL71xa+66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTl
// X4csA+aMHF3v/U7hL/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN
// 7fRMZKDIHLacSPE0GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXF
// w2GqfAFEQbD4wazCh1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVe
// v0Yg/OxbbymeTh/hNCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrO
// eFuhSMLVblUCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAFgeSuMPpriYUrQByO3taiY1+56s+KoAmsyA
// LH15n2+thAgorusX2g1Zd7UNBHBtVO8c+ihpQb+2PnmgTTGT70ndpRbV5F6124Mt
// Hui/X0kjm76RYd1QKN1VFp0Zo+JVdRa+VhDsXWjO0VetINmFhNINFEJctyeHB8oi
// aaDL0wZrevHh47hBqtnrmLl+QVG34aLBRhZ5953leiNvXHUJNaT0nLgf0j9p4esS
// b2bx9uP4pFI1T9wcv/TE3K0rQbu/uqGY6MgznXHyi4qIK/I+WCa3fF2UZ5P/5EUM
// k2ptQneYkLLUVwwmj8C04bYhYe7Z6jkYYp17ojxIP+ejOY1eiE8SYKNjKinmphrM
// 5aceqIyfbE4TPqvicNzIggA4yEMPbTA0PC/wqbCf61oMc15hwacdeIaQGiwsM+pf
// DTk+GBxp3megx/+0XwTQbguleTlHnaaES+ma0vbl6a1rUK0YAUDUrcfFLv6EFtGD
// 6EHxFf7gH9sTfc2RiGhKxUhRbyEree+6INAvXy+AymVYmQmKuAFqkDQJ+09bTfm8
// bDs+00FijzHFBvC8cIhNffj0qqiv35g+9FTwnE9qpunlrtKG/sMgEXX6m8kL1YQ8
// m5DPGhyEZyt5Js2kzzo8TyINPKmrqriYuiD4p4EH13eSRs3ayanQh6ckC7lb+WXq
// 3IrSc5hO
// -----END CERTIFICATE-----`
// 			key := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKAIBAAKCAgEAuEPPnUMlSw41CTdxUNxkQ4gAZ7cPotwYY6sVLGtWoR8gKFSZ
// ImQIor3UF+8HhN/ZOFRv5tSeeQ/MTE72cs2T5mp6eRU8OrSV0npk/TpZfaCx7zob
// sfXB4YU1NZcVWKthFF5X8p//l5gxUMU8V4a01P0aDTmZ67wG3Fhr0AC067GVYvdw
// p1yRt6TUUk8ha7JsiySchUIFhX5QMWmrSNhc1bDnHetejMFlitFFPZkeYG89O/7C
// a1K3ca/VVu+/IJ4h7fbF3rt4uP182cJdHl1L94dQSKCJukaWv+xauWnm4hxOzBK7
// ImpYB/2eP2D33tmuCLeSv4S+bTG1l7hIN9C/xrYPzfun415hM2jMK69aAkQL71xa
// +66kVxioJyNYogYz3ss5ruzDL8K/7bkdO0Zzqldd2+j8lkTlX4csA+aMHF3v/U7h
// L/4Wdwi8ziwToRMq9KK9vuh+mPgcdtFGFml+sU+NQfJNm/BN7fRMZKDIHLacSPE0
// GUkfW+x3dXOP2lWSZe/iOBZ0NOGNdrOnxTRTr7IH7DYU8aXFw2GqfAFEQbD4wazC
// h1AI8lkZr6mPGB1q+HnF2IF7kkgXBHtI5U2KErgcX5BirIVev0Yg/OxbbymeTh/h
// NCcY1kJ1YUCbm9U3U6ZV+d8lj7dQHtugcAzWxSTwpBLVUPrOeFuhSMLVblUCAwEA
// AQKCAgApTupoMvlVTiYNnuREYGQJz59noN5cgELndR8WCiotjLDE2dJKp2pYMX4u
// r2NcImKsAiHj+Z5dPXFrWfhd3EBf01cJdf0+m+VKfi3NpxsQ0smQ+9Hhn1qLmDVJ
// gklCy4jD7DKDLeM6tN+5X74bUROQ+/yvIk6jTk+rbhcdVks422LGAPq8SkBQjx8a
// JKs1XZZ/ywFbzmU2fA62RR4lAnwtW680QeO8Yk7FRAzltkHdFJMBtCcZsD13uxd0
// meKbCVhJ5JyPRi/WKN2oY65EdF3na+pPnc3CeLiq5e2gy2D7J6VyknBpUrXRdMXZ
// J3/p8ZrWUXEQhk26ZP50uNdXy/Bx1jYe+U8mpkTMYVYxgu5K4Zea3yJyRn2piiE/
// 9LnKNy/KsINt/0QE55ldvtciyP8RDA/08eQX0gvtKWWC/UFVRZCeL48bpqLmdAfE
// cMwlk1b0Lmo2PxULFLMAjaTKmcMAbwl53YRit0MtvaIOwiZBUVHE0blRiKC2DMKi
// SA6xLbaYJVDaMcfix8kZkKbC0xBNmL4123qX4RF6IUTPufyUTz/tpjzH6eKDw/88
// LmSx227d7/qWp5gROCDhZOPhtr4rj70JKNqcXUX9iFga+dhloOwfHYjdKndKOLUI
// Gp3K9YkPT/fCfesrguUx8BoleO5pC6RQJhRsemkRGlSY8U1ZsQKCAQEA5FepCn1f
// A46GsBSQ+/pbaHlbsR2syN3J5RmAFLFozYUrqyHE/cbNUlaYq397Ax7xwQkiN3F2
// z/whTxOh4Sk/HdDF4d+I0PZeoxINxgfzyYkx8Xpzn2loxsRO8fb3g+mOxZidjjXv
// vxqUBaj3Y01Ig0UXuw7YqCwys+xg3ELtvcGLBW/7NWMo8zqk2nUjhfcwp4e4AwBt
// Xcsc2aShUlr/RUrJH4adjha6Yaqc/8xTXHW8gZi5L2lucwB0LA+CBe4ES9BZLZdX
// N575/ohXRdjadHKYceYHiamt2326DzaxVJu2EIXU8dgdgOZ/6krITzuePRQHLPHX
// 6bDfdg/WSpFrtwKCAQEAzpVqBcJ1fAI7bOkt89j2zZb1r5uD2f9sp/lA/Dj65QKV
// ShWR7Y6Jr4ShXmFvIenWtjwsl86PflMgtsJefOmLyv8o7PL154XD8SnNbBlds6IM
// MyNKkOJFa5NOrsal7TitaTvtYdKq8Zpqtxe+2kg80wi+tPVQNQS/drOpR3rDiLIE
// La/ty8XDYZsSowlzBX+uoFq7GuMct1Uh2T0/I4Kf0ZLXwYjkRlRk4LrU0BRPhRMu
// MHugOTYFKXShE2a3OEcxqCgvQ/3pn2TV92pPVKBIBGL6uKUwmXQYKaV3G4u10pJ4
// axq/avBOErcKZOReb0SNiOjiIsth8o+hnpYPU5COUwKCAQBksVNV0NtpUhyK4Ube
// FxTgCUQp4pAjM8qoQIp+lY1FtAgBuy6HSneYa59/YQP56FdrbH+uO1bNeL2nhVzJ
// UcsHdt0MMeq/WyV4e6mfPjp/EQT5G6qJDY6quD6n7ORRQ1k2QYqY/6fteeb0aAJP
// w/DKElnYnz9jSbpCJWbBOrJkD0ki6LK6ZDPWrnGr9CPqG4tVFUBL8pBH4B2kzDhn
// fME86TGvuUkZM2SVVQtOsefAyhqKe7KN+cw+4mBYXa5UtxUl6Yap2CcZ2/0aBT2X
// C32qBC69a1a/mheUxuiZdODWEqRCvQGedFLuWLbntnqGlh+9h2tyomM4JkskYO96
// io4ZAoIBAFouLW9AOUseKlTb4dx+DRcoXC4BpGhIsWUOUQkJ0rSgEQ2bJu3d+Erv
// igYKYJocW0eIMys917Qck75UUS0UQpsmEfaGBUTBRw0C45LZ6+abydmVAVsH+6f/
// USzIuOw6frDeoTy/2zHG5+jva7gcKrkxKxcRs6bBYNdvjGkQtUT5+Qr8rsDyntz/
// 9f3IBTcUSuXjVaRiGkoJ1tHfg617u0qgYKEyofv1oWfdB0Oiaig8fEBb51CyPUSg
// jiRLBZaCtbGjgSacNB0JxsHP3buikG2hy7NJIVMLs/SSL9GNhpzapciTj5YeOua+
// ksICUxsdgO+QQg9QW3yoqLPy69Pd2dMCggEBANDLoUf3ZE7Dews6cfIa0WC33NCV
// FkyECaF2MNOp5Q9y/T35FyeA7UeDsTZ6Dy++XGW4uNStrSI6dCyQARqJw+i7gCst
// 2m5lIde01ptzDQ9iO1Dv1XasxX/589CyLq6CxLfRgPMJPDeUEg0X7+a0lBT5Hpwk
// gNnZmws4l3i7RlVMtACCenmof9VtOcMK/9Qr502WHEoGkQR1r6HZFb25841cehL2
// do+oXlr8db++r87a8QQUkizzc6wXD9JffBNo9AO9Ed4HVOukpEA0gqVGBu85N3xW
// jW4KB95bGOTa7r7DM1Up0MbAIwWoeLBGhOIXk7inurZGg+FNjZMA5Lzm6qo=
// -----END RSA PRIVATE KEY-----`
// 			err := s.NewSecret(secretName, cert, key)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			host := "api6.com"

// 			// create ingress
// 			ingress := fmt.Sprintf(`
// apiVersion: networking.k8s.io/v1
// kind: Ingress
// metadata:
//   annotations:
//     kubernetes.io/ingress.class: apisix
//   name: ingress-route
// spec:
//   tls:
//   - hosts:
//     - api6.com
//     secretName: %s
//   rules:
//   - host: api6.com
//     http:
//       paths:
//       - path: /ip
//         pathType: Exact
//         backend:
//           service:
//             name: %s
//             port:
//               number: %d
// `, secretName, backendSvc, backendSvcPort[0])
// 			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ingress))

// 			// key compare
// 			keyCompare := "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"

// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			routes, err := s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1, "routes number not expect")
// 			ssls, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
// 			assert.Len(ginkgo.GinkgoT(), ssls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, ssls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, ssls[0].Key, "tls key not expect")

// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// delete secret
// 			err = s.DeleteResource("secret", secretName)
// 			assert.Nil(ginkgo.GinkgoT(), err, "delete secret error")
// 			// check routes
// 			routes, err = s.ListApisixRoutes()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
// 			assert.Len(ginkgo.GinkgoT(), routes, 1, "routes number not expect")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			ssls, err = s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), ssls, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), cert, ssls[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompare, ssls[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

// 			// re-create secret
// 			certUpdate := `-----BEGIN CERTIFICATE-----
// MIIFcjCCA1qgAwIBAgIJAM7zkxmhGdNEMA0GCSqGSIb3DQEBCwUAMGUxCzAJBgNV
// BAYTAkNOMRAwDgYDVQQIDAdKaWFuZ3N1MQ8wDQYDVQQHDAZTdXpob3UxEDAOBgNV
// BAoMB2FwaTcuYWkxEDAOBgNVBAsMB2FwaTcuYWkxDzANBgNVBAMMBmp3LmNvbTAg
// Fw0yMTA0MDkwNzE1MzNaGA8yMDUxMDQwMjA3MTUzM1owZTELMAkGA1UEBhMCQ04x
// EDAOBgNVBAgMB0ppYW5nc3UxDzANBgNVBAcMBlN1emhvdTEQMA4GA1UECgwHYXBp
// Ny5haTEQMA4GA1UECwwHYXBpNy5haTEPMA0GA1UEAwwGancuY29tMIICIjANBgkq
// hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZ
// NbA0INfLdBq3B5avL0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7E
// IVDKcCKF7JQUi5za9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQ
// PtG9e8pVYTbNhRublGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHh
// NwRyrWOb3zhdDeKlTm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTY
// AlXzwsbVYSOW4gMW39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY
// 5FN1KsTIA5fdDbgqCTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ
// 1kzVx6r70UMq+NJbs7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw1
// 7sUcH4DNHCxnzLK0L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1Gm
// cgaTC92nZsafA5d8r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4t
// A/8si8qUHh/hHKt71dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7
// Y7j4V6li2mkCAwEAAaMjMCEwHwYDVR0RBBgwFoIIYXBpNi5jb22CCiouYXBpNi5j
// b20wDQYJKoZIhvcNAQELBQADggIBAGmueLf8FFxVCgKKPQbHgPJ8/bTBQfBXi4cG
// eg46QT3j4r8cw05CM64NXQs51q8kWa4aIhGzXweZPZZcnLppIZX31HguAsaOTWEv
// 2T3v9nqacMdujdyB+ll/bkwBoVudF47/JqQvHkevE0W43EZGgtmaPWki6psXK0C7
// Fwx95FeUizYrbsDDKbPKQ2iv70oES2bhhKi6K3o6uV6cFuUTxoNsjv9mjMM93xL/
// r5HhOAg7/voXwkvYAkOoehGRF4pwsfmjCJMF9k3BqDOZM+8087EHZFiWCPOvseA/
// FNxtEiDeUt86BGO9wzTv4ZN0gkJOrxATIw15wRemxnXJ8GUweiZh+U2nvDSklZq4
// Z4Wj2tWfa/yIqRBOZyqcAOS6jCnz/pYEGRniO6DMZSN1VX8A5pH8HNMnREW8Sze+
// c9cNZwquESqGFfKMAHOzuyxJhqEvuwwT/JNCLUgtQICPtdAQmNJQEwDAIdmS8VrX
// XNsBIYSloIopKd7IY3V7Y8yASs7jKLrtJN4AE+SpssWuTa4p2LSO08sW3NP8r86G
// 1cs5R6Mckmqaqk5ES9gRbKmhm4goamb2fe2HJ/PTFyOrWtucA6OU3AdrQNXY+qbK
// FskpOMlyl8WZo6JMsbOjd+tVygf9QuhvH9MxYDvppfeHxS0B7mYGZDOV/4G2vn2A
// C1K1xkvo
// -----END CERTIFICATE-----`
// 			keyUpdate := `-----BEGIN RSA PRIVATE KEY-----
// MIIJKQIBAAKCAgEAlXGEL4P5+mjllbb4VxkpqjsHny2wY7QZNbA0INfLdBq3B5av
// L0Gp/nHk1iOMq+RkUu+DDb9i0wKw0j1ygaaN4WNXCa7/LZ7EIVDKcCKF7JQUi5za
// 9A36nTBMLb7ZbFwxnxGkCK2AwdIbUIB6Ciq/d+WEbZelspPQPtG9e8pVYTbNhRub
// lGgVD+LEUHYdvSneveFQxdmHhNQMPXNbh+Qmgz48n7cLMEHhNwRyrWOb3zhdDeKl
// Tm/yfPAQwvQNet+Ol3g19tKhS7WzDndrPfPNYAsoOdXZ5TTYAlXzwsbVYSOW4gMW
// 39p68ZnoseQHBVZkiTfN2CGvWbzd9gUXOiQship+xRfSoVEY5FN1KsTIA5fdDbgq
// CTM4Ed1FTq39hJSBnsQObzQtfjgz0Sivwp9q0NoXX+YM6eBJ1kzVx6r70UMq+NJb
// s7mvAW9Ehzv2m2GsinMq3lyvVwSbsJWX9vOHKxmuWympmOw17sUcH4DNHCxnzLK0
// L0oDV2atpNyKFvmMWSSmw09wOvuE+G+CLwILvgiSiTRPe1GmcgaTC92nZsafA5d8
// r2uxxJRHD7PAQSE5YnsdLsXAFSegYyHss2jq8lzVFwWhTk4tA/8si8qUHh/hHKt7
// 1dYrSRmYTTyj834P8KDwQAMYBGcThtZa7PudOR22toohyte7Y7j4V6li2mkCAwEA
// AQKCAgEAipLCQdUdHWfbS6JoUUtR6TnnbWDOaQV9Qt1q2PGBBN4oze6Z7zXyTbCK
// w04fiNy1cnoJidvn5UZfc/Pqk/I/KboV4TLWCBVqRIJH1QcOLDt7eaIvHZNAyjUY
// zmpj7ijnElsnQayw8gjDrzgS8g6FkMXQsFaoHRkXRsjx7THHTeelV0IsV8bTkPFz
// nDCKjveeRXACmBOHqFGAMBMh0rZqR9SUHn168HqGPZ/wPntY8/mtv8xkAIGp1tQ8
// lqn7Pe7CqA2d7IuPaUbJDCcL2FyUGfT+jfKQpAsGKdRNvlTlU7fOlzDKKzTKP/G5
// ZgrNv9NGUj7erwU9Nfb90r0RtqYIac2CQnBDNx6snsSLx+QlO5wEQ8+xZ3zjKkGi
// VazSaSqpkK2MV7KwyxJD5WEMqyHuUufQGU8TVywi4RkH21VXiwxwCbRLpF/gdBMi
// 2015JF1qd/t4GSHc6CtxqKslSxvNpr2cG6c/5Tx2gC3X6ARMr01YDFdKKI/FrsTQ
// w3/E/WNKoPNgAXy361oaT5QVGOyBbL8h9mmDW6vD3R0aLLzdZKsQU+DNs6tcAl10
// mboKMBDVyg2Qew5QsumD89EpA3Da9VkyMxcPmOJ7oOJqscNF3Ejyx8ePlF+jS3dR
// 467uaXERVCGep2WDuYcv5Y14uUYAVj+P9aH85YjozA2KzG0aiqECggEBAMZvZwwd
// fgub7RrU/0sHWw50JupVaRIjVIgopTmUw6MWDcpjn6BiCsMCJlabHxeN8XJYn3+E
// aro8TZMvMYvjQQ0yrdKbPw8VLoTNfu++Bn/HPQ8gHFV+wzk33xfKRlRqxXic+MlG
// SQ33IV+xr7JoW4fiMjSvOpp3kJj459mSOBLjhcXW7N4sMPhY6O72g2pqa8ncmOJT
// ZU94VlssAxL3B1P1wsH8HsjhIluDHT+9qwsHhq/prIGLs4ydQSNB0m1oDT21zK/R
// jC7fDbTT4oTZzy8QYiLDwW4ugct53HMQCnJfOdX4F6aOxt7k4d8VwoGuliKyjHii
// VX4C+LZfT/64XiUCggEBAMDLyfqY8PK4ULNPrpSh92IOX5iwCcwGmOYDwGIeIbO6
// S1Pp7WGhA6Ryap/+F5gIdhB0qfnhAslMrQNRFfo3EcHF86UiCsfIRbPh7fwC2s0F
// 4G+tST6TsPyCddjWiQsvsmT1eYPNDgPj6JDl3d0mzboVsVge46RORTAhm4k2pobZ
// EUDl+bljWM4LTf+pjN4DTRptwwueSqLVfxdkMhrHSG/NAYFerUKozxJGZeavnUcL
// c+WUCtPJvyQrx2CBhn8LQ7xsPJJPYjNwSf6joMNXPyGvfPYQaxaK1NM0y/7KfuYN
// N8DoBwwnpZIvcznjHDWY0P/cIZFKC2mmq6N062z/bfUCggEAey2oQAMGvVobgy55
// EzALvBsqFQjT4miADs18UxQfpVsJUHsrGboCiC8LcXN1h3+bQ6nzyIqAXf8VAKqp
// DPcS6IhvEm9AY7J4YAPYKiZBjow1QPBj5kZ8FUaze+caZUiqMEbwwLCapMqlsutv
// 70WMm/szwzSLIlvaLLtF4O89U6xc3ASgoQG5nFBEuCHaTfKl2nbPiJ7QItbGdG4L
// sngZ2mqSbSx+R6BJXZk0TN8GECCp4QUjCn+YA0+Sobo4T6Xpokb6OqHPbUEVFwz4
// bhNu4v4+jOoLZsQD2jVZPSvV8E1gb4xD0iaLGM3n0D2HskyX8g332OKcQ07A6SSd
// WbdE6QKCAQEAiC2pzgNHdfowrmcjBkNdLHrAlWYKlX03dIjD08o6vethl7UNAj+s
// BfT3UWk1myKm2jq9cQ25XRx2vHgC0Qki1r8OuN5RxQm2CjgUVERj7hsvi1JYAQZr
// JgC0YuQuSqN3G460NR+ava62r9pdmv70o3L9ICQ5YO4UOsoSRZo/h9I9OJz4hjUh
// HfCoOGS3Zn3ocTmEYml9iITKz2fraDTI+odQf+Oy9/mqwdrN0WLL8cmqJEgsWaoQ
// A+mUW5tBt+zp/GZrZmECGRlAesdzH2c55X5CAsBYE8UeTMznJmI7vh0p+20oxTIf
// 5iDz/7hmTYlSXtdLMoedhhO++qb0P7owHQKCAQBbjTZSMRiqevr3JlEouxioZ+XD
// yol9hMGQEquLWzqWlvP/LvgNT0PNRcE/qrMlPo7kzoq0r8PbL22tBl2C7C0e+BIv
// UnBVSIGJ/c0AhVSDuOAJiF36pvsDysTZXMTFE/9i5bkGOiwtzRNe4Hym/SEZUCpn
// 4hL3iPXCYEWz5hDeoucgdZQjHv2JeQQ3NR9OokwQOwvzlx8uYwgV/6ZT9dJJ3AZL
// 8Z1U5/a5LhrDtIKAsCCpRx99P++Eqt2M2YV7jZcTfEbEvxP4XBYcdh30nbq1uEhs
// 4zEnK1pMx5PnEljN1mcgmL2TPsMVN5DN9zXHW5eNQ6wfXR8rCfHwVIVcUuaB
// -----END RSA PRIVATE KEY-----`
// 			keyCompareUpdate := "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"
// 			// key update compare
// 			err = s.NewSecret(secretName, certUpdate, keyUpdate)
// 			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
// 			// check ssl in APISIX
// 			time.Sleep(10 * time.Second)
// 			tlsUpdate, err := s.ListApisixSsl()
// 			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
// 			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
// 			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
// 			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
// 			// check DP
// 			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()
// 		})
// 	}

// 	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
// 		s := scaffold.NewDefaultV2Scaffold()
// 		apisixTlsSuites(s)
// 		ingressSuites(s)
// 	})
// })
