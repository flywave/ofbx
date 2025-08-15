package ofbx

import (
	"math"

	"github.com/oakmound/oak/v2/alg"
	"github.com/oakmound/oak/v2/alg/floatgeom"
)

// RotationOrder determines the dimension order for rotation
type RotationOrder int

// A block of rotation order sets
const (
	EulerXYZ   RotationOrder = iota
	EulerXZY   RotationOrder = iota
	EulerYZX   RotationOrder = iota
	EulerYXZ   RotationOrder = iota
	EulerZXY   RotationOrder = iota
	EulerZYX   RotationOrder = iota
	SphericXYZ RotationOrder = iota // Currently unsupported. Treated as EulerXYZ.
)

// String returns the string representation of RotationOrder
func (o RotationOrder) String() string {
	switch o {
	case EulerXYZ:
		return "EulerXYZ"
	case EulerXZY:
		return "EulerXZY"
	case EulerYZX:
		return "EulerYZX"
	case EulerYXZ:
		return "EulerYXZ"
	case EulerZXY:
		return "EulerZXY"
	case EulerZYX:
		return "EulerZYX"
	case SphericXYZ:
		return "SphericXYZ"
	default:
		return "Unknown"
	}
}

func (o RotationOrder) rotationMatrix(euler floatgeom.Point3) Matrix {
	x, y, z := euler.X()*alg.DegToRad, euler.Y()*alg.DegToRad, euler.Z()*alg.DegToRad
	a, b := math.Cos(x), math.Sin(x)
	c, d := math.Cos(y), math.Sin(y)
	e, f := math.Cos(z), math.Sin(z)

	te := makeIdentity()

	switch o {
	case EulerXYZ:
		te.m[0] = c * e
		te.m[4] = -c * f
		te.m[8] = d

		te.m[1] = a*f + b*e*d
		te.m[5] = a*e - b*f*d
		te.m[9] = -b * c

		te.m[2] = b*f - a*e*d
		te.m[6] = b*e + a*f*d
		te.m[10] = a * c

	case EulerYXZ:
		ce, cf := c*e, c*f
		de, df := d*e, d*f

		te.m[0] = ce + df*b
		te.m[4] = de*b - cf
		te.m[8] = a * d

		te.m[1] = a * f
		te.m[5] = a * e
		te.m[9] = -b

		te.m[2] = cf*b - de
		te.m[6] = df + ce*b
		te.m[10] = a * c

	case EulerZXY:
		ce, cf := c*e, c*f
		de, df := d*e, d*f

		te.m[0] = ce - df*b
		te.m[4] = -a * f
		te.m[8] = de + cf*b

		te.m[1] = cf + de*b
		te.m[5] = a * e
		te.m[9] = df - ce*b

		te.m[2] = -a * d
		te.m[6] = b
		te.m[10] = a * c

	case EulerZYX:
		ae, af := a*e, a*f
		be, bf := b*e, b*f

		te.m[0] = c * e
		te.m[4] = be*d - af
		te.m[8] = ae*d + bf

		te.m[1] = c * f
		te.m[5] = bf*d + ae
		te.m[9] = af*d - be

		te.m[2] = -d
		te.m[6] = b * c
		te.m[10] = a * c

	case EulerYZX:
		ac, ad := a*c, a*d
		bc, bd := b*c, b*d

		te.m[0] = c * e
		te.m[4] = bd - ac*f
		te.m[8] = bc*f + ad

		te.m[1] = f
		te.m[5] = a * e
		te.m[9] = -b * e

		te.m[2] = -d * e
		te.m[6] = ad*f + bc
		te.m[10] = ac - bd*f

	case EulerXZY:
		ac, ad := a*c, a*d
		bc, bd := b*c, b*d

		te.m[0] = c * e
		te.m[4] = -f
		te.m[8] = d * e

		te.m[1] = ac*f + bd
		te.m[5] = a * e
		te.m[9] = ad*f - bc

		te.m[2] = bc*f - ad
		te.m[6] = b * e
		te.m[10] = bd*f + ac

	case SphericXYZ:
		// SphericXYZ目前不支持，当作EulerXYZ处理
		te.m[0] = c * e
		te.m[4] = -c * f
		te.m[8] = d

		te.m[1] = a*f + b*e*d
		te.m[5] = a*e - b*f*d
		te.m[9] = -b * c

		te.m[2] = b*f - a*e*d
		te.m[6] = b*e + a*f*d
		te.m[10] = a * c

	default:
		panic("Unsupported rotation order")
	}

	// 最后三列保持单位矩阵
	te.m[3], te.m[7], te.m[11] = 0, 0, 0
	te.m[12], te.m[13], te.m[14] = 0, 0, 0
	te.m[15] = 1

	return te
}
