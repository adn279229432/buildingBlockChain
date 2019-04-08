package main

func main() {
	bc := NewBlockchain("1NpxpZkBYd3uYJGMcpzFs6q65WPrr1cDaM")
	cli := CLI{bc}
	cli.Run()
	//wallet:=NewWallet()
	//
	//fmt.Printf("私钥：%x\n", wallet.PrivateKey.D.Bytes())
	//
	//
	////打印公钥， 曲线上的x点和y点
	//fmt.Printf("公钥：%x\n", wallet.PublicKey)
	//
	//fmt.Printf("地址：%x\n", wallet.GetAddress())
	//
	//a,_:=hex.DecodeString("31386e42463850626d3447797276703438455736435644473735454b7a6663716863")
	//
	//ValidateAddress(a)
	//
	//fmt.Println(ValidateAddress(a))
}
