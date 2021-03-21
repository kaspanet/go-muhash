package muhash

// #include "muhash.h"
import "C"

func oneNum3072() Num3072 {
	return Num3072{limbs: [48]C.ulong{1}}
}

type Num3072 C.Num3072

func (lhs *Num3072) SetToOne() {
	C.Num3072_SetToOne((*C.Num3072)(lhs))
}

func (lhs *Num3072) Mul(rhs *Num3072) {
	C.Num3072_Multiply((*C.Num3072)(lhs), (*C.Num3072)(rhs))
}

func (lhs *Num3072) Square() {
	C.Num3072_Square((*C.Num3072)(lhs))
}

func (lhs *Num3072) Divide(rhs *Num3072) {
	C.Num3072_Divide((*C.Num3072)(lhs), (*C.Num3072)(rhs))
}

func (lhs *Num3072) IsOverflow() bool {
	return C.Num3072_IsOverflow((*C.Num3072)(lhs)) == 1
}

func (lhs *Num3072) GetInverse() Num3072 {
	return (Num3072)(C.Num3072_GetInverse((*C.Num3072)(lhs)))
}
