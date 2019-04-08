package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

//挖矿奖励
const subsidy = 100

//交易
type Transation struct {
	ID   []byte     // 交易的hash值（唯一标识符）
	Vin  []TXInput  // 交易中所有的输入
	Vout []TXOutput // 交易中所有的输出

}

//输入
type TXInput struct {
	TXid      []byte // 引用的output所在的交易的id
	Voutindex int    // 引用的output在其交易中的索引
	Signature []byte // 私钥签名，用于解锁utxo
	PubKey    []byte // 公钥
}

//输出
type TXOutput struct {

	Value      int    // 收益金额
	PubkeyHash []byte // 公钥hash（ripemd160(sha256(publickey))）
	index 		int		// 在交易当中的索引，目前初始化为-1

}

type TXOutputs struct {
	Outputs []TXOutput
}

// address是比特币地址
func (out *TXOutput) Lock(address []byte) {
	decodeAddress := base58Decode(address)
	publicKeyHash := decodeAddress[1 : len(decodeAddress)-4]
	out.PubkeyHash = publicKeyHash
}

//打印
func (tx Transation) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.TXid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Voutindex))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubkeyHash))
	}

	return strings.Join(lines, "\n")
}

//序列化
func (tx Transation) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)

	err := enc.Encode(tx)

	checkErr(err)

	return encoded.Bytes()
}

//计算交易的hash值
func (tx *Transation) Hash() []byte {

	txcopy := *tx
	txcopy.ID = []byte{}

	hash := sha256.Sum256(txcopy.Serialize())

	return hash[:]
}

//根据金额与地址新建一个输出
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil,-1}
	txo.Lock([]byte(address))
	return txo
}

//第一笔coinbase交易
func NewCoinbaseTX(to string, data string) *Transation {
	txin := TXInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(subsidy, to)

	tx := Transation{nil, []TXInput{txin}, []TXOutput{*txout}}

	tx.ID = tx.Hash()

	return &tx
}

func (out *TXOutput) CanBeUnlockedWith(pubkeyhash []byte) bool {
	return bytes.Compare(out.PubkeyHash, pubkeyhash) == 0
}

func (in *TXInput) canUnlockOutputWith(pubkeyhash []byte) bool {
	lockinghash := HashPubKey(in.PubKey)
	return bytes.Compare(lockinghash, pubkeyhash) == 0
}

func (tx Transation) IsCoinBase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TXid) == 0 && tx.Vin[0].Voutindex == -1
}

func NewUTXOTransation(from, to string, amount int, bc *Blockchain) *Transation {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := NewWallets()
	checkErr(err)

	wallet := wallets.GetWallet(from)
	pubkey := wallet.PublicKey
	acc, validoutputs := bc.FindSpendableOutputs(HashPubKey(pubkey), amount)

	if acc < amount {
		log.Panic("Error:Not enough funds")
	}
	for txid, outs := range validoutputs {
		txID, err := hex.DecodeString(txid)

		checkErr(err)

		for _, out := range outs {
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}
	outputs = append(outputs, *NewTXOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transation{nil, inputs, outputs}
	tx.ID = tx.Hash()
	bc.SignTransation(&tx, wallet.PrivateKey)
	return &tx
}

// 对交易签名，参数：私钥、该笔交易引用的其他交易
func (tx *Transation) Sign(privkey ecdsa.PrivateKey, prevTXs map[string]Transation) {
	if tx.IsCoinBase() {
		return
	}
	// 遍历当前交易的所有输入，检查一下键值对都满足条件
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.TXid)].ID == nil {
			log.Panic("error:~~")
		}
	}

	// 将当前交易复制一份
	txcopy := tx.TrimmedCopy()

	// 遍历当前交易的所有输入
	for inID, vin := range txcopy.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.TXid)]

		txcopy.Vin[inID].Signature = nil
		txcopy.Vin[inID].PubKey = prevTX.Vout[vin.Voutindex].PubkeyHash	// 将当前输入的公钥设置为其所引用的输出的公钥哈希

		txcopy.ID = txcopy.Hash()
		r, s, err := ecdsa.Sign(rand.Reader, &privkey, txcopy.ID)
		checkErr(err)
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature

	}
}

func (tx *Transation) Verify(prevTXs map[string]Transation) bool {

	if tx.IsCoinBase() {
		return true
	}

	// 遍历当前交易的所有输入，检查一下键值对都满足条件
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.TXid)].ID == nil {
			log.Panic("error")
		}
	}
	// 将当前交易复制一份
	txcopy := tx.TrimmedCopy()

	// 生成椭圆曲线
	curve := elliptic.P256()

	// 遍历当前交易的所有输入
	for inID, vin := range tx.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.TXid)]	//返回当前输入引用的交易
		txcopy.Vin[inID].Signature = nil	// 将当前输入的签名设置为nil

		txcopy.Vin[inID].PubKey = prevTX.Vout[vin.Voutindex].PubkeyHash	// 将当前输入的公钥设置为其所引用的输出的公钥哈希

		txcopy.ID = txcopy.Hash()

		r := big.Int{}
		s := big.Int{}

		siglen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(siglen / 2)])
		s.SetBytes(vin.Signature[(siglen / 2):])

		x := big.Int{}
		y := big.Int{}

		keylen := len(vin.PubKey)

		x.SetBytes(vin.PubKey[:(keylen / 2)])
		y.SetBytes(vin.PubKey[(keylen / 2):])

		rawPubkey := ecdsa.PublicKey{curve, &x, &y}

		if ecdsa.Verify(&rawPubkey, txcopy.ID, &r, &s) == false {
			return false
		}
		txcopy.Vin[inID].PubKey = nil
	}
	return true

}



func (tx *Transation) TrimmedCopy() Transation {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.TXid, vin.Voutindex, nil, nil})
	}
	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubkeyHash,-1})
	}
	txCopy := Transation{tx.ID, inputs, outputs}
	return txCopy
}



func (outs TXOutputs) SerializeTXOutputs() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)

	err := enc.Encode(outs)
	checkErr(err)
	return buff.Bytes()
}
func DeserializeTXOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))

	err := dec.Decode(&outputs)

	checkErr(err)

	return outputs

}
