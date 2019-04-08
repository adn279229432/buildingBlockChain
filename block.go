package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"
)

//定义区块结构体
type Block struct {
	Version       int32  // 版本号
	PrevBlockHash []byte // 上一个区块的hash
	Merkleroot    []byte // merkle根
	Hash          []byte // 当前区块的hash
	Time          int32  // 当前区块时间戳
	Bits          int32  // 难度值
	Nonce         int32  //随机值
	Transations   []*Transation
	Height        int32 //区块高度
}

//序列化
func (b *Block) Serialize() []byte {

	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(b)

	checkErr(err)

	return encoded.Bytes()

}

//反序列化
func DeserializeBlock(d []byte) *Block {
	var block Block

	decode := gob.NewDecoder(bytes.NewReader(d))
	err := decode.Decode(&block)
	checkErr(err)
	return &block
}

//根据前一个hash增加区块
func NewBlock(transations []*Transation, prevBlockHash []byte, height int32) *Block {

	block := &Block{
		2,
		prevBlockHash,
		[]byte{}, // TODO 根据交易的哈希来生成

		[]byte{}, // 挖矿的时候产生
		int32(time.Now().Unix()),
		404454260,	// 实际比特币当中的难度会实时变化
		0,
		transations,
		height,
	}

	block.createMerkelTreeRoot(transations)

	pow := NewProofofWork(block)

	nonce, hash := pow.Run()

	block.Hash = hash
	block.Nonce = nonce

	return block
}

//创世区块
func NewGensisBlock(transations []*Transation) *Block {
	block := &Block{
		2,
		[]byte{},
		[]byte{},
		[]byte{},
		int32(time.Now().Unix()),
		404454260,
		0,
		transations,
		0,
	}

	block.createMerkelTreeRoot(transations)

	pow := NewProofofWork(block)

	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	//block.String()
	return block
}

//打印区块
func (b *Block) String() {
	fmt.Printf("version:%s\n", strconv.FormatInt(int64(b.Version), 10))
	fmt.Printf("Prev.BlockHash:%x\n", b.PrevBlockHash)
	fmt.Printf("Prev.merkleroot:%x\n", b.Merkleroot)
	fmt.Printf("cur.Hash:%x\n", b.Hash)
	fmt.Printf("Time:%s\n", strconv.FormatInt(int64(b.Time), 10))
	fmt.Printf("Bits:%s\n", strconv.FormatInt(int64(b.Bits), 10))
	fmt.Printf("nonce:%s\n", strconv.FormatInt(int64(b.Nonce), 10))
}

//func TestNewSerialize() {
//
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
//	deBlock := DeserializeBlock(block.Serialize())
//
//	deBlock.String()
//}

func (b *Block) createMerkelTreeRoot(transations []*Transation) {
	var tranHash [][]byte

	for _, tx := range transations {

		tranHash = append(tranHash, tx.Hash())
	}

	mTree := NewMerkleTree(tranHash)

	b.Merkleroot = mTree.RootNode.Data
}
