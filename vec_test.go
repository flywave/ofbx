package ofbx

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPoint3Arithmetic covers the vector math used by the library and tests.
func TestPoint3Arithmetic(t *testing.T) {
	a := Point3{1, 2, 3}
	b := Point3{0.5, 0.5, 0.5}

	assert.Equal(t, Point3{1.5, 2.5, 3.5}, a.Add(b))
	assert.Equal(t, Point3{0.5, 1.5, 2.5}, a.Sub(b))
	assert.Equal(t, Point3{0.5, 1, 1.5}, a.Mul(b))
	assert.Equal(t, Point3{2, 4, 6}, a.MulConst(2))
	assert.InDelta(t, math.Sqrt(14), a.Magnitude(), 1e-12)
	assert.Equal(t, Point3{0, 0, 1}, Point3{0, 0, 42}.Normalize())
	assert.Equal(t, Point3{}, Point3{}.Normalize()) // zero vector stays zero
	assert.Equal(t, 3.0, Point3{3, 0, 0}.X())
}

// TestPoint4AxisOrder pins the floatgeom-compatible quaternion layout:
// W is component 0.
func TestPoint4AxisOrder(t *testing.T) {
	p := Point4{10, 20, 30, 40}
	assert.Equal(t, 10.0, p.W())
	assert.Equal(t, 20.0, p.X())
	assert.Equal(t, 30.0, p.Y())
	assert.Equal(t, 40.0, p.Z())
	assert.Equal(t, 30.0, p.Dim(2))
}

// TestPoint2Accessors covers Point2.
func TestPoint2Accessors(t *testing.T) {
	p := Point2{1.5, -2.5}
	assert.Equal(t, 1.5, p.X())
	assert.Equal(t, -2.5, p.Y())
}
