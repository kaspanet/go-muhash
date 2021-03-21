package muhash

import (
	"math/rand"
	"testing"
)


type CUint = _Ctype_ulong

func TestNum3072_GetInverse(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(0))
	var element Num3072
	for i := 0; i < 5; i++ {
		for i := range element.limbs {
			element.limbs[i] = CUint(r.Uint64())
		}
		inv := element.GetInverse()
		again := inv.GetInverse()

		if again != element {
			t.Fatalf("Expected double inverting to be equal, found: %v != %v", again, element)
		}
	}
}

func num3072equalToUint(a *Num3072, b uint) bool {
	if uint(a.limbs[0]) != b {
		return false
	}
	for j := 1; j < len(a.limbs); j++ {
		if a.limbs[j] != 0 {
			return false
		}
	}
	return true
}

func TestNum3072_DivOverflow(t *testing.T) {
	var overflownOne Num3072
	for i := range overflownOne.limbs {
		overflownOne.limbs[i] = CUint(maxUint)
	}
	overflownOne.limbs[0] -= primeDiff - 2 // full maxUINT is 2^3072-1, so I need to substract the primeDiff and add 2 to make it overflown 1.
	regularOne := oneNum3072()
	overflownOne.Divide(&regularOne)
	if overflownOne != oneNum3072() {
		t.Fatalf("Expected overflownOne/one to be equal to one, instead got: %v", overflownOne)
	}

}

func TestNum3072_MulMax(t *testing.T) {
	t.Parallel()
	var max Num3072
	for i := range max.limbs {
		max.limbs[i] = CUint(maxUint)
	}
	max.limbs[0] -= primeDiff
	copyMax := max
	max.Mul(&copyMax)
	if !num3072equalToUint(&max, 1) {
		t.Fatalf("(p-1)*(p-1) mod p should equal 1, instead got: %v", max)
	}
}

func TestNum3072MulDiv(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(1))
	var list [loopsN]Num3072
	start := oneNum3072()
	for i := 0; i < loopsN; i++ {
		for n := range list[i].limbs {
			list[i].limbs[n] = CUint(r.Uint64())
		}
		start.Mul(&list[i])
	}
	if start == oneNum3072() {
		t.Errorf("start is 1 even though it shouldn't be: start '%x', one: %x\n", start, one())
	}

	for i := 0; i < loopsN; i++ {
		start.Divide(&list[i])
	}
	if start != oneNum3072() {
		t.Errorf("start should be 1 but it isn't: start: '%x', one: '%x'\n", start, one())
	}
}
