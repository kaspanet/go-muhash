package muhash

import (
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
	benchmarkData100Bytes = make([][100]byte, BenchmarkIterations)
	benchmarkDataMuHash   = make([]MuHash, BenchmarkIterations)
)

func TestMain(m *testing.M) {
	for _, vector := range testVectorsStrings {
		res := testVector{}
		err := errors.New("")
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

	r := rand.New(rand.NewSource(1))
	for i := 0; i < BenchmarkIterations; i++ {
		n, err := r.Read(benchmarkData100Bytes[i][:])
		if err != nil || n != len(benchmarkData100Bytes[i]) {
			panic(err)
		}
	}

	set := NewMuHash()
	for i := 0; i < BenchmarkIterations; i++ {
		data := [100]byte{}
		n, err := r.Read(data[:])
		if err != nil || n != len(data) {
			panic(err)
		}
		set.Add(data[:])
		benchmarkDataMuHash[i] = set.Clone()
	}

	os.Exit(m.Run())
}

func TestVectorsMuHash_Hash(t *testing.T) {
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
	if !m1.Finalize().IsEqual(zeroHash) {
		t.Fatalf("m1 was expected to have a zero hash, but was '%s' instead", m1.Finalize())
	}
}

func TestVectorsMuHash_Commutativity(t *testing.T) {
	m := NewMuHash()
	zeroHash := m.Finalize()

	// Check that if we subtract values from zero and then re-add them, we return to zero.
	for _, test := range testVectors {
		m.Remove(test.dataElement)
	}

	for _, test := range testVectors {
		m.Add(test.dataElement)
	}
	if !m.Finalize().IsEqual(zeroHash) {
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

	if !m1.Finalize().IsEqual(m2.Finalize()) {
		t.Fatalf("m1 and m2 was exepcted to have the same hash, but got instead m1 '%s' and m2 '%s'", m1.Finalize(), m2.Finalize())
	}
}

func TestParseMuHashFail(t *testing.T) {
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
		t.Errorf("shouldn't be able to parse a multiset bigger with x bigger than the field size: '%s'", err)
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
	r := rand.New(rand.NewSource(1))
	set := NewMuHash()
	emptySet := NewMuHash()
	data := [100]byte{}
	n, err := r.Read(data[:])
	if err != nil || n != len(data) {
		t.Fatalf("failed generating random data '%x' '%d' ", err, n)
	}
	set.Add(data[:])
	if *set.Finalize() == *emptySet.Finalize() {
		t.Errorf("expected set to be empty. found: '%x'", set.Finalize())
	}
	set.Reset()
	if *set.Finalize() != *emptySet.Finalize() {
		t.Errorf("expected set to be empty. found: '%x'", set.Finalize())
	}
}

const loopsN = 150

func TestMuHashAddRemove(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	list := [loopsN][100]byte{}
	set := NewMuHash()
	set2 := set.Clone()
	for i := 0; i < loopsN; i++ {
		data := [100]byte{}
		n, err := r.Read(data[:])
		if err != nil || n != len(data) {
			t.Fatalf("Failed generating random data. read: '%d' bytes. .'%x'", n, err)
		}
		set.Add(data[:])
		list[i] = data
	}
	if set.Finalize().IsEqual(set2.Finalize()) {
		t.Errorf("sets are the same when they should be different: set '%x'\n", set.Finalize())
	}

	for i := 0; i < loopsN; i++ {
		set.Remove(list[i][:])
	}
	if !set.Finalize().IsEqual(set2.Finalize()) {
		t.Errorf("sets are different when they should be the same: set1: '%x', set2: '%x'\n", set.Finalize(), set2.Finalize())
	}
}

const BenchmarkIterations = 1_000_000

func BenchmarkMuHash_Add(b *testing.B) {
	b.ReportAllocs()
	set := NewMuHash()
	for i := 0; i < b.N; i++ {
		set.Add(benchmarkData100Bytes[i][:])
	}
}

func BenchmarkMuHash_Remove(b *testing.B) {
	b.ReportAllocs()
	set := NewMuHash()
	for i := 0; i < b.N; i++ {
		set.Remove(benchmarkData100Bytes[i][:])
	}
}

func BenchmarkMuHash_Combine(b *testing.B) {
	b.ReportAllocs()
	set := NewMuHash()
	for i := 0; i < b.N; i++ {
		set.Combine(benchmarkDataMuHash[i])
	}
}

func BenchmarkMuHash_Finalize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkDataMuHash[i].Finalize()
	}
}
