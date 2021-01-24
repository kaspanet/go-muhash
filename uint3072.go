package muhash

import (
	"math/big"
	"math/bits"
)

const (
	limbs   = elementWordSize
	maxUint = ^uint(0)
)

type uint3072 [limbs]uint

// Extract the lowest limb of [low,high,carry] into n, and left shift the number by 1 limb.
func extract3(low, high, carry, n *uint) {
	*n = *low
	*low = *high
	*high = *carry
	*carry = 0
}

// [low,high] = a * b
func mul(low, high *uint, a, b uint) {
	*high, *low = bits.Mul(a, b)
}

// [c0,c1,c2] += n * [d0,d1,d2]. c2 is 0 initially
func mulnadd3(c0, c1, c2 *uint, d0, d1, d2, n uint) {
	var carry, tmpLow uint
	tmpHigh, tmpLow := bits.Mul(d0, n)
	*c0, carry = bits.Add(*c0, tmpLow, 0)
	tmpHigh += carry

	tmpHigh2, tmpLow2 := bits.Mul(d1, n)

	*c1, carry = bits.Add(tmpLow2, *c1, 0)
	tmpHigh2 += carry
	*c1, carry = bits.Add(*c1, tmpHigh, 0)
	tmpHigh2 += carry

	*c2 = tmpHigh2 + d2*n
}

// [low,high] *= n
func muln2(low, high *uint, n uint) {
	var tmpLow, tmpHigh uint
	tmpHigh, *low = bits.Mul(*low, n)
	_, tmpLow = bits.Mul(*high, n)
	*high = tmpHigh + tmpLow
}

// [low,high,carry] += a * b
func muladd3(low, high, carry *uint, a, b uint) {
	var tmpCarry uint
	tmpHigh, tmpLow := bits.Mul(a, b)
	*low, tmpCarry = bits.Add(*low, tmpLow, tmpCarry)
	*high, tmpCarry = bits.Add(*high, tmpHigh, tmpCarry)
	*carry += tmpCarry
}

// [low,high,carry] += 2 * a * b
func muldbladd3(low, high, carry *uint, a, b uint) {
	var tmpCarry uint
	tmpHigh, tmpLow := bits.Mul(a, b)

	*low, tmpCarry = bits.Add(*low, tmpLow, tmpCarry)
	*high, tmpCarry = bits.Add(*high, tmpHigh, tmpCarry)
	*carry += tmpCarry

	*low, tmpCarry = bits.Add(*low, tmpLow, 0)
	*high, tmpCarry = bits.Add(*high, tmpHigh, tmpCarry)
	*carry += tmpCarry
}

func addnextract2(low, high, n *uint, a uint) {
	var carry uint

	*low, carry = bits.Add(*low, a, carry)
	*high, carry = bits.Add(*high, 0, carry)
	*high, carry = bits.Add(*high, 0, carry)

	// extract
	*n = *low
	*low = *high
	*high = carry
}

func assert(cond bool) {
	if !cond {
		panic("assert failed")
	}
}

func (lhs *uint3072) Mul(rhs *uint3072) {
	var carryLow, carryHigh, carryHighest uint
	var tmp uint3072
	// Compute limbs 0..N-2 of lhs*rhs into tmp, including one reduction.
	for j := 0; j < limbs-1; j++ {
		var low, high, carry uint
		mul(&low, &high, lhs[1+j], rhs[limbs+j-(1+j)])
		for i := 2 + j; i < limbs; i++ {
			muladd3(&low, &high, &carry, lhs[i], rhs[limbs+j-i])
		}
		mulnadd3(&carryLow, &carryHigh, &carryHighest, low, high, carry, primeDiff)
		for i := 0; i < j+1; i++ {
			muladd3(&carryLow, &carryHigh, &carryHighest, lhs[i], rhs[j-i])
		}
		extract3(&carryLow, &carryHigh, &carryHighest, &tmp[j])
	}

	// Compute limb N-1 of a*b into tmp.
	assert(carryHighest == 0)
	for i := 0; i < limbs; i++ {
		muladd3(&carryLow, &carryHigh, &carryHighest, lhs[i], rhs[limbs-1-i])
	}
	extract3(&carryLow, &carryHigh, &carryHighest, &tmp[limbs-1])

	// Perform a second reduction.
	muln2(&carryLow, &carryHigh, primeDiff)
	for j := 0; j < limbs; j++ {
		addnextract2(&carryLow, &carryHigh, &lhs[j], tmp[j])
	}

	assert(carryHighest == 0)
	assert(carryLow == 0 || carryLow == 1)

	// Perform up to two more reductions if the internal state has already
	// overflown the MAX of uint3072 or if it is larger than the modulus or
	// if both are the case.
	if lhs.IsOverflow() {
		lhs.FullReduce()
	}
	if carryLow > 0 {
		lhs.FullReduce()
	}
}

func (lhs *uint3072) Square() {
	var low, high, carry uint
	var tmp uint3072

	// Compute limbs 0..N-2 of this*this into tmp, including one reduction.
	for j := 0; j < limbs-1; j++ {
		var carryLow, carryHigh, carryHighest uint

		for i := 0; i < (limbs-1-j)/2; i++ {
			muldbladd3(&carryLow, &carryHigh, &carryHighest, lhs[i+j+1], lhs[limbs-1-i])
		}
		if (j+1)&1 == 1 {
			muladd3(&carryLow, &carryHigh, &carryHighest, lhs[(limbs-1-j)/2+j+1], lhs[limbs-1-(limbs-1-j)/2])
		}
		mulnadd3(&low, &high, &carry, carryLow, carryHigh, carryHighest, primeDiff)

		for i := 0; i < (j+1)/2; i++ {
			muldbladd3(&low, &high, &carry, lhs[i], lhs[j-i])
		}

		if (j+1)&1 == 1 {
			muladd3(&low, &high, &carry, lhs[(j+1)/2], lhs[j-(j+1)/2])
		}
		extract3(&low, &high, &carry, &tmp[j])
	}
	assert(carry == 0)

	for i := 0; i < limbs/2; i++ {
		muldbladd3(&low, &high, &carry, lhs[i], lhs[limbs-1-i])
	}
	extract3(&low, &high, &carry, &tmp[limbs-1])

	// Perform a second reduction
	muln2(&low, &high, primeDiff)
	for j := 0; j < limbs; j++ {
		addnextract2(&low, &high, &lhs[j], tmp[j])
	}

	assert(high == 0)
	assert(low == 0 || low == 1)

	// Perform up to two more reductions if the internal state has already
	// overflown the MAX of uint3072 or if it is larger than the modulus or
	// if both are the case.
	if lhs.IsOverflow() {
		lhs.FullReduce()
	}
	if low > 0 {
		lhs.FullReduce()
	}
}

func (lhs *uint3072) Divide(rhs *uint3072) {
	if lhs.IsOverflow() {
		lhs.FullReduce()
	}
	if rhs.IsOverflow() {
		rhs.FullReduce()
	}

	rightWords := make([]big.Word, limbs)
	for i := range rightWords {
		rightWords[i] = big.Word(rhs[i])
	}
	var right big.Int
	right.SetBits(rightWords)
	right.ModInverse(&right, prime)

	var inv uint3072
	for i, word := range right.Bits() {
		inv[i] = uint(word)
	}
	lhs.Mul(&inv)
	if lhs.IsOverflow() {
		lhs.FullReduce()
	}
}

// lhs = lhs^(2^exp) * mul
func (lhs *uint3072) squareNmul(exp int, mul *uint3072) {
	for j := 0; j < exp; j++ {
		lhs.Square()
	}
	lhs.Mul(mul)
}

func (lhs *uint3072) GetInverse() uint3072 {
	// For fast exponentiation a sliding window exponentiation with repunit
	// precomputation is utilized. See "Fast Point Decompression for Standard
	// Elliptic Curves" (Brumley, Järvinen, 2008).

	var powers [12]uint3072 // powers[i] = a^(2^(2^i)-1)
	var res uint3072

	powers[0] = *lhs
	for i := 0; i < 11; i++ {
		powers[i+1] = powers[i]
		for j := 0; j < (1 << i); j++ {
			powers[i+1].Square()
		}
		powers[i+1].Mul(&powers[i])
	}
	res = powers[11]

	res.squareNmul(512, &powers[9])
	res.squareNmul(256, &powers[8])
	res.squareNmul(128, &powers[7])
	res.squareNmul(64, &powers[6])
	res.squareNmul(32, &powers[5])
	res.squareNmul(8, &powers[3])
	res.squareNmul(2, &powers[1])
	res.squareNmul(1, &powers[0])
	res.squareNmul(5, &powers[2])
	res.squareNmul(3, &powers[0])
	res.squareNmul(2, &powers[0])
	res.squareNmul(4, &powers[0])
	res.squareNmul(4, &powers[1])
	res.squareNmul(3, &powers[0])
	return res
}

func (lhs *uint3072) IsOverflow() bool {
	if lhs[0] <= maxUint-primeDiff {
		return false
	}
	for i := 1; i < limbs; i++ {
		if lhs[i] != maxUint {
			return false
		}
	}
	return true
}

func (lhs *uint3072) FullReduce() {
	low := uint(primeDiff)
	var high uint
	for i := 0; i < limbs; i++ {
		addnextract2(&low, &high, &lhs[i], lhs[i])
	}
}

func (lhs *uint3072) SetToOne() {
	lhs[0] = 1
	for i := 1; i < limbs; i++ {
		lhs[i] = 0
	}
}

func one() uint3072 {
	return uint3072{1}
}
