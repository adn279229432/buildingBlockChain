package main

import (
	"encoding/hex"
	"github.com/boltdb/bolt"
	"log"
)

type UTXOSet struct {
	bchain *Blockchain
}

const utxoBucket = "chainset"

// 持久化所有的UTXO，键-txid 值-Outputs，每次调用这个函数，都会清空该数据库，重新储存最新的键值对
func (u UTXOSet) Reindex() {
	db := u.bchain.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		checkErr(err)
		return nil

	})
	checkErr(err)

	UTXO := u.bchain.FindAllUTXO()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			checkErr(err)
			err = b.Put(key, outs.SerializeTXOutputs())
			checkErr(err)
		}
		return nil
	})
	checkErr(err)
}

// 根据pubkeyhash查找属于其所有的utxo
func (u *UTXOSet) FindUTXObyPubkeyHash(pubkeyhash []byte) []TXOutput {
	var UTXOs []TXOutput

	db := u.bchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeTXOutputs(v)

			for _, out := range outs.Outputs {
				if out.CanBeUnlockedWith(pubkeyhash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	checkErr(err)
	return UTXOs
}

func (u UTXOSet) update(block *Block) {

	db := u.bchain.db
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		// 遍历该block的所有交易（Transations）
		for _, tx := range block.Transations {
			// 当 当前交易不是coinbase的时候
			if tx.IsCoinBase() == false {
				// 遍历当前交易的所有输入
				for _, vin := range tx.Vin {
					updateouts := TXOutputs{}
					outsbytes := b.Get(vin.TXid)	// 返回当前交易的输入引用的utxo所在的交易中所有utxo的编码值
					outs := DeserializeTXOutputs(outsbytes)	// 解码上一行的编码值

					for _, out := range outs.Outputs {
						outIdx:=out.index

						if outIdx != vin.Voutindex {

							updateouts.Outputs = append(updateouts.Outputs, out)
						}
					}
					if len(updateouts.Outputs) == 0 {
						err := b.Delete(vin.TXid)	// 当前交易不存在任何utxo
						checkErr(err)
					} else {
						err := b.Put(vin.TXid, updateouts.SerializeTXOutputs())
						checkErr(err)
					}
				}
			}
			newOutputs := TXOutputs{}

			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			err := b.Put(tx.ID, newOutputs.SerializeTXOutputs())
			checkErr(err)
		}
		return nil
	})
	checkErr(err)

}
