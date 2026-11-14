package ofbx

import (
	"testing"
)

func TestResolveEnumProperty(t *testing.T) {
	tests := []struct {
		name     string
		object   Obj
		propName string
		default_ int
		expected int
	}{
		{
			name:     "nonexistent property",
			object:   &Object{element: &Element{}},
			propName: "nonexistent",
			default_: 100,
			expected: 100,
		},
		{
			name:     "valid property with insufficient properties",
			object:   &Object{element: &Element{}},
			propName: "test",
			default_: 42,
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEnumProperty(tt.object, tt.propName, tt.default_)
			if result != tt.expected {
				t.Errorf("resolveEnumProperty() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestResolveVec3Property(t *testing.T) {
	tests := []struct {
		name     string
		object   Obj
		propName string
		default_ Point3
		expected Point3
	}{
		{
			name:     "nonexistent property",
			object:   &Object{element: &Element{}},
			propName: "nonexistent",
			default_: Point3{4, 5, 6},
			expected: Point3{4, 5, 6},
		},
		{
			name:     "insufficient properties",
			object:   &Object{element: &Element{}},
			propName: "test",
			default_: Point3{0, 0, 0},
			expected: Point3{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveVec3Property(tt.object, tt.propName, tt.default_)
			if result != tt.expected {
				t.Errorf("resolveVec3Property() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSplatVec2(t *testing.T) {
	tests := []struct {
		name        string
		mapping     VertexDataMapping
		data        []Point2
		indices     []int
		origIndices []int
		expected    []Point2
	}{
		{
			name:     "ByPolygonVertex - no indices",
			mapping:  ByPolygonVertex,
			data:     []Point2{{1, 2}, {3, 4}},
			indices:  []int{},
			expected: []Point2{{1, 2}, {3, 4}},
		},
		{
			name:     "ByPolygonVertex - with indices",
			mapping:  ByPolygonVertex,
			data:     []Point2{{1, 2}, {3, 4}},
			indices:  []int{0, 1, 0},
			expected: []Point2{{1, 2}, {3, 4}, {1, 2}},
		},
		{
			name:        "ByVertex - with origIndices",
			mapping:     ByVertex,
			data:        []Point2{{1, 2}, {3, 4}},
			origIndices: []int{0, 1, 0},
			expected:    []Point2{{1, 2}, {3, 4}, {1, 2}},
		},
		{
			name:     "ByPolygon - no indices",
			mapping:  ByPolygon,
			data:     []Point2{{1, 2}, {3, 4}},
			indices:  []int{},
			expected: []Point2{{1, 2}, {3, 4}},
		},
		{
			name:     "ByPolygon - with indices",
			mapping:  ByPolygon,
			data:     []Point2{{1, 2}, {3, 4}},
			indices:  []int{1, 0},
			expected: []Point2{{3, 4}, {1, 2}},
		},
		{
			name:        "ByVertex - negative indices",
			mapping:     ByVertex,
			data:        []Point2{{1, 2}, {3, 4}},
			origIndices: []int{-1, -2},
			expected:    []Point2{{1, 2}, {3, 4}},
		},
		{
			name:        "ByVertex - out of bounds indices",
			mapping:     ByVertex,
			data:        []Point2{{1, 2}},
			origIndices: []int{0, 1, 2},
			expected:    []Point2{{1, 2}, {0, 0}, {0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splatVec2(tt.mapping, tt.data, tt.indices, tt.origIndices)
			if len(result) != len(tt.expected) {
				t.Fatalf("splatVec2() length = %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splatVec2()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSplatVec3(t *testing.T) {
	tests := []struct {
		name        string
		mapping     VertexDataMapping
		data        []Point3
		indices     []int
		origIndices []int
		expected    []Point3
	}{
		{
			name:     "ByPolygonVertex - no indices",
			mapping:  ByPolygonVertex,
			data:     []Point3{{1, 2, 3}, {4, 5, 6}},
			indices:  []int{},
			expected: []Point3{{1, 2, 3}, {4, 5, 6}},
		},
		{
			name:     "ByPolygonVertex - with indices",
			mapping:  ByPolygonVertex,
			data:     []Point3{{1, 2, 3}, {4, 5, 6}},
			indices:  []int{0, 1, 0},
			expected: []Point3{{1, 2, 3}, {4, 5, 6}, {1, 2, 3}},
		},
		{
			name:        "ByVertex - with origIndices",
			mapping:     ByVertex,
			data:        []Point3{{1, 2, 3}, {4, 5, 6}},
			origIndices: []int{0, 1, 0},
			expected:    []Point3{{1, 2, 3}, {4, 5, 6}, {1, 2, 3}},
		},
		{
			name:        "ByVertex - negative indices",
			mapping:     ByVertex,
			data:        []Point3{{1, 2, 3}, {4, 5, 6}},
			origIndices: []int{-1, -2},
			expected:    []Point3{{1, 2, 3}, {4, 5, 6}},
		},
		{
			name:        "ByVertex - out of bounds indices",
			mapping:     ByVertex,
			data:        []Point3{{1, 2, 3}},
			origIndices: []int{0, 1, 2},
			expected:    []Point3{{1, 2, 3}, {0, 0, 0}, {0, 0, 0}},
		},
		{
			name:     "ByPolygon - no indices",
			mapping:  ByPolygon,
			data:     []Point3{{1, 2, 3}},
			indices:  []int{},
			expected: []Point3{{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.mapping != ByPolygon {
					t.Errorf("splatVec3() panicked: %v", r)
				}
			}()

			result := splatVec3(tt.mapping, tt.data, tt.indices, tt.origIndices)
			if len(result) != len(tt.expected) {
				t.Fatalf("splatVec3() length = %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splatVec3()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSplatVec4(t *testing.T) {
	tests := []struct {
		name        string
		mapping     VertexDataMapping
		data        []Point4
		indices     []int
		origIndices []int
		expected    []Point4
	}{
		{
			name:     "ByPolygonVertex - no indices",
			mapping:  ByPolygonVertex,
			data:     []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
			indices:  []int{},
			expected: []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
		},
		{
			name:     "ByPolygonVertex - with indices",
			mapping:  ByPolygonVertex,
			data:     []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
			indices:  []int{0, 1, 0},
			expected: []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}, {1, 2, 3, 4}},
		},
		{
			name:        "ByVertex - with origIndices",
			mapping:     ByVertex,
			data:        []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
			origIndices: []int{0, 1, 0},
			expected:    []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}, {1, 2, 3, 4}},
		},
		{
			name:        "ByVertex - negative indices",
			mapping:     ByVertex,
			data:        []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
			origIndices: []int{-1, -2},
			expected:    []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
		},
		{
			name:        "ByVertex - out of bounds indices",
			mapping:     ByVertex,
			data:        []Point4{{1, 2, 3, 4}},
			origIndices: []int{0, 1, 2},
			expected:    []Point4{{1, 2, 3, 4}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("splatVec4() panicked: %v", r)
				}
			}()

			result := splatVec4(tt.mapping, tt.data, tt.indices, tt.origIndices)
			if len(result) != len(tt.expected) {
				t.Fatalf("splatVec4() length = %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splatVec4()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRemapVec2(t *testing.T) {
	tests := []struct {
		name     string
		input    []Point2
		mapping  []int
		expected []Point2
	}{
		{
			name:     "empty input",
			input:    []Point2{},
			mapping:  []int{0, 1, 2},
			expected: []Point2{},
		},
		{
			name:     "basic remap",
			input:    []Point2{{1, 2}, {3, 4}},
			mapping:  []int{1, 0, 1},
			expected: []Point2{{1, 2}, {3, 4}, {3, 4}, {1, 2}, {3, 4}},
		},
		{
			name:     "out of bounds mapping",
			input:    []Point2{{1, 2}},
			mapping:  []int{0, 1, 2},
			expected: []Point2{{1, 2}, {1, 2}, {0, 0}, {0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := make([]Point2, len(tt.input))
			copy(out, tt.input)

			RemapVec2(&out, tt.mapping)

			if len(out) != len(tt.expected) {
				t.Fatalf("RemapVec2() length = %d, want %d", len(out), len(tt.expected))
			}
			for i := range out {
				if out[i] != tt.expected[i] {
					t.Errorf("RemapVec2()[%d] = %v, want %v", i, out[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRemapVec3(t *testing.T) {
	tests := []struct {
		name     string
		input    []Point3
		mapping  []int
		expected []Point3
	}{
		{
			name:     "empty input",
			input:    []Point3{},
			mapping:  []int{0, 1, 2},
			expected: []Point3{},
		},
		{
			name:     "basic remap",
			input:    []Point3{{1, 2, 3}, {4, 5, 6}},
			mapping:  []int{1, 0, 1},
			expected: []Point3{{1, 2, 3}, {4, 5, 6}, {4, 5, 6}, {1, 2, 3}, {4, 5, 6}},
		},
		{
			name:     "out of bounds mapping",
			input:    []Point3{{1, 2, 3}},
			mapping:  []int{0, 1, 2},
			expected: []Point3{{1, 2, 3}, {1, 2, 3}, {0, 0, 0}, {0, 0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := make([]Point3, len(tt.input))
			copy(out, tt.input)

			RemapVec3(&out, tt.mapping)

			if len(out) != len(tt.expected) {
				t.Fatalf("RemapVec3() length = %d, want %d", len(out), len(tt.expected))
			}
			for i := range out {
				if out[i] != tt.expected[i] {
					t.Errorf("RemapVec3()[%d] = %v, want %v", i, out[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRemapVec4(t *testing.T) {
	tests := []struct {
		name     string
		input    []Point4
		mapping  []int
		expected []Point4
	}{
		{
			name:     "empty input",
			input:    []Point4{},
			mapping:  []int{0, 1, 2},
			expected: []Point4{},
		},
		{
			name:     "basic remap",
			input:    []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}},
			mapping:  []int{1, 0, 1},
			expected: []Point4{{1, 2, 3, 4}, {5, 6, 7, 8}, {5, 6, 7, 8}, {1, 2, 3, 4}, {5, 6, 7, 8}},
		},
		{
			name:     "out of bounds mapping",
			input:    []Point4{{1, 2, 3, 4}},
			mapping:  []int{0, 1, 2},
			expected: []Point4{{1, 2, 3, 4}, {1, 2, 3, 4}, {0, 0, 0, 0}, {0, 0, 0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := make([]Point4, len(tt.input))
			copy(out, tt.input)

			RemapVec4(&out, tt.mapping)

			if len(out) != len(tt.expected) {
				t.Fatalf("RemapVec4() length = %d, want %d", len(out), len(tt.expected))
			}
			for i := range out {
				if out[i] != tt.expected[i] {
					t.Errorf("RemapVec4()[%d] = %v, want %v", i, out[i], tt.expected[i])
				}
			}
		})
	}
}
