package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

const dbFile = "blockchain.db"
const blockBucket = "blocks"
const genesisData = "ruok"

type Blockchain struct {
	tip []byte //最近的一个区块的hash值
	db  *bolt.DB
}

type BlockChainIterateor struct {
	currenthash []byte
	db          *bolt.DB
}

func (bc *Blockchain) MineBlock(transations []*Transation) *Block {
	for _, tx := range transations {
		if bc.VerifyTransation(tx) != true {
			log.Panic("error:invalid transation")
		} else {
			fmt.Println("verify success")
		}
	}

	var lasthash []byte
	var lastheight int32
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		lasthash = b.Get([]byte("l"))
		blockdata := b.Get([]byte(lasthash))

		block := DeserializeBlock(blockdata)

		lastheight = block.Height
		return nil
	})

	checkErr(err)

	newBlock := NewBlock(transations, lasthash, lastheight+1)

	err=bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())

		checkErr(err)

		err = b.Put([]byte("l"), newBlock.Hash)

		checkErr(err)

		bc.tip = newBlock.Hash
		return nil
	})
	checkErr(err)

	return newBlock
}

//TODO 拆分成两个函数
// 新建区块链，并且将区块链持久化
func NewBlockchain(address string) *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)

	checkErr(err)

	err = db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blockBucket))

		if b == nil {

			fmt.Println("区块链不存在，创建一个新的区块链")
			transation := NewCoinbaseTX(address, genesisData)
			genesis := NewGensisBlock([]*Transation{transation})
			b, err := tx.CreateBucket([]byte(blockBucket))

			checkErr(err)

			err = b.Put(genesis.Hash, genesis.Serialize())

			checkErr(err)

			err = b.Put([]byte("l"), genesis.Hash)
			tip = genesis.Hash

		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	checkErr(err)

	bc := Blockchain{tip, db}

	set := UTXOSet{&bc}
	// 将所有UTXO找到并且持久化
	set.Reindex()

	return &bc
}

func (bc *Blockchain) iterator() *BlockChainIterateor {

	bci := &BlockChainIterateor{bc.tip, bc.db}

	return bci
}

func (i *BlockChainIterateor) Next() *Block {

	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		deblock := b.Get(i.currenthash)
		block = DeserializeBlock(deblock)
		return nil
	})

	checkErr(err)

	i.currenthash = block.PrevBlockHash
	return block
}
func (bc *Blockchain) printBlockchain() {
	bci := bc.iterator()

	for {
		block := bci.Next()
		block.String()
		fmt.Println()

		// 创世区块的prehash是0
		if len(block.PrevBlockHash) == 0 {
			break
		}

	}

}



func (bc *Blockchain) SignTransation(tx *Transation, prikey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transation)
	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransationById(vin.TXid)
		checkErr(err)
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	tx.Sign(prikey, prevTXs)
}

func (bc *Blockchain) FindTransationById(ID []byte) (Transation, error) {
	bci := bc.iterator()
	for {
		block := bci.Next()
		for _, tx := range block.Transations {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transation{}, errors.New("transation not found")
}

func (bc *Blockchain) VerifyTransation(tx *Transation) bool {
	prevTXs := make(map[string]Transation)	// 键-交易id   值- 交易

	// 遍历该笔交易的所有输入
	for _, vin := range tx.Vin {
		// 找到该笔输入引用的交易
		prevTX, err := bc.FindTransationById(vin.TXid)

		checkErr(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Verify(prevTXs)
}

//	返回所有utxo（未花费输出）
func (bc *Blockchain) FindAllUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)	// 键-txid 值-属于该tx的所有utxo

	spentTXs := make(map[string][]int)	// 键-txid 值tx中使用过的out下标

	bci := bc.iterator()

	// 遍历区块链中所有区块，从最近的区块往前遍历
	for {
		block := bci.Next()
		// 遍历该区块中的所有交易
		for _, tx := range block.Transations {

			txID := hex.EncodeToString(tx.ID)
		Outputs:
			// 遍历该交易中的所有输出
			for outIdx, out := range tx.Vout {
				// 如果记录该交易用过的out切片不为空
				if spentTXs[txID] != nil {
					// 如果我当前这笔输出的索引号存在于spentTXs[txID]当中，则直接遍历下一笔输出
					for _, spendOutIds := range spentTXs[txID] {
						if spendOutIds == outIdx {
							continue Outputs
						}
					}

				}
				out.index=outIdx	// 记录这笔输出的所在其交易当中的索引
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if tx.IsCoinBase() == false {
				for _, in := range tx.Vin {
					inTXID := hex.EncodeToString(in.TXid)

					spentTXs[inTXID] = append(spentTXs[inTXID], in.Voutindex)
				}
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return UTXO
}
// 获取区块链的最高高度
func (bc *Blockchain) GetBestHeight() int32 {
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		lastHash := b.Get([]byte("l"))
		blockdata := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockdata)
		return nil
	})
	checkErr(err)

	return lastBlock.Height
}

// 获取区块链中所有区块的区块哈希
func (bc *Blockchain) getblockhash() [][]byte {
	var blocks [][]byte

	bci := bc.iterator()

	for {
		block := bci.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return blocks
}

// 根据blockhash获取区块
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("Block is not Fund ")
		}

		block = *DeserializeBlock(blockData)
		return nil
	})

	checkErr(err)
	return block, nil
}

//	向区块链中添加区块
func (bc *Blockchain) AddBlock(block *Block) {
	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		blockIndb := b.Get(block.Hash)
		// 如果数据库当中已经存在该区块，怎不继续存储
		if blockIndb != nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		checkErr(err)
		lastHash := b.Get([]byte("l"))
		lastBlockdata := b.Get(lastHash)
		lastblock := DeserializeBlock(lastBlockdata)

		// 如果想要添加的区块高度比数据库中存储的最近一个区块的高度高，增进行更新
		if block.Height > lastblock.Height {
			err = b.Put([]byte("l"), block.Hash)
			checkErr(err)
			bc.tip = block.Hash
		}
		return nil
	})
	checkErr(err)
}

//	找出pubkeyhash的尽可能满足金额amount的utxo，返回值是utxo总金额、[交易id]utxo的索引的映射
func (bc *Blockchain) FindSpendableOutputs(pubkeyhash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)

	UTXOMap:=bc.FindAllUTXO()	// 键 - txid 值 - utxo的在其交易中的索引

	accumulated := 0

	for txid,outs:=range UTXOMap{
		for _,out:=range outs.Outputs{
			if out.CanBeUnlockedWith(pubkeyhash) && accumulated < amount{
				accumulated += out.Value
				unspentOutputs[txid] = append(unspentOutputs[txid], out.index)
				if accumulated >= amount {
					return accumulated, unspentOutputs
				}
			}
		}
	}

	return accumulated, unspentOutputs
}



//
////找出包含有未花费输出的transation，傻逼的函数
//func (bc *Blockchain) FindUnspentTransations(pubkeyhash []byte) []Transation {
//	var unspentTXs []Transation         //所有未被完全花费的交易
//	spendTXOs := make(map[string][]int) //string 交易的哈希值 []int 已经被花费的输出的序号
//
//	bci := bc.iterator()
//
//	for {
//		block := bci.Next()
//		for _, tx := range block.Transations {
//			txID := hex.EncodeToString(tx.ID)
//
//		output:
//			for outIdx, out := range tx.Vout {
//				if spendTXOs[txID] != nil {
//					for _, spentOut := range spendTXOs[txID] {
//						if spentOut == outIdx {
//							continue output
//						}
//					}
//				}
//				if out.CanBeUnlockedWith(pubkeyhash) {
//					unspentTXs = append(unspentTXs, *tx)
//				}
//			}
//
//			if tx.IsCoinBase() == false {
//				for _, in := range tx.Vin {
//					if in.canUnlockOutputWith(pubkeyhash) {
//						inTxId := hex.EncodeToString(in.TXid)
//						spendTXOs[inTxId] = append(spendTXOs[inTxId], in.Voutindex)
//					}
//				}
//			}
//
//		}
//		if len(block.PrevBlockHash) == 0 {
//			break
//		}
//	}
//	return unspentTXs
//}
//
////也是个傻逼的函数，找出了属于pubkeyhash的含有未花费输出的交易的所有output
//func (bc *Blockchain) FindUTXO(pubkeyhash []byte) []TXOutput {
//	var UTXOs []TXOutput
//	unspendTransation := bc.FindUnspentTransations(pubkeyhash)
//
//	for _, tx := range unspendTransation {
//		for _, out := range tx.Vout {
//			if out.CanBeUnlockedWith(pubkeyhash) {
//				UTXOs = append(UTXOs, out)
//			}
//		}
//	}
//	return UTXOs
//}



