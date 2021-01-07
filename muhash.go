package muhash

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
	"math/big"
	"math/bits"
)

const (
	// HashSize of array used to store hashes. See Hash.
	HashSize = 32
	// SerializedMuHashSize defines the length in bytes of SerializedMuHash
	SerializedMuHashSize = elementByteSize

	elementBitSize  = 3072
	elementByteSize = elementBitSize / 8
	elementWordSize = 3072 / bits.UintSize
	wordSizeInBytes = bits.UintSize / 8
)

var (
	// 2^3072 - 1103717, the largest 3072-bit safe prime number, is used as the modulus.
	prime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 3072), big.NewInt(1103717))

	// EmptyMuHashHash is the hash of `NewMuHash().Finalize()`
	EmptyMuHashHash = Hash{0x32, 0x9d, 0x0a, 0x9d, 0x0c, 0xe1, 0x81, 0x7a, 0xa8, 0x82, 0xf8, 0x09, 0x35, 0xf2, 0x6e, 0x72, 0x4b, 0x0d, 0x6f, 0x7c, 0xe7, 0x9e, 0xeb, 0x3f, 0x5d, 0x20, 0x1a, 0x5a, 0xd9, 0x9e, 0x9b, 0x1c}
)

// Hash is a type encapsulating the result of hashing some unknown sized data.
// it typically represents Blake2b.
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

// SetBytes sets the bytes which represent the hash. An error is returned if
// the number of bytes passed in is not HashSize.
func (hash *Hash) SetBytes(newHash []byte) error {
	if len(newHash) != HashSize {
		return errors.Errorf("invalid hash length got %d, expected %d", len(newHash),
			HashSize)
	}
	copy(hash[:], newHash)
	return nil
}

// String returns the Hash as the hexadecimal string
func (hash Hash) String() string {
	return hex.EncodeToString(hash[:])
}

// MuHash is a type used to create a Multiplicative Hash
// which is a rolling(homomorphic) hash that you can add and remove elements from
// and receive the same resulting hash as-if you never hashed them.
// Because of that the order of adding and removing elements doesn't matter.
// Use NewMuHash to initialize a MuHash, or DeserializeMuHash to parse a MuHash.
type MuHash struct {
	numerator   *big.Int
	denominator *big.Int
}

// SerializedMuHash is a is a byte array representing the storage representation of a MuHash
type SerializedMuHash [SerializedMuHashSize]byte

// String returns the SerializedMultiSet as the hexadecimal string
func (serialized *SerializedMuHash) String() string {
	return hex.EncodeToString(serialized[:])
}

// String returns the MultiSet as the hexadecimal string
func (mu MuHash) String() string {
	return mu.Serialize().String()
}

// NewMuHash return an empty initialized set.
// when finalized it should be equal to a finalized set with all elements removed.
func NewMuHash() MuHash {
	return MuHash{
		numerator:   big.NewInt(1),
		denominator: big.NewInt(1),
	}
}

// Reset clears the muhash from all data. Equivalent to creating a new empty set
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

// Add hashes the data and adds it to the muhash.
// Supports arbitrary length data (subject to the underlying hash function(Blake2b) limits)
func (mu MuHash) Add(data []byte) {
	element := dataToElement(data)
	mu.numerator.Mul(mu.numerator, element)
	mu.numerator.Mod(mu.numerator, prime)
}

// Remove hashes the data and removes it from the multiset.
// Supports arbitrary length data (subject to the underlying hash function(Blake2b) limits)
func (mu MuHash) Remove(data []byte) {
	element := dataToElement(data)
	mu.denominator.Mul(mu.denominator, element)
	mu.denominator.Mod(mu.denominator, prime)
}

// Combine will add the MuHash together. Equivalent to manually adding all the data elements
// from one set to the other.
func (mu MuHash) Combine(other MuHash) {
	mu.numerator.Mul(mu.numerator, other.numerator)
	mu.denominator.Mul(mu.denominator, other.denominator)
	mu.numerator.Mod(mu.numerator, prime)
	mu.denominator.Mod(mu.denominator, prime)
}

// Finalize will return a hash(Blake2b) of the multiset.
// Because the returned value is a hash of a multiset you cannot "Un-Finalize" it.
// If this is meant for storage then Serialize should be used instead.
func (mu MuHash) normalize() {
	// numerator * 1/denominator mod prime
	mu.denominator.ModInverse(mu.denominator, prime)
	mu.numerator.Mul(mu.numerator, mu.denominator)
	mu.numerator.Mod(mu.numerator, prime)
	mu.denominator.SetUint64(1)
}

// Serialize returns a serialized version of the MuHash. This is the only right way to serialize a multiset for storage.
// This MuHash is not finalized, this is meant for storage.
func (mu MuHash) Serialize() *SerializedMuHash {
	mu.normalize()
	var out SerializedMuHash
	b := mu.numerator.Bits()
	for i := range b {
		switch bits.UintSize {
		case 64:
			binary.LittleEndian.PutUint64(out[i*wordSizeInBytes:], uint64(b[i]))
		case 32:
			binary.LittleEndian.PutUint32(out[i*wordSizeInBytes:], uint32(b[i]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
	return &out
}

// DeserializeMuHash will deserialize the MuHash that `Serialize()` serialized.
func DeserializeMuHash(serialized *SerializedMuHash) (*MuHash, error) {
	b := [384 / wordSizeInBytes]big.Word{}
	bytesToWordsLE((*[elementByteSize]byte)(serialized), &b)
	numerator := new(big.Int).SetBits(b[:])
	if numerator.Cmp(prime) >= 0 {
		return nil, errors.New("Overflow in the MuHash field")
	}

	return &MuHash{
		numerator:   numerator,
		denominator: new(big.Int).SetUint64(1),
	}, nil
}

func (mu MuHash) Finalize() *Hash {
	ret := Hash(blake2b.Sum256(mu.Serialize()[:]))
	return &ret
}

func dataToElement(data []byte) *big.Int {
	var zeros12 [12]byte
	hashed := blake2b.Sum256(data)
	stream, err := chacha20.NewUnauthenticatedCipher(hashed[:], zeros12[:])
	if err != nil {
		panic(err)
	}
	var elementsBytes [elementByteSize]byte
	stream.XORKeyStream(elementsBytes[:], elementsBytes[:])
	var elementsWords [elementWordSize]big.Word
	bytesToWordsLE(&elementsBytes, &elementsWords)
	return new(big.Int).SetBits(elementsWords[:])
}

func bytesToWordsLE(elementsBytes *[elementByteSize]byte, elementsWords *[elementWordSize]big.Word) {
	for i := range elementsWords {
		switch bits.UintSize {
		case 64:
			elementsWords[i] = big.Word(binary.LittleEndian.Uint64(elementsBytes[i*wordSizeInBytes:]))
		case 32:
			elementsWords[i] = big.Word(binary.LittleEndian.Uint32(elementsBytes[i*wordSizeInBytes:]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
}
