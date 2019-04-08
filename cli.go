package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct {
	bc *Blockchain
}

func (cli *CLI) addBlock() {
	cli.bc.MineBlock([]*Transation{})
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 1 {
		fmt.Println("参数小于1")
		os.Exit(1)
	}
	fmt.Println(os.Args)
}
func (cli *CLI) printChain() {
	cli.bc.printBlockchain()
}

func (cli *CLI) getBalance(address string) {

	balance := 0
	decodeAddress := base58Decode([]byte(address))
	pubkeyHash := decodeAddress[1 : len(decodeAddress)-4]

	set := UTXOSet{cli.bc}
	UTXOs := set.FindUTXObyPubkeyHash(pubkeyHash)
	//UTXOs := cli.bc.FindUTXO(pubkeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("balance of %s:%d\n", address, balance)
}

func (cli *CLI) send(from, to string, amount int) {
	tx := NewUTXOTransation(from, to, amount, cli.bc)

	newblock := cli.bc.MineBlock([]*Transation{tx})

	set := UTXOSet{cli.bc}

	set.update(newblock)

	//cli.getBalance("1NpxpZkBYd3uYJGMcpzFs6q65WPrr1cDaM")
	//cli.getBalance("1MVh4SCLbdnoXDT1pCmhepJ9ZMSdXTqsrB")
	fmt.Printf("Success")
}

func (cli *CLI) printUsage() {
	fmt.Println("USages:")
	fmt.Println("addblock-增加区块:")
	fmt.Println("printChain:打印区块链")
}
func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	fmt.Printf("your address:%s\n", address)
}
func (cli *CLI) listAddress() {
	wallets, err := NewWallets()
	checkErr(err)
	addresses := wallets.getAddress()

	for _, address := range addresses {
		fmt.Println("address", address)
	}

}
func (cli *CLI) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")

	if nodeID == "" {
		fmt.Println("NODE_ID is not set")
		os.Exit(1)
	}

	addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)

	printChainCmd := flag.NewFlagSet("printChain", flag.ExitOnError)

	getBalanceCMD := flag.NewFlagSet("getbalance", flag.ExitOnError)
	getBalanceAddress := getBalanceCMD.String("address", "", "the address to get balance of")

	startNodeCmd := flag.NewFlagSet("startNodeCmd", flag.ExitOnError)
	startNodeMinner := startNodeCmd.String("minner", "", "minnerAddress")

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendCmd.String("from", "", "source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	createWalletCMD := flag.NewFlagSet("createWallet", flag.ExitOnError)
	listAddressCMD := flag.NewFlagSet("listaddress", flag.ExitOnError)

	getBestHeightCMD := flag.NewFlagSet("getBestHeight", flag.ExitOnError)
	switch os.Args[1] {
	case "startNodeCmd":
		err := startNodeCmd.Parse(os.Args[2:])
		checkErr(err)
	case "getBestHeight":
		err := getBestHeightCMD.Parse(os.Args[2:])
		checkErr(err)
	case "createWallet":
		err := createWalletCMD.Parse(os.Args[2:])
		checkErr(err)
	case "listaddress":
		err := listAddressCMD.Parse(os.Args[2:])
		checkErr(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		checkErr(err)
	case "getbalance":
		err := getBalanceCMD.Parse(os.Args[2:])
		checkErr(err)
	case "addblock":
		err := addBlockCmd.Parse(os.Args[2:])
		checkErr(err)
	case "printChain":
		err := printChainCmd.Parse(os.Args[2:])
		checkErr(err)
	default:
		cli.printUsage()
		os.Exit(1)
	}
	if addBlockCmd.Parsed() {
		cli.addBlock()
	}
	if printChainCmd.Parsed() {
		cli.printChain()
	}
	if getBalanceCMD.Parsed() {

		if *getBalanceAddress == "" {
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
	if createWalletCMD.Parsed() {
		cli.createWallet()
	}
	if listAddressCMD.Parsed() {
		cli.listAddress()
	}
	if getBestHeightCMD.Parsed() {
		cli.getBestHeight()
	}
	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.stratNode(nodeID, *startNodeMinner)
	}
}
func (cli *CLI) getBestHeight() {

	fmt.Println(cli.bc.GetBestHeight())
}

func (cli *CLI) stratNode(nodeID string, minnerAddress string) {
	fmt.Printf("starting node%s", nodeID)

	if len(minnerAddress) > 0 {
		if ValidateAddress([]byte(minnerAddress)) {
			fmt.Printf(" minner is on %s", minnerAddress)
		} else {
			log.Panic("error minner address")
		}
	}
	startServer(nodeID, minnerAddress, cli.bc)
}
