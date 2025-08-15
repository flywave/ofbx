package ofbx

import (
	"testing"

	"github.com/oakmound/oak/v2/alg/floatgeom"
	"github.com/stretchr/testify/assert"
)

func TestRotationMatrix(t *testing.T) {
	tests := []struct {
		name  string
		order RotationOrder
		euler floatgeom.Point3
	}{
		{
			name:  "EulerXYZ - Zero rotation",
			order: EulerXYZ,
			euler: floatgeom.Point3{0, 0, 0},
		},
		{
			name:  "EulerXYZ - 90 degrees X",
			order: EulerXYZ,
			euler: floatgeom.Point3{90, 0, 0},
		},
		{
			name:  "EulerXYZ - Combined rotation",
			order: EulerXYZ,
			euler: floatgeom.Point3{45, 30, 60},
		},
		{
			name:  "EulerZYX - Combined rotation",
			order: EulerZYX,
			euler: floatgeom.Point3{45, 30, 60},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.rotationMatrix(tt.euler)
			assert.NotNil(t, result)
			
			// 验证是有效的旋转矩阵（行列式接近1）
			det := result.m[0]*(result.m[5]*result.m[10]-result.m[6]*result.m[9]) -
				result.m[1]*(result.m[4]*result.m[10]-result.m[6]*result.m[8]) +
				result.m[2]*(result.m[4]*result.m[9]-result.m[5]*result.m[8])
			assert.InDelta(t, 1.0, det, 0.001, "Rotation matrix should have determinant 1")
		})
	}
}

func TestRotationMatrixAllOrders(t *testing.T) {
	euler := floatgeom.Point3{30, 45, 60}

	orders := []RotationOrder{
		EulerXYZ, EulerXZY, EulerYZX, EulerYXZ, EulerZXY, EulerZYX, SphericXYZ,
	}

	for _, order := range orders {
		t.Run(order.String(), func(t *testing.T) {
			// 主要测试不panic
			assert.NotPanics(t, func() {
				matrix := order.rotationMatrix(euler)
				assert.NotNil(t, matrix)
			})
		})
	}
}

func TestRotationOrderString(t *testing.T) {
	orders := []RotationOrder{
		EulerXYZ, EulerXZY, EulerYZX, EulerYXZ, EulerZXY, EulerZYX, SphericXYZ,
	}

	expectedStrings := []string{
		"EulerXYZ", "EulerXZY", "EulerYZX", "EulerYXZ", "EulerZXY", "EulerZYX", "SphericXYZ",
	}

	for i, order := range orders {
		assert.Equal(t, expectedStrings[i], order.String())
	}
}

func TestRotationMatrixSphericXYZ(t *testing.T) {
	// SphericXYZ目前不支持，应该被当作EulerXYZ处理
	order := SphericXYZ
	euler := floatgeom.Point3{45, 30, 60}

	matrix := order.rotationMatrix(euler)
	expected := EulerXYZ.rotationMatrix(euler)

	for i := 0; i < 16; i++ {
		assert.InDelta(t, expected.m[i], matrix.m[i], 0.0001, "SphericXYZ should behave like EulerXYZ")
	}
}
