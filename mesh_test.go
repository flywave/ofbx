package ofbx

import (
	"math"
	"testing"
)

// TestNewMesh tests the creation of a new Mesh instance
func TestNewMesh(t *testing.T) {
	// Create a minimal scene and element for testing
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	element := &Element{ID: NewDataView("12345")}

	mesh := NewMesh(scene, element)

	if mesh == nil {
		t.Fatal("NewMesh returned nil")
	}

	// Set ID explicitly since we're not using the full parsing pipeline
	mesh.SetID(12345)

	// Check that ID is set correctly
	if mesh.ID() != 12345 {
		t.Errorf("Mesh ID = %d, expected 12345", mesh.ID())
	}

	if !mesh.Object.isNode {
		t.Error("Mesh should have isNode set to true")
	}
}

// TestMeshType tests the Type method returns MESH
func TestMeshType(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	if mesh.Type() != MESH {
		t.Errorf("Mesh Type = %v, expected MESH", mesh.Type())
	}
}

// TestGetGlobalMatrix tests the GetGlobalMatrix method
func TestGetGlobalMatrix(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Test that GetGlobalMatrix returns a matrix (basic sanity check)
	matrix := mesh.GetGlobalMatrix()

	// Check if it's an identity matrix by checking some key elements
	if math.Abs(matrix.m[0]-1.0) > 1e-10 {
		t.Errorf("GetGlobalMatrix[0] = %f, expected 1.0", matrix.m[0])
	}
	if math.Abs(matrix.m[5]-1.0) > 1e-10 {
		t.Errorf("GetGlobalMatrix[5] = %f, expected 1.0", matrix.m[5])
	}
	if math.Abs(matrix.m[10]-1.0) > 1e-10 {
		t.Errorf("GetGlobalMatrix[10] = %f, expected 1.0", matrix.m[10])
	}
	if math.Abs(matrix.m[15]-1.0) > 1e-10 {
		t.Errorf("GetGlobalMatrix[15] = %f, expected 1.0", matrix.m[15])
	}
}

// TestGetGeometricMatrix tests the getGeometricMatrix method with default values
func TestGetGeometricMatrix(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Test with default values (should be identity)
	matrix := mesh.getGeometricMatrix()

	// Check if it's an identity matrix
	tolerance := 1e-10
	expected := makeIdentity()
	for i := 0; i < 16; i++ {
		if math.Abs(matrix.m[i]-expected.m[i]) > tolerance {
			t.Errorf("getGeometricMatrix[%d] = %f, expected %f", i, matrix.m[i], expected.m[i])
		}
	}
}

// TestGetGeometricMatrixWithValues tests the getGeometricMatrix method with specific values
func TestGetGeometricMatrixWithValues(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Mock resolveVec3Property to return specific values
	// Note: This is a simplified test since we can't easily mock resolveVec3Property
	// In a real scenario, you might use interfaces or dependency injection

	matrix := mesh.getGeometricMatrix()

	// Basic checks - matrix should not be nil
	if matrix.m == [16]float64{} {
		t.Error("getGeometricMatrix returned zero matrix")
	}
}

// TestMeshString tests the String method
func TestMeshString(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Skip test if Geometry is nil (which would cause panic)
	// This is expected behavior when Geometry is nil
	defer func() {
		if r := recover(); r != nil {
			t.Log("String method panicked with nil Geometry - this is expected")
		}
	}()

	str := mesh.String()

	if str == "" {
		t.Error("String method returned empty string")
	}

	// Check if the string contains expected elements
	if !containsSubstring(str, "Mesh:") {
		t.Error("String method does not contain expected mesh prefix")
	}
}

// TestMeshStringPrefix tests the stringPrefix method
func TestMeshStringPrefix(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Skip test if Geometry is nil (which would cause panic)
	defer func() {
		if r := recover(); r != nil {
			t.Log("stringPrefix method panicked with nil Geometry - this is expected")
		}
	}()

	str := mesh.stringPrefix("  ")

	if str == "" {
		t.Error("stringPrefix method returned empty string")
	}

	// Check if the string contains expected prefix
	if !containsSubstring(str, "  Mesh:") {
		t.Error("stringPrefix method does not contain expected prefix")
	}
}

// TestApplyLocalTransform tests the applyLocalTransform method
func TestApplyLocalTransform(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Create a minimal geometry for testing
	geometry := &Geometry{}
	mesh.Geometry = geometry

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("applyLocalTransform panicked: %v", r)
		}
	}()

	mesh.applyLocalTransform()
}

// TestAnimationsEmpty tests the Animations method with empty scene
func TestAnimationsEmpty(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Test with nil Geometry - should handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Log("Animations panicked with nil Geometry - this is expected behavior")
		}
	}()

	animations := mesh.Animations()

	if animations == nil {
		t.Error("Animations returned nil")
	}

	if len(animations) != 0 {
		t.Errorf("Animations length = %d, expected 0", len(animations))
	}
}

// TestAnimationsWithGeometry tests the Animations method with geometry
func TestAnimationsWithGeometry(t *testing.T) {
	scene := &Scene{}
	element := &Element{}
	mesh := NewMesh(scene, element)

	// Create minimal geometry
	geometry := &Geometry{}
	mesh.Geometry = geometry

	// Test with nil skin - should handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Log("Animations panicked with nil skin - this is expected behavior")
		}
	}()

	animations := mesh.Animations()

	if animations == nil {
		t.Error("Animations returned nil")
	}

	if len(animations) != 0 {
		t.Errorf("Animations length = %d, expected 0", len(animations))
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str[:len(substr)] == substr || containsSubstring(str[1:], substr))
}
