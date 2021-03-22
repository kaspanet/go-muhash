// Package muhash provides an implementation of a Multiplicative Hash,
// a cryptographic data structure that allows you to have a rolling hash function
// that you can add and remove elements from, without the need to re-serialize and re-hash the whole data set.
package muhash

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
	"math/big"
)

const (
	// HashSize of array used to store hashes. See Hash.
	HashSize = 32
	// SerializedMuHashSize defines the length in bytes of SerializedMuHash
	SerializedMuHashSize = elementByteSize

	elementBitSize  = 3072
	elementByteSize = elementBitSize / 8

	primeDiff = 1103717
)

var (
	// 2^3072 - 1103717, the largest 3072-bit safe prime number, is used as the modulus.
	prime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 3072), big.NewInt(primeDiff))

	// EmptyMuHashHash is the hash of `NewMuHash().Finalize()`
	EmptyMuHashHash = Hash{0x32, 0x9d, 0x0a, 0x9d, 0x0c, 0xe1, 0x81, 0x7a, 0xa8, 0x82, 0xf8, 0x09, 0x35, 0xf2, 0x6e, 0x72, 0x4b, 0x0d, 0x6f, 0x7c, 0xe7, 0x9e, 0xeb, 0x3f, 0x5d, 0x20, 0x1a, 0x5a, 0xd9, 0x9e, 0x9b, 0x1c}

	errOverflow = errors.New("Overflow in the MuHash field")
)

// Hash is a type encapsulating the result of hashing some unknown sized data.
// it typically represents Blake2b.
type Hash [HashSize]byte

// IsEqual returns true if target is the same as hash.
func (hash Hash) IsEqual(target *Hash) bool {
	if target == nil {
		return false
	}
	return hash == *target
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
	numerator   num3072
	denominator num3072
}

// SerializedMuHash is a is a byte array representing the storage representation of a MuHash
type SerializedMuHash [SerializedMuHashSize]byte

// String returns the SerializedMultiSet as the hexadecimal string
func (serialized SerializedMuHash) String() string {
	return hex.EncodeToString(serialized[:])
}

// String returns the MultiSet as the hexadecimal string
func (mu MuHash) String() string {
	return mu.Serialize().String()
}

// NewMuHash return an empty initialized set.
// when finalized it should be equal to a finalized set with all elements removed.
func NewMuHash() *MuHash {
	return &MuHash{
		numerator:   oneNum3072(),
		denominator: oneNum3072(),
	}
}

// Reset clears the muhash from all data. Equivalent to creating a new empty set
func (mu *MuHash) Reset() {
	mu.numerator.SetToOne()
	mu.denominator.SetToOne()
}

// Clone the muhash to create a new one
func (mu MuHash) Clone() *MuHash {
	return &mu
}

// Add hashes the data and adds it to the muhash.
// Supports arbitrary length data (subject to the underlying hash function(Blake2b) limits)
func (mu *MuHash) Add(data []byte) {
	var element num3072
	dataToElement(data, &element)
	mu.addElement(&element)
}

func (mu *MuHash) addElement(element *num3072) {
	mu.numerator.Mul(element)
}

// Remove hashes the data and removes it from the multiset.
// Supports arbitrary length data (subject to the underlying hash function(Blake2b) limits)
func (mu *MuHash) Remove(data []byte) {
	var element num3072
	dataToElement(data, &element)
	mu.removeElement(&element)
}

func (mu *MuHash) removeElement(element *num3072) {
	mu.denominator.Mul(element)
}

// Combine will add the MuHash together. Equivalent to manually adding all the data elements
// from one set to the other.
func (mu *MuHash) Combine(other *MuHash) {
	mu.numerator.Mul(&other.numerator)
	mu.denominator.Mul(&other.denominator)
}

// Finalize will return a hash(Blake2b) of the multiset.
// Because the returned value is a hash of a multiset you cannot "Un-Finalize" it.
// If this is meant for storage then Serialize should be used instead.
func (mu *MuHash) normalize() {
	mu.numerator.Divide(&mu.denominator)
	mu.denominator.SetToOne()
}

// Serialize returns a serialized version of the MuHash. This is the only right way to serialize a multiset for storage.
// This MuHash is not finalized, this is meant for storage.
func (mu *MuHash) Serialize() *SerializedMuHash {
	var out SerializedMuHash
	mu.serializeInner(&out)
	return &out
}

func (mu *MuHash) serializeInner(out *SerializedMuHash) {
	mu.normalize()
	b := mu.numerator
	for i := range b.limbs {
		switch wordSize {
		case 64:
			binary.LittleEndian.PutUint64(out[i*wordSizeInBytes:], uint64(b.limbs[i]))
		case 32:
			binary.LittleEndian.PutUint32(out[i*wordSizeInBytes:], uint32(b.limbs[i]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
}

// DeserializeMuHash will deserialize the MuHash that `Serialize()` serialized.
func DeserializeMuHash(serialized *SerializedMuHash) (*MuHash, error) {
	numerator := num3072{}
	bytesToWordsLE((*[elementByteSize]byte)(serialized), &numerator.limbs)
	if numerator.IsOverflow() {
		return nil, errOverflow
	}

	return &MuHash{
		numerator:   numerator,
		denominator: oneNum3072(),
	}, nil
}

// Finalize will return a hash(blake2b) of the multiset.
// Because the returned value is a hash of a multiset you cannot "Un-Finalize" it.
// If this is meant for storage then Serialize should be used instead.
func (mu *MuHash) Finalize() Hash {
	var serialized SerializedMuHash
	mu.serializeInner(&serialized)
	return blake2b.Sum256(serialized[:])
}

func dataToElement(data []byte, out *num3072) {
	var zeros12 [12]byte
	hashed := blake2b.Sum256(data)
	stream, err := chacha20.NewUnauthenticatedCipher(hashed[:], zeros12[:])
	if err != nil {
		panic(err)
	}
	var elementsBytes [elementByteSize]byte
	stream.XORKeyStream(elementsBytes[:], elementsBytes[:])
	bytesToWordsLE(&elementsBytes, &out.limbs)
}

func bytesToWordsLE(elementsBytes *[elementByteSize]byte, elementsWords *[elementWordSize]word) {
	for i := range elementsWords {
		switch wordSize {
		case 64:
			elementsWords[i] = word(binary.LittleEndian.Uint64(elementsBytes[i*wordSizeInBytes:]))
		case 32:
			elementsWords[i] = word(binary.LittleEndian.Uint32(elementsBytes[i*wordSizeInBytes:]))
		default:
			panic("Only 32/64 bits machines are supported")
		}
	}
}
