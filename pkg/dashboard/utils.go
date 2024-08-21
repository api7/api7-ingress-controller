package dashboard

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"

	v1 "github.com/api7/api7-ingress-controller/api/dashboard/v1"
)

var (
	ErrUnknownApisixResourceType = errors.New("unknown apisix resource type")
)

type ResourceTypes interface {
	*v1.Route | *v1.Ssl | *v1.Service | *v1.StreamRoute | *v1.GlobalRule | *v1.Consumer | *v1.PluginConfig
}

func PKCS5Padding(plaintext []byte, blockSize int) []byte {
	padding := blockSize - len(plaintext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plaintext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func AesEencryptPrivatekey(data []byte, aeskey []byte) (string, error) {
	xcode, err := AesEncrypt(data, aeskey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(xcode), nil
}
