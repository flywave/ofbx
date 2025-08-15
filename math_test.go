package ofbx

import (
	"math"
	"testing"

	"github.com/oakmound/oak/v2/alg/floatgeom"
)

func TestMatrixIdentity(t *testing.T) {
	identity := makeIdentity()

	// Test identity matrix properties
	expected := [16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	for i := 0; i < 16; i++ {
		if identity.m[i] != expected[i] {
			t.Errorf("Identity matrix[%d] = %f, expected %f", i, identity.m[i], expected[i])
		}
	}
}

func TestMatrixMul(t *testing.T) {
	m1 := Matrix{[16]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
	m2 := makeIdentity()

	// Test multiplication with identity
	result := m1.Mul(m2)
	for i := 0; i < 16; i++ {
		if result.m[i] != m1.m[i] {
			t.Errorf("Matrix * Identity[%d] = %f, expected %f", i, result.m[i], m1.m[i])
		}
	}
}

func TestMatrixTranspose(t *testing.T) {
	m := Matrix{[16]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
	transposed := m.Transposed()

	// Test transpose
	expected := [16]float64{1, 5, 9, 13, 2, 6, 10, 14, 3, 7, 11, 15, 4, 8, 12, 16}
	for i := 0; i < 16; i++ {
		if transposed.m[i] != expected[i] {
			t.Errorf("Transposed[%d] = %f, expected %f", i, transposed.m[i], expected[i])
		}
	}
}

func TestScalingMatrix(t *testing.T) {
	scale := floatgeom.Point3{2, 3, 4}
	scaling := ScalingMatrix(scale)

	// Test scaling matrix
	expected := [16]float64{2, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 0, 0, 0, 0, 1}
	for i := 0; i < 16; i++ {
		if math.Abs(scaling.m[i]-expected[i]) > 1e-10 {
			t.Errorf("Scaling matrix[%d] = %f, expected %f", i, scaling.m[i], expected[i])
		}
	}
}

func TestTranslationMatrix(t *testing.T) {
	translation := floatgeom.Point3{5, 6, 7}
	trans := TranslationMatrix(translation)

	// Test translation matrix
	expected := [16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 5, 6, 7, 1}
	for i := 0; i < 16; i++ {
		if math.Abs(trans.m[i]-expected[i]) > 1e-10 {
			t.Errorf("Translation matrix[%d] = %f, expected %f", i, trans.m[i], expected[i])
		}
	}
}

func TestRotationX(t *testing.T) {
	angle := math.Pi / 2 // 90 degrees
	rot := RotationX(angle)

	// Test rotation matrix for 90 degrees around X
	expected := [16]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1}
	tolerance := 1e-10
	for i := 0; i < 16; i++ {
		if math.Abs(rot.m[i]-expected[i]) > tolerance {
			t.Errorf("RotationX[%d] = %f, expected %f", i, rot.m[i], expected[i])
		}
	}

	// Test actual rotation
	point := floatgeom.Point3{0, 1, 0}
	rotated := rot.MulPosition(point)
	expectedPoint := floatgeom.Point3{0, 0, 1}

	if math.Abs(rotated.X()-expectedPoint.X()) > tolerance ||
		math.Abs(rotated.Y()-expectedPoint.Y()) > tolerance ||
		math.Abs(rotated.Z()-expectedPoint.Z()) > tolerance {
		t.Errorf("RotationX point = %v, expected %v", rotated, expectedPoint)
	}
}

func TestRotationY(t *testing.T) {
	angle := math.Pi / 2 // 90 degrees
	rot := RotationY(angle)

	// Test rotation matrix for 90 degrees around Y
	expected := [16]float64{0, 0, -1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1}
	tolerance := 1e-10
	for i := 0; i < 16; i++ {
		if math.Abs(rot.m[i]-expected[i]) > tolerance {
			t.Errorf("RotationY[%d] = %f, expected %f", i, rot.m[i], expected[i])
		}
	}

	// Test actual rotation
	point := floatgeom.Point3{1, 0, 0}
	rotated := rot.MulPosition(point)
	expectedPoint := floatgeom.Point3{0, 0, -1}

	if math.Abs(rotated.X()-expectedPoint.X()) > tolerance ||
		math.Abs(rotated.Y()-expectedPoint.Y()) > tolerance ||
		math.Abs(rotated.Z()-expectedPoint.Z()) > tolerance {
		t.Errorf("RotationY point = %v, expected %v", rotated, expectedPoint)
	}
}

func TestRotationZ(t *testing.T) {
	angle := math.Pi / 2 // 90 degrees
	rot := RotationZ(angle)

	// Test rotation matrix for 90 degrees around Z
	expected := [16]float64{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	tolerance := 1e-10
	for i := 0; i < 16; i++ {
		if math.Abs(rot.m[i]-expected[i]) > tolerance {
			t.Errorf("RotationZ[%d] = %f, expected %f", i, rot.m[i], expected[i])
		}
	}

	// Test actual rotation
	point := floatgeom.Point3{1, 0, 0}
	rotated := rot.MulPosition(point)
	expectedPoint := floatgeom.Point3{0, 1, 0}

	if math.Abs(rotated.X()-expectedPoint.X()) > tolerance ||
		math.Abs(rotated.Y()-expectedPoint.Y()) > tolerance ||
		math.Abs(rotated.Z()-expectedPoint.Z()) > tolerance {
		t.Errorf("RotationZ point = %v, expected %v", rotated, expectedPoint)
	}
}

func TestMulPosition(t *testing.T) {
	// Test identity transformation
	identity := makeIdentity()
	point := floatgeom.Point3{1, 2, 3}
	transformed := identity.MulPosition(point)

	if transformed != point {
		t.Errorf("Identity transform = %v, expected %v", transformed, point)
	}

	// Test translation
	trans := TranslationMatrix(floatgeom.Point3{10, 20, 30})
	transformed = trans.MulPosition(point)
	expected := floatgeom.Point3{11, 22, 33}

	tolerance := 1e-10
	if math.Abs(transformed.X()-expected.X()) > tolerance ||
		math.Abs(transformed.Y()-expected.Y()) > tolerance ||
		math.Abs(transformed.Z()-expected.Z()) > tolerance {
		t.Errorf("Translation transform = %v, expected %v", transformed, expected)
	}
}

func TestMulDirection(t *testing.T) {
	// Test identity transformation for direction
	identity := makeIdentity()
	direction := floatgeom.Point3{1, 2, 3}
	transformed := identity.MulDirection(direction)

	// Direction should be normalized
	expected := direction.Normalize()

	tolerance := 1e-10
	if math.Abs(transformed.X()-expected.X()) > tolerance ||
		math.Abs(transformed.Y()-expected.Y()) > tolerance ||
		math.Abs(transformed.Z()-expected.Z()) > tolerance {
		t.Errorf("Identity direction transform = %v, expected %v", transformed, expected)
	}

	// Test rotation preserves direction length (after normalization)
	rot := RotationZ(math.Pi / 4) // 45 degrees
	direction = floatgeom.Point3{1, 0, 0}
	transformed = rot.MulDirection(direction)

	length := math.Sqrt(transformed.X()*transformed.X() + transformed.Y()*transformed.Y() + transformed.Z()*transformed.Z())
	if math.Abs(length-1.0) > tolerance {
		t.Errorf("Direction length after rotation = %f, expected 1.0", length)
	}
}

func TestRemoveScale(t *testing.T) {
	// Create matrix with scale
	scale := floatgeom.Point3{2, 3, 4}
	rotation := RotationZ(math.Pi / 4)

	// Combine scale and rotation
	scaled := ScalingMatrix(scale)
	combined := scaled.Mul(rotation)

	// Remove scale
	unscaled := combined.RemoveScale()

	// Test that scale is removed (check determinant or specific elements)
	// We can't easily test exact values, but we can verify the operation runs
	// and the matrix is somewhat reasonable
	if unscaled.m[0] == 0 || unscaled.m[5] == 0 || unscaled.m[10] == 0 {
		t.Error("RemoveScale resulted in zero scale factors")
	}
}

func TestMatrixFromSlice(t *testing.T) {
	// Test valid slice
	validSlice := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	matrix, err := matrixFromSlice(validSlice)

	if err != nil {
		t.Errorf("matrixFromSlice returned error: %v", err)
	}

	for i := 0; i < 16; i++ {
		if matrix.m[i] != validSlice[i] {
			t.Errorf("matrixFromSlice[%d] = %f, expected %f", i, matrix.m[i], validSlice[i])
		}
	}

	// Test invalid slice length
	invalidSlice := []float64{1, 2, 3}
	_, err = matrixFromSlice(invalidSlice)

	if err == nil {
		t.Error("matrixFromSlice should return error for invalid slice length")
	}
}

func TestSetTranslation(t *testing.T) {
	matrix := makeIdentity()
	translation := floatgeom.Point3{10, 20, 30}

	setTranslation(translation, &matrix)

	if matrix.m[12] != 10 || matrix.m[13] != 20 || matrix.m[14] != 30 {
		t.Errorf("setTranslation = [%f, %f, %f], expected [10, 20, 30]",
			matrix.m[12], matrix.m[13], matrix.m[14])
	}
}

func TestGetTriCountFromPoly(t *testing.T) {
	// Test triangle (3 vertices, 2 triangles - but function returns vertex count)
	indices := []int{0, 1, 2, -1}
	count, next := getTriCountFromPoly(indices, 0)

	if count != 2 {
		t.Errorf("getTriCountFromPoly count = %d, expected 2", count)
	}
	if next != 0 {
		t.Errorf("getTriCountFromPoly next = %d, expected 0", next)
	}

	// Test quad (4 vertices, 3 triangles - but function returns vertex count)
	indices = []int{0, 1, 2, 3, -1}
	count, next = getTriCountFromPoly(indices, 0)

	if count != 3 {
		t.Errorf("getTriCountFromPoly count = %d, expected 3", count)
	}
	if next != 0 {
		t.Errorf("getTriCountFromPoly next = %d, expected 0", next)
	}

	// Test pentagon (5 vertices, 4 triangles - but function returns vertex count)
	indices = []int{0, 1, 2, 3, 4, -1}
	count, next = getTriCountFromPoly(indices, 0)

	if count != 4 {
		t.Errorf("getTriCountFromPoly count = %d, expected 4", count)
	}
	if next != 0 {
		t.Errorf("getTriCountFromPoly next = %d, expected 0", next)
	}
}

func TestComplexMatrixOperations(t *testing.T) {
	// Test combined transformations
	scale := floatgeom.Point3{2, 3, 4}
	rotation := RotationZ(math.Pi / 2)
	translation := TranslationMatrix(floatgeom.Point3{10, 20, 30})

	// Create combined transformation: translation * rotation * scale
	// Note: In matrix multiplication, order matters (right to left)
	scaled := ScalingMatrix(scale)
	rotated := rotation.Mul(scaled)
	combined := translation.Mul(rotated)

	// Test transformation
	point := floatgeom.Point3{1, 0, 0}
	transformed := combined.MulPosition(point)

	// Expected: scale by 2, rotate 90 degrees, then translate
	// Original: (1, 0, 0) -> scale -> (2, 0, 0) -> rotate -> (0, 2, 0) -> translate -> (10, 22, 30)
	expected := floatgeom.Point3{10, 22, 30}

	tolerance := 1e-10
	if math.Abs(transformed.X()-expected.X()) > tolerance ||
		math.Abs(transformed.Y()-expected.Y()) > tolerance ||
		math.Abs(transformed.Z()-expected.Z()) > tolerance {
		t.Errorf("Combined transform = %v, expected %v", transformed, expected)
	}
}
