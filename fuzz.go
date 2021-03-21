// +build gofuzz

package muhash

import "C"
import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

func Fuzz(data []byte) int {
	if len(data) < elementByteSize {
		replace := make([]byte, elementByteSize)
		copy(replace, data[:])
		data = replace
	}
	startNum := oneNum()
	startUint := oneUint3072()
	for start := 0; start+elementByteSize <= len(data); start += elementByteSize {
		current := data[start : start+elementByteSize]
		if (current[0] & 1) == 1 {
			startNum.Divide(getNum3072(current[:]))
			startUint.Divide(getUint3072(current[:]))
		} else {
			startNum.Mul(getNum3072(current[:]))
			startUint.Mul(getUint3072(current[:]))
		}
	}

	if !areEqual(&startNum, &startUint) {
		panic(fmt.Sprintf("Expected %v == %v", startNum, startUint))
	}
	return 1
}

func areEqual(num *num3072, uin *uint3072) bool {
	for i := range uin {
		if uin[i] != uint(num.limbs[i]) {
			return false
		}
	}
	return true
}

func oneUint3072() uint3072 {
	return uint3072{1}
}
func oneNum() num3072 {
	return num3072{limbs: [48]C.ulong{1}}
}

func getNum3072(data []byte) *num3072 {
	var num num3072
	for i := range num.limbs {
		switch bits.UintSize {
		case 64:
			num.limbs[i] = C.ulong(binary.LittleEndian.Uint64(data[i*wordSizeInBytes:]))
		case 32:
			num.limbs[i] = C.ulong(binary.LittleEndian.Uint32(data[i*wordSizeInBytes:]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
	return &num
}

func getUint3072(data []byte) *uint3072 {
	var num uint3072
	for i := range num {
		switch bits.UintSize {
		case 64:
			num[i] = uint(binary.LittleEndian.Uint64(data[i*wordSizeInBytes:]))
		case 32:
			num[i] = uint(binary.LittleEndian.Uint32(data[i*wordSizeInBytes:]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
	return &num
}
