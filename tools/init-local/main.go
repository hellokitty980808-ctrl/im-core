// init-local 生成独立本地环境的随机凭据与 JWT 密钥，不覆盖已有 .env。
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

func randomSecret() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
func run() error {
	if _, err := os.Stat(".env"); err == nil {
		return fmt.Errorf(".env 已存在，请保留原有本地凭据")
	} else if !os.IsNotExist(err) {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: public,
	})
	secrets := make([]string, 3)
	for i := range secrets {
		secrets[i], err = randomSecret()
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintf(
		file,
		"COMPOSE_PROJECT_NAME=imcore\nDB_USER=root\nDB_PASSWORD=%s\nREDIS_PASSWORD=%s\nRABBITMQ_USER=imcore\nRABBITMQ_PASSWORD=%s\nJWT_PUBLIC_KEY=%s\nJWT_PRIVATE_KEY=%s\nGATEWAY_PORT=18000\nWEBSOCKET_PORT=16200\nCAPTCHA_DISABLED=false\n",
		secrets[0],
		secrets[1],
		secrets[2],
		base64.StdEncoding.EncodeToString(publicPEM),
		base64.StdEncoding.EncodeToString(privatePEM),
	)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("本地 .env 已创建；随机凭据与密钥不会写入仓库")
}
