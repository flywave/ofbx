package ofbx

import (
	"fmt"
	"math"

	"github.com/oakmound/oak/v2/alg/floatgeom"
)

// UpVector specifies which canonical axis represents up in the system (typically Y or Z).
type UpVector int

// UpVector Options
const (
	UpVectorX UpVector = 1
	UpVectorY UpVector = 2
	UpVectorZ UpVector = 3
)

// FrontVector is a vector with origin at the screen pointing toward the camera.
type FrontVector int

// FrontVector Parity Options
const (
	FrontVectorParityEven FrontVector = 1
	FrontVectorParityOdd  FrontVector = 2
)

// CoordSystem specifies the third vector of the system.
type CoordSystem int

// CoordSystem options
const (
	CoordSystemRight CoordSystem = iota
	CoordSystemLeft  CoordSystem = iota
)

// Matrix is a 16 sized slice that we operate on as if it was actually a matrix
type Matrix struct {
	m [16]float64 // last 4 are translation
}

func (mat *Matrix) ToArray() [16]float64 {
	return mat.m
}
func (m Matrix) Transposed() Matrix {
	return Matrix{m: [16]float64{
		m.m[0], m.m[4], m.m[8], m.m[12],
		m.m[1], m.m[5], m.m[9], m.m[13],
		m.m[2], m.m[6], m.m[10], m.m[14],
		m.m[3], m.m[7], m.m[11], m.m[15],
	}}
}
func matrixFromSlice(fs []float64) (Matrix, error) {
	if len(fs) != 16 {
		return Matrix{}, fmt.Errorf("expected 16 values, got %d", len(fs))
	}
	var a [16]float64
	copy(a[:], fs)
	return Matrix{a}, nil
}

// 添加缩放矩阵生成函数
func ScalingMatrix(scale floatgeom.Point3) Matrix {
	m := makeIdentity()
	m.m[0] = scale.X()
	m.m[5] = scale.Y()
	m.m[10] = scale.Z()
	return m
}

// 添加平移矩阵生成函数
func TranslationMatrix(t floatgeom.Point3) Matrix {
	m := makeIdentity()
	m.m[12] = t.X()
	m.m[13] = t.Y()
	m.m[14] = t.Z()
	return m
}

// 添加矩阵变换方法
func (m Matrix) MulPosition(v floatgeom.Point3) floatgeom.Point3 {
	// 齐次坐标变换 (x,y,z,1)
	x := v[0]*m.m[0] + v[1]*m.m[4] + v[2]*m.m[8] + m.m[12]
	y := v[0]*m.m[1] + v[1]*m.m[5] + v[2]*m.m[9] + m.m[13]
	z := v[0]*m.m[2] + v[1]*m.m[6] + v[2]*m.m[10] + m.m[14]
	return floatgeom.Point3{x, y, z}
}

func (m Matrix) MulDirection(v floatgeom.Point3) floatgeom.Point3 {
	// 方向变换忽略平移 (x,y,z,0)
	x := v[0]*m.m[0] + v[1]*m.m[4] + v[2]*m.m[8]
	y := v[0]*m.m[1] + v[1]*m.m[5] + v[2]*m.m[9]
	z := v[0]*m.m[2] + v[1]*m.m[6] + v[2]*m.m[10]
	return floatgeom.Point3{x, y, z}.Normalize()
}

func (m Matrix) RemoveScale() Matrix {
	// 提取旋转分量并重新标准化
	rot := Matrix{m.m}
	sx := math.Sqrt(m.m[0]*m.m[0] + m.m[1]*m.m[1] + m.m[2]*m.m[2])
	sy := math.Sqrt(m.m[4]*m.m[4] + m.m[5]*m.m[5] + m.m[6]*m.m[6])
	sz := math.Sqrt(m.m[8]*m.m[8] + m.m[9]*m.m[9] + m.m[10]*m.m[10])

	// 去除缩放因子
	for i := 0; i < 4; i++ {
		rot.m[i] /= sx
		rot.m[i+4] /= sy
		rot.m[i+8] /= sz
	}
	return rot
}

// Quat probably can bve removed
type Quat struct {
	X, Y, Z, W float64
}

// Mul multiplies the values of two matricies together and returns the output
func (m1 Matrix) Mul(m2 Matrix) Matrix {
	res := [16]float64{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			tmp := 0.0
			for k := 0; k < 4; k++ {
				tmp += m1.m[i+k*4] * m2.m[k+j*4]
			}
			res[i+j*4] = tmp
		}
	}
	return Matrix{res}
}

func (m Matrix) isZero() bool {
	for i := 0; i < 16; i++ {
		if m.m[i] != 0 {
			return false
		}
	}
	return true
}

func setTranslation(v floatgeom.Point3, m *Matrix) {
	m.m[12] = v.X()
	m.m[13] = v.Y()
	m.m[14] = v.Z()
}

func makeIdentity() Matrix {
	return Matrix{[16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}}
}

func RotationX(angle float64) Matrix {
	m2 := makeIdentity()
	//radian
	c := math.Cos(angle)
	s := math.Sin(angle)
	m2.m[5] = c
	m2.m[10] = c
	m2.m[9] = -s
	m2.m[6] = s
	return m2
}

func RotationY(angle float64) Matrix {
	m2 := makeIdentity()
	//radian
	c := math.Cos(angle)
	s := math.Sin(angle)
	m2.m[0] = c
	m2.m[10] = c
	m2.m[8] = s
	m2.m[2] = -s
	return m2
}

func RotationZ(angle float64) Matrix {
	m2 := makeIdentity()
	//radian
	c := math.Cos(angle)
	s := math.Sin(angle)
	m2.m[0] = c
	m2.m[5] = c
	m2.m[4] = -s
	m2.m[1] = s
	return m2
}

func getTriCountFromPoly(indices []int, idx int) (int, int) {
	count := 1
	for indices[idx+1+count] >= 0 {
		count++
	}
	return count, idx
}
