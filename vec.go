package ofbx

import "math"

// Vector types used throughout the library. They are fixed-size arrays so
// they can be compared with == and used as map keys like the floatgeom types
// this project originally depended on.
//
// Note the Point4 axis order quirk inherited from the library's floatgeom
// heritage: W is component 0 and X/Y/Z are 1/2/3 (quaternion layout). FBX
// vertex colors are stored as (r, g, b, a) in components 0..3, so always use
// direct indexing for colors.

// Point2 is a 2D vector (x, y).
type Point2 [2]float64

// X returns the x component.
func (p Point2) X() float64 { return p[0] }

// Y returns the y component.
func (p Point2) Y() float64 { return p[1] }

// Point3 is a 3D vector (x, y, z).
type Point3 [3]float64

// X returns the x component.
func (p Point3) X() float64 { return p[0] }

// Y returns the y component.
func (p Point3) Y() float64 { return p[1] }

// Z returns the z component.
func (p Point3) Z() float64 { return p[2] }

// Add returns the component-wise sum of the operands.
func (p Point3) Add(ps ...Point3) Point3 {
	out := p
	for _, v := range ps {
		out[0] += v[0]
		out[1] += v[1]
		out[2] += v[2]
	}
	return out
}

// Sub returns the component-wise difference of the operands.
func (p Point3) Sub(ps ...Point3) Point3 {
	out := p
	for _, v := range ps {
		out[0] -= v[0]
		out[1] -= v[1]
		out[2] -= v[2]
	}
	return out
}

// Mul returns the component-wise product of the operands.
func (p Point3) Mul(ps ...Point3) Point3 {
	out := p
	for _, v := range ps {
		out[0] *= v[0]
		out[1] *= v[1]
		out[2] *= v[2]
	}
	return out
}

// MulConst scales the vector by the given factors (one shared factor when a
// single value is given, matching the floatgeom calling convention).
func (p Point3) MulConst(fs ...float64) Point3 {
	out := p
	for _, f := range fs {
		out[0] *= f
		out[1] *= f
		out[2] *= f
	}
	return out
}

// Magnitude returns the vector's length.
func (p Point3) Magnitude() float64 {
	return math.Sqrt(p[0]*p[0] + p[1]*p[1] + p[2]*p[2])
}

// Normalize returns the unit vector; the zero vector maps to itself.
func (p Point3) Normalize() Point3 {
	m := p.Magnitude()
	if m == 0 {
		return p
	}
	return Point3{p[0] / m, p[1] / m, p[2] / m}
}

// Point4 is a 4D vector in quaternion layout: W is component 0, X/Y/Z are
// components 1/2/3.
type Point4 [4]float64

// W returns component 0.
func (p Point4) W() float64 { return p[0] }

// X returns component 1.
func (p Point4) X() float64 { return p[1] }

// Y returns component 2.
func (p Point4) Y() float64 { return p[2] }

// Z returns component 3.
func (p Point4) Z() float64 { return p[3] }

// Dim returns the i'th component.
func (p Point4) Dim(i int) float64 { return p[i] }
