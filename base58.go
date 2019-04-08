package main

import (
	"bytes"

	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func base58Encode(input []byte) []byte {
	result := make([]byte, 0)
	x := new(big.Int).SetBytes(input) //将input作为大整型存储
	base := big.NewInt(58)            //58的大整数。此行代码等同于base:=big.NewInt(int64(len(b58Alphabet)))
	mod := new(big.Int)               //存储余数
	zeroBigInt := big.NewInt(0)
	for x.Cmp(zeroBigInt) > 0 {
		x.DivMod(x, base, mod)                            //参数中x：被除数，base：除数，mod：余数；
		result = append(result, b58Alphabet[mod.Int64()]) //将base58编码直接存储到结果中
	}
	// 如果我们要转换的input前面的数据都是零，但我们计算的时候并没有把它放进去，所以我们现在要用等长度的1(b58Alphabet[0]==1)填在结果后面（后面会将元素全部反转，后面的会变成前面）
	for _, v := range input {
		if v == 0x00 {
			result = append(result, b58Alphabet[0])
		} else {
			break
		}
	}
	// 比如我们要存储的编码是abc（假设是这样的），但我们存储下来的却是cba，所以需要反转切片
	ReverseBytes(result)

	return result
}
func base58Decode(input []byte) []byte {
	zeroLength := 0
	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, v := range input {
		if v == '1' {
			zeroLength++
		} else {
			break
		}
	}
	input = input[zeroLength:]
	for _, v := range input {
		idx := bytes.IndexByte(b58Alphabet, v)                       //余数
		result.Mul(result, base).Add(result, big.NewInt(int64(idx))) // 比如十进制123就等于0*10+1=1；1*10+2=12；12*10+3=123
	}
	rtn := result.Bytes()
	rtn = append(bytes.Repeat([]byte{0x00}, zeroLength), rtn...)
	return rtn

}
