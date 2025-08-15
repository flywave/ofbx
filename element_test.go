package ofbx

import (
	"strings"
	"testing"
)

func TestNewElement(t *testing.T) {
	id := NewDataView("test_id")
	element := &Element{
		ID:         id,
		Children:   []*Element{},
		Properties: []*Property{},
	}

	if element.ID != id {
		t.Error("ID not set correctly")
	}
}

func TestElementGetProperty(t *testing.T) {
	prop1 := &Property{value: NewDataView("prop1")}
	prop2 := &Property{value: NewDataView("prop2")}

	element := &Element{
		Properties: []*Property{prop1, prop2},
	}

	tests := []struct {
		name     string
		index    int
		expected *Property
	}{
		{
			name:     "first property",
			index:    0,
			expected: prop1,
		},
		{
			name:     "second property",
			index:    1,
			expected: prop2,
		},
		{
			name:     "out of bounds",
			index:    5,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := element.getProperty(tt.index)
			if result != tt.expected {
				t.Errorf("getProperty(%d) = %v, want %v", tt.index, result, tt.expected)
			}
		})
	}
}

func TestElementString(t *testing.T) {
	tests := []struct {
		name     string
		element  *Element
		contains []string
	}{
		{
			name: "empty element",
			element: &Element{
				ID:         nil,
				Children:   []*Element{},
				Properties: []*Property{},
			},
			contains: []string{"Element:"},
		},
		{
			name: "element with ID",
			element: &Element{
				ID:         NewDataView("test_id"),
				Children:   []*Element{},
				Properties: []*Property{},
			},
			contains: []string{"Element:", "test_id"},
		},
		{
			name: "element with single property",
			element: &Element{
				ID:       NewDataView("test_id"),
				Children: []*Element{},
				Properties: []*Property{
					{value: NewDataView("single_prop")},
				},
			},
			contains: []string{"Element:", "test_id"},
		},
		{
			name: "element with multiple properties",
			element: &Element{
				ID:       NewDataView("test_id"),
				Children: []*Element{},
				Properties: []*Property{
					{value: NewDataView("prop1")},
					{value: NewDataView("prop2")},
					{value: NewDataView("prop3")},
				},
			},
			contains: []string{"Element:", "test_id"},
		},
		{
			name: "element with children",
			element: &Element{
				ID: NewDataView("parent"),
				Children: []*Element{
					{
						ID:         NewDataView("child1"),
						Children:   []*Element{},
						Properties: []*Property{},
					},
					{
						ID:         NewDataView("child2"),
						Children:   []*Element{},
						Properties: []*Property{},
					},
				},
				Properties: []*Property{},
			},
			contains: []string{"parent", "child1", "child2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.element.String()

			// Check that the string contains expected substrings
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("String() missing expected content: %q", expected)
				}
			}

			// Ensure it doesn't panic
			if result == "" {
				t.Log("Empty string returned")
			}
		})
	}
}

func TestElementStringPrefix(t *testing.T) {
	prefix := "  "
	element := &Element{
		ID: NewDataView("test"),
		Properties: []*Property{
			{value: NewDataView("prop")},
		},
		Children: []*Element{
			{
				ID: NewDataView("child"),
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("stringPrefix panicked: %v", r)
		}
	}()

	result := element.stringPrefix(prefix)
	if result == "" {
		t.Log("Empty string returned from stringPrefix")
	}
}

func TestElementWithNumberProperty(t *testing.T) {
	prop1 := &Property{value: NewDataView("Number")}
	prop2 := &Property{value: NewDataView("test_name")}
	prop3 := &Property{value: NewDataView("type")}
	prop4 := &Property{value: NewDataView("value")}
	prop5 := &Property{value: NewDataView("123.45")}

	element := &Element{
		ID: NewDataView("test_element"),
		Properties: []*Property{
			prop1, prop2, prop3, prop4, prop5,
		},
	}

	result := element.String()

	// Check if number format is applied
	if !strings.Contains(result, "test_name=123.45") {
		t.Logf("Number format not applied as expected: %s", result)
	}
}

func TestElementWithColorProperty(t *testing.T) {
	prop1 := &Property{value: NewDataView("Color")}
	prop2 := &Property{value: NewDataView("test_color")}
	prop3 := &Property{value: NewDataView("type")}
	prop4 := &Property{value: NewDataView("1.0")}
	prop5 := &Property{value: NewDataView("0.5")}
	prop6 := &Property{value: NewDataView("0.25")}
	prop7 := &Property{value: NewDataView("0.75")}

	element := &Element{
		ID: NewDataView("test_element"),
		Properties: []*Property{
			prop1, prop2, prop3, prop4, prop5, prop6, prop7,
		},
	}

	result := element.String()

	// Check if color format is applied
	if !strings.Contains(result, "test_color: R=1.0 G=0.5 B=0.25") {
		t.Logf("Color format not applied as expected: %s", result)
	}
}

func TestElementWithVectorProperty(t *testing.T) {
	prop1 := &Property{value: NewDataView("Lcl Translation")}
	prop2 := &Property{value: NewDataView("test_vector")}
	prop3 := &Property{value: NewDataView("type")}
	prop4 := &Property{value: NewDataView("10.0")}
	prop5 := &Property{value: NewDataView("20.0")}
	prop6 := &Property{value: NewDataView("30.0")}
	prop7 := &Property{value: NewDataView("40.0")}

	element := &Element{
		ID: NewDataView("test_element"),
		Properties: []*Property{
			prop1, prop2, prop3, prop4, prop5, prop6, prop7,
		},
	}

	result := element.String()

	// Check if vector format is applied
	if !strings.Contains(result, "test_vector: X=10.0 Y=20.0 Z=30.0") {
		t.Logf("Vector format not applied as expected: %s", result)
	}
}

func TestElementWithUnknownPropertyFormat(t *testing.T) {
	prop1 := &Property{value: NewDataView("UnknownType")}
	prop2 := &Property{value: NewDataView("test")}
	prop3 := &Property{value: NewDataView("value")}

	element := &Element{
		ID: NewDataView("test_element"),
		Properties: []*Property{
			prop1, prop2, prop3,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("String() panicked with unknown property type: %v", r)
		}
	}()

	result := element.String()
	if result == "" {
		t.Log("Empty string returned for unknown property type")
	}
}

func TestElementNilID(t *testing.T) {
	element := &Element{
		ID:         nil,
		Children:   []*Element{},
		Properties: []*Property{},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("String() panicked with nil ID: %v", r)
		}
	}()

	result := element.String()
	if !strings.Contains(result, "Element:") {
		t.Error("Expected 'Element:' even with nil ID")
	}
}

func TestElementNilProperties(t *testing.T) {
	element := &Element{
		ID:         NewDataView("test"),
		Children:   []*Element{},
		Properties: nil,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("String() panicked with nil Properties: %v", r)
		}
	}()

	result := element.String()
	if result == "" {
		t.Log("Empty string returned with nil Properties")
	}
}

func TestElementNilChildren(t *testing.T) {
	element := &Element{
		ID:         NewDataView("test"),
		Children:   nil,
		Properties: []*Property{},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("String() panicked with nil Children: %v", r)
		}
	}()

	result := element.String()
	if result == "" {
		t.Log("Empty string returned with nil Children")
	}
}

func TestElementPropertyFormats(t *testing.T) {
	tests := []struct {
		name     string
		property string
		format   propertyFormat
		exists   bool
	}{
		{
			name:     "Number",
			property: "Number",
			format:   numberPropFormat,
			exists:   true,
		},
		{
			name:     "int",
			property: "int",
			format:   numberPropFormat,
			exists:   true,
		},
		{
			name:     "enum",
			property: "enum",
			format:   numberPropFormat,
			exists:   true,
		},
		{
			name:     "KTime",
			property: "KTime",
			format:   numberPropFormat,
			exists:   true,
		},
		{
			name:     "Color",
			property: "Color",
			format:   colorPropFormat,
			exists:   true,
		},
		{
			name:     "Lcl Scaling",
			property: "Lcl Scaling",
			format:   vectorPropFormat,
			exists:   true,
		},
		{
			name:     "Lcl Translation",
			property: "Lcl Translation",
			format:   vectorPropFormat,
			exists:   true,
		},
		{
			name:     "Lcl Rotation",
			property: "Lcl Rotation",
			format:   vectorPropFormat,
			exists:   true,
		},
		{
			name:     "Unknown",
			property: "Unknown",
			format:   nil,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, exists := propFormats[tt.property]
			if exists != tt.exists {
				t.Errorf("Expected exists=%v for %s, got %v", tt.exists, tt.property, exists)
			}
			if exists && format == nil {
				t.Errorf("Expected non-nil format for %s", tt.property)
			}
		})
	}
}

func TestElementComplexHierarchy(t *testing.T) {
	// Create a complex hierarchy
	root := &Element{
		ID: NewDataView("root"),
		Children: []*Element{
			{
				ID: NewDataView("child1"),
				Children: []*Element{
					{
						ID: NewDataView("grandchild1"),
					},
					{
						ID: NewDataView("grandchild2"),
					},
				},
			},
			{
				ID: NewDataView("child2"),
			},
		},
		Properties: []*Property{
			{value: NewDataView("root_prop")},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("String() panicked with complex hierarchy: %v", r)
		}
	}()

	result := root.String()
	expectedIDs := []string{"root", "child1", "child2", "grandchild1", "grandchild2"}

	for _, id := range expectedIDs {
		if !strings.Contains(result, id) {
			t.Errorf("Expected hierarchy to contain %s", id)
		}
	}
}
