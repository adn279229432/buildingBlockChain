package main

import (
	"bytes"
	"crypto/sha256"
	"math/big"
)

type ProofOfWork struct {
	block   *Block
	tartget *big.Int
}

const targetBits = 16

func NewProofofWork(b *Block) *ProofOfWork {

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))	// 实际目标值不是这样求，而是根据难度值来推
	pow := &ProofOfWork{b, target}
	return pow
}

//func main() {
//	target := big.NewInt(1)
//	target.Lsh(target, uint(256-targetBits))
//	fmt.Printf("v-----%v\n",target.Bytes())
//	fmt.Printf("s-----%s\n",target.Bytes())
//	fmt.Printf("d-----%d\n",target.Bytes())
//	fmt.Printf("x-----%x\n",target.Bytes())
//}
func (pow *ProofOfWork) prepareData(nonce int32) []byte {

	data := bytes.Join(
		[][]byte{
			IntToHex(pow.block.Version),
			pow.block.PrevBlockHash,
			pow.block.Merkleroot,
			IntToHex(pow.block.Time),
			IntToHex(pow.block.Bits),
			IntToHex(nonce)},
		[]byte{},
	)
	return data
}

func (pow *ProofOfWork) Run() (int32, []byte) {

	var nonce int32
	nonce = 0

	var secondhash [32]byte

	var currenthash big.Int

	for nonce < maxnonce {

		//序列化
		data := pow.prepareData(nonce)
		//double hash
		fitstHash := sha256.Sum256(data)
		secondhash = sha256.Sum256(fitstHash[:])
		//	fmt.Printf("%x\n",secondhash)

		currenthash.SetBytes(secondhash[:])
		//比较
		if currenthash.Cmp(pow.tartget) == -1 {
			break
		} else {
			nonce++
		}
	}

	return nonce, secondhash[:]
}

// 验证POW
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)

	fitstHash := sha256.Sum256(data)
	secondhash := sha256.Sum256(fitstHash[:])
	hashInt.SetBytes(secondhash[:])
	isValid := hashInt.Cmp(pow.tartget) == -1

	return isValid
}

//func TestPow() {
//	//初始化区块
//	block := &Block{
//		2,
//		[]byte{},
//		[]byte{},
//		[]byte{},
//		1418755780,
//		404454260,
//		0,
//		[]*Transation{},
//	}
//
//	pow := NewProofofWork(block)
//
//	nonce, _ := pow.Run()
//
//	block.Nonce = nonce
//
//	fmt.Println("POW:", pow.Validate())
//
//}
