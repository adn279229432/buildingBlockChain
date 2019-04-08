package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
)

// 我们的实例中，用端口号的不同来区分节点

type Version struct {
	Version    int    // 版本号
	BestHeight int32  // 区块最高的高度
	AddrFrom   string // 发送者地址
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type blocksend struct {
	AddrFrom string
	Block    []byte
}

const commandLength = 12

const nodeVersion = 0x00

var blockInTranit [][]byte

var knownNodes = []string{"localhost:3000"} // 存储已经探测到的网络

var nodeAddress string // 存储本区块运行的网络地址

func (ver *Version) String() {
	fmt.Println("Version:", ver.Version)
	fmt.Println("BestHeight:", ver.BestHeight)
	fmt.Println("AddrFrom:", ver.AddrFrom)
}

// 开启服务器，nodeID代表port，minerAddress 代表矿工地址
func startServer(nodeID, minerAddress string, bc *Blockchain) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID) // 构建当前节点地址
	ln, err := net.Listen("tcp", nodeAddress)
	checkErr(err)
	defer ln.Close()

	//bc := NewBlockchain("1NpxpZkBYd3uYJGMcpzFs6q65WPrr1cDaM")

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc) //向knownNodes[0](已经探测到的网络)发送自己的版本信息Version{}
	}

	for {
		conn, err := ln.Accept()

		checkErr(err)

		go handleConnection(conn, bc)

	}

}

// addr是目标地址
func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()

	payload := gobEncode(Version{nodeVersion, bestHeight, nodeAddress})

	request := append(commandToBytes("version"), payload...)

	sendData(addr, request)

}

func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)

	checkErr(err)

	// 获取指令
	command := bytesToCommand(request[:commandLength])

	switch command {
	case "version":
		fmt.Printf("\nstr:获取version\n")
		handleVersion(request, bc)

	case "getblocks":
		handleGetBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "block":
		handleBlock(request, bc)
	}
}

func handleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload Version
	buff.Write(request[commandLength:])

	dec := gob.NewDecoder(&buff)

	err := dec.Decode(&payload)
	checkErr(err)
	payload.String()
	myBestHeight := bc.GetBestHeight()      // 本区块的高度
	foreignBestHeight := payload.BestHeight // 外部节点传递的进来的区块高度

	if myBestHeight < foreignBestHeight {

		sendGetBlock(payload.AddrFrom) // 向外部节点发送获取区块请求

	} else {

		sendVersion(payload.AddrFrom, bc)

	}

	if !nodeIsKnow(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}

}

func handleGetBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])

	dec := gob.NewDecoder(&buff)

	err := dec.Decode(&payload)

	checkErr(err)

	block := bc.getblockhash()
	sendInv(payload.Addrfrom, "block", block)
}

func sendInv(addr string, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	sendData(addr, request)
}

func handleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer

	var payload inv

	buff.Write(request[commandLength:])

	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)

	checkErr(err)

	fmt.Printf("Recieve inventory %d , %s", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blockInTranit = payload.Items // 将外部节点传递进来的所有区块的哈希值存储
		blockHash := payload.Items[0] // 最近的区块哈希
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}

		// 将最近的区块从blockInTranit中删除
		for _, b := range blockInTranit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blockInTranit = newInTransit
	}
}

func handleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer

	var payload blocksend

	buff.Write(request[commandLength:])

	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)

	checkErr(err)

	blockdata := payload.Block

	block := DeserializeBlock(blockdata)
	bc.AddBlock(block)
	fmt.Println("Recieve a new Block")

	if len(blockInTranit) > 0 {
		blockHash := blockInTranit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)
		blockInTranit = blockInTranit[1:]
	} else {
		set := UTXOSet{bc}
		set.Reindex()
	}
}

func handleGetData(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[:commandLength])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)
	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		checkErr(err)
		sendBlock(payload.AddrFrom, &block)
	}

}

func sendBlock(addr string, block *Block) {
	data := blocksend{nodeAddress, block.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	sendData(addr, request)
}

func sendGetData(addr string, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress, kind, id})

	request := append(commandToBytes("getdata"), payload...)

	sendData(addr, request)
}

type getblocks struct {
	Addrfrom string
}

func sendGetBlock(addr string) {
	payload := gobEncode(getblocks{nodeAddress})

	request := append(commandToBytes("getblocks"), payload...)

	sendData(addr, request)
}

// 查看传入地址是否在knownNodes（已知节点）集合中
func nodeIsKnow(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}
	return false
}

// addr是目标地址
func sendData(addr string, data []byte) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)

		// 剔除不可用节点
		var updateNodes []string

		for _, node := range knownNodes {
			if node != addr {
				updateNodes = append(updateNodes, node)
			}
		}
		knownNodes = updateNodes
	}

	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))

	checkErr(err)

}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x00 {
			command = append(command, b)
		}
	}
	return fmt.Sprintf("%s", command)
}

func gobEncode(data interface{}) []byte {

	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)

	switch t := data.(type) {
	case Version:
		err := enc.Encode(&t)
		checkErr(err)
	case inv:
		err := enc.Encode(&t)
		checkErr(err)
	case blocksend:
		err := enc.Encode(&t)
		checkErr(err)
	case getdata:
		err := enc.Encode(&t)
		checkErr(err)
	case getblocks:
		err := enc.Encode(&t)
		checkErr(err)
		
	}

	return buff.Bytes()
}
