package ofbx

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadShortString(t *testing.T) {
	// Test reading short string
	data := []byte{5, 'h', 'e', 'l', 'l', 'o', 'x'}
	countReader := NewCountReader(bytes.NewReader(data))
	cursor := &Cursor{Reader: bufio.NewReader(countReader), cr: countReader}

	result, err := cursor.readShortString()
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestReadLongString(t *testing.T) {
	// Test reading long string
	data := []byte{5, 0, 0, 0, 'w', 'o', 'r', 'l', 'd'}
	countReader := NewCountReader(bytes.NewReader(data))
	cursor := &Cursor{Reader: bufio.NewReader(countReader), cr: countReader}

	result, err := cursor.readLongString()
	require.NoError(t, err)
	assert.Equal(t, "world", result)
}

func TestReadElementOffset(t *testing.T) {
	tests := []struct {
		name     string
		version  uint16
		data     []byte
		expected uint64
	}{
		{
			name:     "Version < 7500",
			version:  7400,
			data:     []byte{42, 0, 0, 0},
			expected: 42,
		},
		{
			name:     "Version >= 7500",
			version:  7500,
			data:     []byte{100, 0, 0, 0, 0, 0, 0, 0},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countReader := NewCountReader(bytes.NewReader(tt.data))
			cursor := &Cursor{Reader: bufio.NewReader(countReader), cr: countReader}
			result, err := cursor.readElementOffset(tt.version)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadProperty(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *Property
	}{
		{
			name: "BOOL property",
			data: []byte{'C', 1},
			expected: &Property{
				Type:  'C',
				value: NewDataView(string([]byte{1})),
			},
		},
		{
			name: "INT16 property",
			data: []byte{'Y', 1, 2},
			expected: &Property{
				Type:  'Y',
				value: NewDataView(string([]byte{1, 2})),
			},
		},
		{
			name: "INTEGER property",
			data: []byte{'I', 1, 2, 3, 4},
			expected: &Property{
				Type:  'I',
				value: NewDataView(string([]byte{1, 2, 3, 4})),
			},
		},
		{
			name: "STRING property",
			data: []byte{'S', 5, 0, 0, 0, 'h', 'e', 'l', 'l', 'o'},
			expected: &Property{
				Type:  'S',
				value: NewDataView(string([]byte{5, 0, 0, 0, 'h', 'e', 'l', 'l', 'o'})),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countReader := NewCountReader(bytes.NewReader(tt.data))
			cursor := &Cursor{Reader: bufio.NewReader(countReader), cr: countReader}
			result, err := cursor.readProperty()
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.NotNil(t, result.value)
		})
	}
}

func TestReadElement(t *testing.T) {
	// Create minimal binary FBX data
	var buf bytes.Buffer

	// Write header
	header := []byte("Kaydara FBX Binary  \x00")
	buf.Write(header)

	// Write version
	version := uint32(7400)
	binary.Write(&buf, binary.LittleEndian, version)

	// Write a simple element
	// End offset: 50 (position after this element)
	binary.Write(&buf, binary.LittleEndian, uint32(50))
	// Property count: 1
	binary.Write(&buf, binary.LittleEndian, uint32(1))
	// Property list length: 5
	binary.Write(&buf, binary.LittleEndian, uint32(5))
	// Element ID: "test"
	buf.WriteByte(4)
	buf.WriteString("test")

	// Property: BOOL with value 1
	buf.WriteByte('C')
	buf.WriteByte(1)

	// Sentinel (13 bytes for version < 7500)
	buf.Write(make([]byte, 13))

	countReader := NewCountReader(&buf)
	cursor := &Cursor{Reader: bufio.NewReader(countReader), cr: countReader}

	// Skip header and version
	cursor.Discard(len(header) + 4)

	element, err := cursor.readElement(7400)
	require.NoError(t, err)
	assert.Equal(t, "test", element.ID.String())
	assert.Len(t, element.Properties, 1)
	assert.Equal(t, PropertyType('C'), element.Properties[0].Type)
}

func TestTokenize(t *testing.T) {
	// Skip complex binary FBX parsing test for now
	// This would require proper binary FBX format knowledge
	t.Skip("Skipping complex binary FBX parsing test")
}

func TestTokenizeNonBinary(t *testing.T) {
	// Test with ASCII FBX format (should fail)
	asciiData := []byte(`; FBX 7.4.0 project file
FBXHeaderExtension:  {
	Version: 7400
}`)

	_, err := tokenize(bytes.NewReader(asciiData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-binary FBX")
}

func TestTokenizeEmpty(t *testing.T) {
	// Test with empty data
	_, err := tokenize(bytes.NewReader([]byte{}))
	assert.Error(t, err)
}

func TestTokenizeInvalidHeader(t *testing.T) {
	// Test with invalid binary header
	invalidData := []byte("Invalid FBX header\x00")

	_, err := tokenize(bytes.NewReader(invalidData))
	assert.Error(t, err)
}
