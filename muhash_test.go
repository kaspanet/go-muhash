package muhash

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

type testVector struct {
	dataElement    []byte
	multisetHash   Hash
	cumulativeHash Hash
}

var testVectors []testVector

var testVectorsStrings = []struct {
	dataElementHex string
	multisetHash   string
	cumulativeHash string
}{
	{
		"982051fd1e4ba744bbbe680e1fee14677ba1a3c3540bf7b1cdb606e857233e0e00000000010000000100f2052a0100000043410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac",
		"8aba1bb6ea174fba90d4a626463859646ff02c854fb99f2619c9200fa70c2e93",
		"8aba1bb6ea174fba90d4a626463859646ff02c854fb99f2619c9200fa70c2e93",
	},
	{
		"d5fdcc541e25de1c7a5addedf24858b8bb665c9f36ef744ee42c316022c90f9b00000000020000000100f2052a010000004341047211a824f55b505228e4c3d5194c1fcfaa15a456abdf37f9b9d97a4040afc073dee6c89064984f03385237d92167c13e236446b417ab79a0fcae412ae3316b77ac",
		"95fb628ed07fd2187fda0184f3966312ba98baf3ac83639b8c6dd7cc7a09d8b4",
		"b85145198ec445421a85748101ec2bc019daa5ecda8eda2380181d6c8ebf3463",
	},
	{
		"44f672226090d85db9a9f2fbfe5f0f9609b387af7be5b7fbb7a1767c831c9e9900000000030000000100f2052a0100000043410494b9d3e76c5b1629ecf97fff95d7a4bbdac87cc26099ada28066c6ff1eb9191223cd897194a08d0c2726c5747f1db49e8cf90e75dc3e3550ae9b30086f3cd5aaac",
		"78f145af890dbbb59a4d86b6376e282c16af61c7cdf33d495357df4be9c35013",
		"e8cf5b87a35612fda22dcc06ce3d512a44c4e46c118594adc71190515b418a52",
	},
}

var (
	maxMuHash = MuHash{}
)

func TestMain(m *testing.M) {
	for _, vector := range testVectorsStrings {
		res := testVector{}
		var err error
		res.dataElement, err = hex.DecodeString(vector.dataElementHex)
		if err != nil {
			panic(fmt.Sprintf("failed parsing the hex: '%s', err: '%s'", vector.dataElementHex, err))
		}
		data, err := hex.DecodeString(vector.multisetHash)
		if err != nil {
			panic(fmt.Sprintf("failed parsing the hex: '%s', err: '%s'", vector.multisetHash, err))
		}
		err = res.multisetHash.SetBytes(data)
		if err != nil {
			panic(fmt.Sprintf("failed setting the hash: '%x', err: '%s'", data, err))
		}
		data, err = hex.DecodeString(vector.cumulativeHash)
		if err != nil {
			panic(fmt.Sprintf("failed parsing the hex: '%s', err: '%s'", vector.cumulativeHash, err))
		}
		err = res.cumulativeHash.SetBytes(data)
		if err != nil {
			panic(fmt.Sprintf("failed setting the hash: '%x', err: '%s'", data, err))
		}
		testVectors = append(testVectors, res)
	}
	var max uint3072
	for i := range max {
		max[i] = maxUint
	}
	maxMuHash = MuHash{
		numerator:   max,
		denominator: max,
	}

	os.Exit(m.Run())
}

func elementFromByte(i byte) []byte {
	out := [32]byte{i}
	return out[:]
}

func TestRandomMuHashArithmetic(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		var res Hash
		var table [4]byte
		for i := 0; i < 4; i++ {
			table[i] = byte(r.Int31n(1 << 3)) // [0, 2^3) can't overflow byte.
		}
		for order := 0; order < 4; order++ {
			acc := NewMuHash()
			for i := 0; i < 4; i++ {
				t := table[i^order]
				if (t & 4) == 1 {
					acc.Remove(elementFromByte(t & 3))
				} else {
					acc.Add(elementFromByte(t & 3))
				}
			}
			out := acc.Finalize()
			if order == 0 {
				res = out
			} else {
				if !res.IsEqual(&out) {
					t.Fatalf("Expected %s == %s", res, out)
				}
			}
		}

		x := elementFromByte(byte(r.Int31n(1 << 3))) // x=X
		y := elementFromByte(byte(r.Int31n(1 << 3))) // x=X, y=Y
		z := NewMuHash()                             // x=X, y=X, z=1.
		yx := NewMuHash()                            // x=X, y=X, z=1, yx=1
		yx.Add(y)                                    // x=X, y=X, z=1, yx=Y
		yx.Add(x)                                    // x=X, y=X, z=1, yx=Y*X
		yx.normalize()

		z.Add(x)                       // x=X, y=Y, z=X, yx=Y*X
		z.Add(y)                       // x=X, y=Y, z=X*Y, yx = Y*X
		z.removeElement(&yx.numerator) // x=X, y=Y, z=1, yx=Y*X

		if !z.Finalize().IsEqual(&EmptyMuHashHash) {
			t.Fatalf("Expected %s == %s", z.Finalize(), EmptyMuHashHash)
		}
	}
}

func TestNewPreComputed(t *testing.T) {
	t.Parallel()
	expected := "afd9eb8885b98062d6720cfb034886bc332b10251adc037d2a5fc4c17c11832f"
	acc := NewMuHash()
	acc.Add(elementFromByte(0))
	acc.Add(elementFromByte(1))
	acc.Remove(elementFromByte(2))
	if acc.Finalize().String() != expected {
		t.Fatalf("Expected %s == %s", expected, acc.Finalize())
	}

	acc = NewMuHash()
	acc.Add(elementFromByte(0))
	acc.Add(elementFromByte(1))
	acc.Remove(elementFromByte(2))
	if acc.Finalize().String() != expected {
		t.Fatalf("Expected %s == %s", expected, acc.Finalize())
	}
}

func TestMuHash_Serialize(t *testing.T) {
	t.Parallel()
	expected, err := hex.DecodeString("ad8b80dae66ba6c0c63c02079cdac340f26ca6614584431de4c46a46e521bc5c0e5bb7e475e2df1c501c34dfd9534731a6e631c9d4fce641da66b08a26f8ebb738e0bc8bb5ae07f9fc58bdcf790444df315a8eadc3edc8e27325623fce2e25c6d03a785eb482c9887af6b72bd757e977c958e25ea33b631c77e52713b5c66e8f8d7bdc04f50ce4cc68eca4dde3a1621de22c1634de13fdae65b43ee1caeefa71804276b84a159669e0522fde03364311bd57607e01b68b8e55d68b84c1c8e038248de9af3c7aeb16a9261edbe6ece62a14a4d770fbf006d179a9c5ca8226a5dae7e2cb81a31c3db35aa18d3a3eac994c7e9fc61ea0ebb32b49dd2a6c7e1eca086a39b9ee29fffe587e46a6d25a1df5dd285b43daf3176432a58725940067f69eb6fe8b3f80e137a2642fb8f66395cd3865a3259a4499351191335ca53d04153179717125a500c87e95493a25547bf1e96ea18d174bd857debdb10d2f33d1ce220da7ffb1e56ef5be8d6a855b5b8cea70b3dd32cf9bc533fca33d71560ac6e182")
	if err != nil {
		t.Fatalf("Failed deserializing hex string: %v", err)
	}
	check := NewMuHash()
	check.Add(elementFromByte(1))
	check.Add(elementFromByte(2))
	serialized := check.Serialize()
	if !bytes.Equal(expected, serialized[:]) {
		t.Fatalf("Expected %x == %s", expected, serialized)
	}

	deserialized, err := DeserializeMuHash(serialized)
	if err != nil {
		t.Fatalf("Failed deserializing muhash: %v", err)
	}
	checkHash := check.Finalize()
	if !deserialized.Finalize().IsEqual(&checkHash) {
		t.Fatalf("Expected %s == %s", deserialized.Finalize(), check.Finalize())
	}

	overflow, err := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	if err != nil {
		t.Fatalf("Failed deserializing hex string: %v", err)
	}

	if copy(serialized[:], overflow) != len(overflow) {
		t.Fatalf("Failed copying %x into SerializedMuHash", overflow)
	}

	_, err = DeserializeMuHash(serialized)
	if !errors.Is(err, errOverflow) {
		t.Fatalf("Expected %s, instead found: %s", errOverflow, err)
	}

	serializedZeros := SerializedMuHash{}
	zeroed := NewMuHash()
	zeroed.addElement(&uint3072{}) // multiply by zero.
	serialized = zeroed.Serialize()
	if !bytes.Equal(serialized[:], serializedZeros[:]) {
		t.Fatalf("expected serialized to be all zeros, instead found: %s", serialized)
	}
	deserialized, err = DeserializeMuHash(serialized)
	if err != nil {
		t.Fatalf("Failed deserializing zeros: %v", err)
	}
	zeroed.normalize()
	deserialized.normalize()
	if zeroed.numerator != deserialized.numerator {
		t.Fatalf("Expected %x == %x", zeroed.numerator, deserialized.numerator)
	}
}

func TestVectorsMuHash_Hash(t *testing.T) {
	t.Parallel()
	for _, test := range testVectors {
		m := NewMuHash()
		m.Add(test.dataElement)
		mFinal := m.Finalize()
		if !m.Finalize().IsEqual(&test.multisetHash) {
			t.Fatalf("MuHash-Hash returned incorrect hash serialization, expected: '%s', found: '%s'", test.multisetHash, mFinal)
		}
	}
	m := NewMuHash()
	if !m.Finalize().IsEqual(&EmptyMuHashHash) {
		t.Fatalf("Empty set did not return zero hash, got: '%s' instead", m.Finalize())
	}
}

func TestVectorsMuHash_AddRemove(t *testing.T) {
	t.Parallel()
	m := NewMuHash()
	for i, test := range testVectors {
		m.Add(test.dataElement)
		mFinal := m.Finalize()
		if !mFinal.IsEqual(&test.cumulativeHash) {
			t.Fatalf("Test #%d: MuHash-Add returned incorrect hash. Expected '%s' but got '%s'", i, test.cumulativeHash, mFinal)
		}
	}

	for i := len(testVectors) - 1; i > 0; i-- {
		m.Remove(testVectors[i].dataElement)
		mFinal := m.Finalize()
		if !mFinal.IsEqual(&testVectors[i-1].cumulativeHash) {
			t.Fatalf("Test #%d: MuHash-Remove returned incorrect hash. Expected '%s' but got '%s'", i, testVectors[i].cumulativeHash, mFinal)
		}
	}
}

func TestVectorsMuHash_CombineSubtract(t *testing.T) {
	t.Parallel()
	m1 := NewMuHash()
	zeroHash := m1.Finalize()

	for _, test := range testVectors {
		m1.Add(test.dataElement)
	}

	m2 := NewMuHash()
	for _, test := range testVectors {
		m2.Remove(test.dataElement)
	}
	m1.Combine(m2)
	if !m1.Finalize().IsEqual(&zeroHash) {
		t.Fatalf("m1 was expected to have a zero hash, but was '%s' instead", m1.Finalize())
	}
}

func TestVectorsMuHash_Commutativity(t *testing.T) {
	t.Parallel()
	m := NewMuHash()
	zeroHash := m.Finalize()

	// Check that if we subtract values from zero and then re-add them, we return to zero.
	for _, test := range testVectors {
		m.Remove(test.dataElement)
	}

	for _, test := range testVectors {
		m.Add(test.dataElement)
	}
	if !m.Finalize().IsEqual(&zeroHash) {
		t.Fatalf("m was expected to be zero hash, but was '%s' instead", m.Finalize())
	}

	// Here we first remove an element from an empty multiset, and then add some other
	// elements, and then we create a new empty multiset, then we add the same elements
	// we added to the previous multiset, and then we remove the same element we remove
	// the same element we removed from the previous multiset. According to commutativity
	// laws, the result should be the same.
	removeIndex := 0
	removeData := testVectors[removeIndex].dataElement

	m1 := NewMuHash()
	m1.Remove(removeData)

	for i, test := range testVectors {
		if i != removeIndex {
			m1.Add(test.dataElement)
		}
	}

	m2 := NewMuHash()
	for i, test := range testVectors {
		if i != removeIndex {
			m2.Add(test.dataElement)
		}
	}
	m2.Remove(removeData)

	m2Hash := m2.Finalize()
	if !m1.Finalize().IsEqual(&m2Hash) {
		t.Fatalf("m1 and m2 was exepcted to have the same hash, but got instead m1 '%s' and m2 '%s'", m1.Finalize(), m2.Finalize())
	}
}

func TestParseMuHashFail(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	data := SerializedMuHash{}
	copy(data[:], prime.Bytes())
	// reverse because it's little endian.
	for i := len(data)/2 - 1; i >= 0; i-- {
		opp := len(data) - 1 - i
		data[i], data[opp] = data[opp], data[i]
	}

	_, err := DeserializeMuHash(&data)
	if err == nil {
		t.Fatalf("shouldn't be able to parse a multiset bigger with x bigger than the field size: '%s'", err)
	}
	data[0] = 0
	_, err = DeserializeMuHash(&data)
	if err != nil {
		t.Errorf("It should parse muhash lower than the field size %v", err)
	}
	set := NewMuHash()
	n, err := r.Read(data[:])
	if err != nil || n != len(data) {
		t.Fatalf("failed generating random data '%s' '%d' ", err, n)
	}
	set.Add(data[:])

}

func TestMuHash_Reset(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	set := NewMuHash()
	emptySetHash := NewMuHash().Finalize()
	data := [100]byte{}
	n, err := r.Read(data[:])
	if err != nil || n != len(data) {
		t.Fatalf("failed generating random data '%v' '%d' ", err, n)
	}
	set.Add(data[:])
	if set.Finalize().IsEqual(&emptySetHash) {
		t.Errorf("expected set to be empty. found: '%s'", set.Finalize())
	}
	set.Reset()
	if !set.Finalize().IsEqual(&emptySetHash) {
		t.Errorf("expected set to be empty. found: '%s'", set.Finalize())
	}
}

const loopsN = 1024

func TestMuHashAddRemove(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	list := [loopsN][100]byte{}
	set := NewMuHash()
	set2Hash := set.Clone().Finalize()
	for i := 0; i < loopsN; i++ {
		data := [100]byte{}
		n, err := r.Read(data[:])
		if err != nil || n != len(data) {
			t.Fatalf("Failed generating random data. read: '%d' bytes. .'%v'", n, err)
		}
		set.Add(data[:])
		list[i] = data
	}
	if set.Finalize().IsEqual(&set2Hash) {
		t.Errorf("sets are the same when they should be different: set '%s', set2: %s\n", set.Finalize(), set2Hash)
	}

	for i := 0; i < loopsN; i++ {
		set.Remove(list[i][:])
	}
	if !set.Finalize().IsEqual(&set2Hash) {
		t.Errorf("sets are different when they should be the same: set1: '%s', set2: '%s'\n", set.Finalize(), set2Hash)
	}
}

func BenchmarkMuHash_Add(b *testing.B) {
	set := NewMuHash()
	var data [100]byte
	for i := range data {
		data[i] = 0xFF
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(data[:])
	}
}

func BenchmarkMuHash_Remove(b *testing.B) {
	set := NewMuHash()
	var data [100]byte
	for i := range data {
		data[i] = 0xFF
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Remove(data[:])
	}
}

func BenchmarkMuHash_CombineWorst(b *testing.B) {
	set := NewMuHash()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Combine(&maxMuHash)
	}
}

func BenchmarkMuHash_CombineBest(b *testing.B) {
	set := NewMuHash()
	empty := NewMuHash()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Combine(empty)
	}
}

func BenchmarkMuHash_CombineRand(b *testing.B) {
	r := rand.New(rand.NewSource(0))
	set := NewMuHash()
	var element MuHash
	for i := range element.numerator {
		element.numerator[i] = uint(r.Uint64())
		element.denominator[i] = uint(r.Uint64())
	}
	element.normalize()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Combine(&element)
	}
}

func BenchmarkMuHash_Clone(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		maxMuHash.Clone()
	}
}

func BenchmarkMuHash_normalizeWorst(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		maxMuHash.Clone().normalize()
	}
}

func BenchmarkMuHash_normalizeBest(b *testing.B) {
	empty := NewMuHash()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		empty.Clone().normalize()
	}
}

func BenchmarkMuHash_normalizeRand(b *testing.B) {
	r := rand.New(rand.NewSource(0))
	var set MuHash
	for i := range set.numerator {
		set.numerator[i] = uint(r.Uint64())
		set.denominator[i] = uint(r.Uint64())
	}
	set.normalize()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Clone().normalize()
	}
}

func BenchmarkMuHash_Finalize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		maxMuHash.Clone().Finalize()
	}
}
