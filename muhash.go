package muhash

import (
	"encoding/binary"
	"math/big"
	"math/bits"
	"unsafe"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
)

const (
	HashSize        = 32
	ElementBitSize  = 3072
	ElementByteSize = ElementBitSize / 8
	ElementWordSize = 3072 / bits.UintSize
	WordSizeInBytes = bits.UintSize / 8
)

var (
	// 2^3072 - 1103717, the largest 3072-bit safe prime number, is used as the modulus.
	prime *big.Int
)

func init() {
	prime = big.NewInt(1)
	prime.Lsh(prime, 3072)
	prime.Sub(prime, big.NewInt(1103717))
}

type Hash [HashSize]byte

// IsEqual returns true if target is the same as hash.
func (hash *Hash) IsEqual(target *Hash) bool {
	if hash == nil && target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}

type MuHash struct {
	numerator   *big.Int
	denominator *big.Int
}

func NewMuHash() MuHash {
	return MuHash{
		numerator:   big.NewInt(1),
		denominator: big.NewInt(1),
	}
}

func (mu MuHash) Reset() {
	mu.numerator.SetUint64(1)
	mu.denominator.SetUint64(1)
}

func (mu MuHash) Clone() MuHash {
	return MuHash{
		numerator:   new(big.Int).Set(mu.numerator),
		denominator: new(big.Int).Set(mu.denominator),
	}
}

func (mu MuHash) Add(data []byte) {
	element := dataToElement(data)
	mu.numerator.Mul(mu.numerator, element)
	mu.numerator.Mod(mu.numerator, prime)
}

func (mu MuHash) Remove(data []byte) {
	element := dataToElement(data)
	mu.denominator.Mul(mu.denominator, element)
	mu.denominator.Mod(mu.denominator, prime)
}

func (mu MuHash) Combine(other MuHash) {
	mu.numerator.Mul(mu.numerator, other.numerator)
	mu.denominator.Mul(mu.denominator, other.denominator)
	mu.numerator.Mod(mu.numerator, prime)
	mu.denominator.Mod(mu.denominator, prime)
}

func (mu MuHash) Finalize() *Hash {
	// numerator * 1/denominator mod prime
	mu.denominator.ModInverse(mu.denominator, prime)
	mu.numerator.Mul(mu.numerator, mu.denominator)
	mu.numerator.Mod(mu.numerator, prime)
	mu.denominator.SetUint64(1)
	var out [ElementByteSize]byte
	b := mu.numerator.Bits()
	for i := range b {
		binary.LittleEndian.PutUint64(out[i*WordSizeInBytes:], uint64(b[i]))
	}

	ret := Hash(blake2b.Sum256(out[:]))
	return &ret
}

func dataToElement(data []byte) *big.Int {
	var elementsWords [ElementWordSize]big.Word
	var zeros12 [12]byte
	hashed := blake2b.Sum256(data)
	stream, err := chacha20.NewUnauthenticatedCipher(hashed[:], zeros12[:])
	if err != nil {
		panic(err)
	}
	elementsBytes := (*[ElementByteSize]byte)(unsafe.Pointer(&elementsWords))
	stream.XORKeyStream(elementsBytes[:], elementsBytes[:])
	return new(big.Int).SetBits(elementsWords[:])
}
