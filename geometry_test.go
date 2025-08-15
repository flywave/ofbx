package ofbx

import (
	"testing"

	"github.com/oakmound/oak/v2/alg/floatgeom"
)

func TestNewGeometry(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geometry")}

	geom := NewGeometry(scene, element)

	if geom == nil {
		t.Fatal("NewGeometry should not return nil")
	}

	if geom.Type() != GEOMETRY {
		t.Errorf("Expected geometry type to be GEOMETRY, got %v", geom.Type())
	}

	if geom.ID() != 0 {
		t.Errorf("Expected geometry ID to be 0, got %d", geom.ID())
	}

	if len(geom.Vertices) != 0 {
		t.Errorf("Expected empty vertices slice, got length %d", len(geom.Vertices))
	}

	if len(geom.Normals) != 0 {
		t.Errorf("Expected empty normals slice, got length %d", len(geom.Normals))
	}

	if len(geom.Tangents) != 0 {
		t.Errorf("Expected empty tangents slice, got length %d", len(geom.Tangents))
	}

	if len(geom.oldVerts) != 0 {
		t.Errorf("Expected empty oldVerts slice, got length %d", len(geom.oldVerts))
	}
}

func TestGeometryString(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test empty geometry string
	str := geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty")
	}

	// Test with vertices
	geom.Vertices = []floatgeom.Point3{
		{1, 2, 3},
		{4, 5, 6},
	}

	str = geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with vertices")
	}

	// Test with normals
	geom.Normals = []floatgeom.Point3{
		{0, 0, 1},
		{0, 1, 0},
	}

	str = geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with normals")
	}

	// Test with faces
	geom.Faces = [][]int{
		{0, 1, 2},
		{2, 1, 3},
	}

	str = geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with faces")
	}
}

func TestGeometryTriangulate(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test quad (4 vertices) - should produce 3 triangles (fan triangulation)
	indices := []int{0, 1, 2, 3, -1}
	old := geom.triangulate(indices)

	// Should have 9 indices (3 triangles * 3 vertices for quad)
	if len(old) != 9 {
		t.Errorf("Expected 9 indices for quad triangulation, got %d", len(old))
	}

	// Test triangle (3 vertices) - should produce 2 triangles
	indices = []int{0, 1, 2, -1}
	old = geom.triangulate(indices)

	// Should have 6 indices (2 triangles * 3 vertices)
	if len(old) != 6 {
		t.Errorf("Expected 6 indices for triangle triangulation, got %d", len(old))
	}

	// Test empty indices
	indices = []int{}
	old = geom.triangulate(indices)
	if len(old) != 0 {
		t.Errorf("Expected 0 indices for empty input, got %d", len(old))
	}
}

func TestGeometryApplyMatrix(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Set up test vertices
	geom.Vertices = []floatgeom.Point3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}

	// Set up test normals
	geom.Normals = []floatgeom.Point3{
		{1, 0, 0},
		{0, 1, 0},
	}

	// Create a simple rotation matrix (90 degrees around Z axis)
	m := &Matrix{
		m: [16]float64{
			0, -1, 0, 0,
			1, 0, 0, 0,
			0, 0, 1, 0,
			0, 0, 0, 1,
		},
	}

	// Apply matrix
	geom.applyMatrix(m)

	// Check if vertices were transformed correctly
	expectedVertices := []floatgeom.Point3{
		{0, -1, 0}, // (1,0,0) -> (0,-1,0) - 90度逆时针旋转
		{1, 0, 0},  // (0,1,0) -> (1,0,0)
		{0, 0, 1},  // (0,0,1) -> (0,0,1)
	}

	for i, v := range geom.Vertices {
		if v != expectedVertices[i] {
			t.Errorf("Vertex %d: expected %+v, got %+v", i, expectedVertices[i], v)
		}
	}

	// Check if normals were transformed correctly
	expectedNormals := []floatgeom.Point3{
		{0, -1, 0}, // (1,0,0) -> (0,-1,0)
		{1, 0, 0},  // (0,1,0) -> (1,0,0)
	}

	for i, n := range geom.Normals {
		if n != expectedNormals[i] {
			t.Errorf("Normal %d: expected %+v, got %+v", i, expectedNormals[i], n)
		}
	}
}

func TestVertexAdd(t *testing.T) {
	v := &Vertex{index: -1, next: nil}

	// First addition
	v.add(5)
	if v.index != 5 {
		t.Errorf("Expected vertex index to be 5, got %d", v.index)
	}

	// Second addition should create next vertex
	v.add(10)
	if v.next == nil {
		t.Fatal("Expected next vertex to be created")
	}
	if v.next.index != 10 {
		t.Errorf("Expected next vertex index to be 10, got %d", v.next.index)
	}

	// Third addition should go to the end
	v.add(15)
	if v.next.next == nil {
		t.Fatal("Expected third vertex to be created")
	}
	if v.next.next.index != 15 {
		t.Errorf("Expected third vertex index to be 15, got %d", v.next.next.index)
	}
}

func TestGeometryWithColors(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test with colors
	geom.Colors = []floatgeom.Point4{
		{1, 0, 0, 1},
		{0, 1, 0, 1},
	}

	str := geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with colors")
	}
}

func TestGeometryWithUVs(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test with UVs
	geom.UVs[0] = []floatgeom.Point2{
		{0, 0},
		{1, 1},
	}

	str := geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with UVs")
	}
}

func TestGeometryWithSkin(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test with skin
	skin := &Skin{Object: *NewObject(scene, &Element{ID: NewDataView("test_skin")})}
	geom.Skin = skin

	str := geom.String()
	if str == "" {
		t.Error("Geometry string should not be empty with skin")
	}
}

func TestGeometryStringPrefix(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Test stringPrefix with custom prefix
	geom.Vertices = []floatgeom.Point3{
		{1, 2, 3},
	}

	str := geom.stringPrefix("  ")
	if str == "" {
		t.Error("Geometry stringPrefix should not be empty")
	}

	// Check if prefix is applied
	if str[:2] != "  " {
		t.Error("Prefix should be applied to the output")
	}
}

func TestGeometryGetOldVerts(t *testing.T) {
	scene := &Scene{}
	element := &Element{ID: NewDataView("test_geom")}
	geom := NewGeometry(scene, element)

	// Initially should be empty
	oldVerts := geom.GetOldVerts()
	if oldVerts == nil {
		t.Error("GetOldVerts should not return nil")
	}
	if len(oldVerts) != 0 {
		t.Errorf("Expected empty oldVerts, got length %d", len(oldVerts))
	}

	// After triangulation, should have data
	indices := []int{0, 1, 2, -1}
	geom.triangulate(indices)
	oldVerts = geom.GetOldVerts()
	if len(oldVerts) == 0 {
		t.Error("Expected oldVerts to have data after triangulation")
	}
}
