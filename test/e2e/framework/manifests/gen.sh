$ openssl genrsa -out client.key 2048

$ openssl req -new -key client.key -out client.csr

$ openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 3650 -sha256 -extfile v3.ext
