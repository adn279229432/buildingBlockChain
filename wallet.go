package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)

//用于存储私钥和公钥

type Wallet struct {
	PrivateKey ecdsa.PrivateKey

	PublicKey []byte
}

func NewWallet() *Wallet {
	privateKey, publicKey := newKeyPair()

	return &Wallet{privateKey, publicKey}

}

func (w *Wallet) GetAddress() []byte {
	pubkeyHash := HashPubKey(w.PublicKey)
	versionPayload := append([]byte{version}, pubkeyHash...)
	checksum := checkSum(versionPayload)
	fullPayLoad := append(versionPayload, checksum...)
	address := base58Encode(fullPayLoad)
	return address
}

func HashPubKey(pubkey []byte) []byte {
	pubkeyHash256 := sha256.Sum256(pubkey)
	PIPEMD160Hasher := ripemd160.New()

	_, err := PIPEMD160Hasher.Write(pubkeyHash256[:])

	checkErr(err)

	publicRIPEMD160 := PIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

func checkSum(payloac []byte) []byte {
	firstSHA := sha256.Sum256(payloac)
	secondSHA := sha256.Sum256(firstSHA[:])
	//checksum 是前面的4个字节
	checksum := secondSHA[:4]
	return checksum
}

func ValidateAddress(address []byte) bool {
	publicHash := base58Decode(address)
	actualCheckSum := publicHash[len(publicHash)-4:]

	pubHash := publicHash[1 : len(publicHash)-4]

	targetCheckSum := checkSum(append([]byte{0x00}, pubHash...))

	return bytes.Compare(targetCheckSum, actualCheckSum) == 0
}

//生成私钥和公钥，生成的私钥为结构体ecdsa.PrivateKey的指针
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	//生成椭圆曲线
	curve := elliptic.P256()
	//产生的是一个结构体指针，结构体类型为ecdsa.PrivateKey
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	checkErr(err)
	//x坐标与y坐标拼接在一起，生成公钥
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}
