package ofbx

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestNewDataView(t *testing.T) {
	input := "hello world"
	dv := NewDataView(input)

	if dv == nil {
		t.Fatal("Expected non-nil DataView")
	}

	// Test String method
	result := dv.String()
	if result != input {
		t.Errorf("Expected '%s', got '%s'", input, result)
	}
}

func TestBufferDataView(t *testing.T) {
	buffer := bytes.NewBufferString("test data")
	dv := BufferDataView(buffer)

	if dv == nil {
		t.Fatal("Expected non-nil DataView")
	}

	result := dv.String()
	if result != "test data" {
		t.Errorf("Expected 'test data', got '%s'", result)
	}
}

func TestDataViewString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "simple text",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "unicode text",
			input: "Hello 世界 🌍",
			want:  "Hello 世界 🌍",
		},
		{
			name:  "binary data",
			input: string([]byte{0x00, 0x01, 0x02, 0xFF}),
			want:  string([]byte{0x00, 0x01, 0x02, 0xFF}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(tt.input)
			result := dv.String()
			if result != tt.want {
				t.Errorf("String() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestDataViewToUint64(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint64
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 0,
		},
		{
			name:     "one",
			data:     []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "max uint64",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: math.MaxUint64,
		},
		{
			name:     "little endian 123456789",
			data:     []byte{0x15, 0xCD, 0x5B, 0x07, 0x00, 0x00, 0x00, 0x00},
			expected: 123456789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.touint64()
			if result != tt.expected {
				t.Errorf("touint64() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDataViewToInt64(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int64
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 0,
		},
		{
			name:     "positive one",
			data:     []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "negative one",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: -1,
		},
		{
			name:     "max int64",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F},
			expected: math.MaxInt64,
		},
		{
			name:     "min int64",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80},
			expected: math.MinInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toint64()
			if result != tt.expected {
				t.Errorf("toint64() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDataViewToInt32(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int32
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0,
		},
		{
			name:     "positive one",
			data:     []byte{0x01, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "negative one",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: -1,
		},
		{
			name:     "max int32",
			data:     []byte{0xFF, 0xFF, 0xFF, 0x7F},
			expected: math.MaxInt32,
		},
		{
			name:     "min int32",
			data:     []byte{0x00, 0x00, 0x00, 0x80},
			expected: math.MinInt32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toInt32()
			if result != tt.expected {
				t.Errorf("toInt32() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDataViewToUint32(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint32
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0,
		},
		{
			name:     "one",
			data:     []byte{0x01, 0x00, 0x00, 0x00},
			expected: 1,
		},
		{
			name:     "max uint32",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: math.MaxUint32,
		},
		{
			name:     "little endian 123456789",
			data:     []byte{0x15, 0xCD, 0x5B, 0x07},
			expected: 123456789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toUint32()
			if result != tt.expected {
				t.Errorf("toUint32() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDataViewToDouble(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: 0.0,
		},
		{
			name:     "one",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F},
			expected: 1.0,
		},
		{
			name:     "pi",
			data:     []byte{0x18, 0x2D, 0x44, 0x54, 0xFB, 0x21, 0x09, 0x40},
			expected: math.Pi,
		},
		{
			name:     "negative one",
			data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0xBF},
			expected: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toDouble()
			if math.Abs(result-tt.expected) > 1e-15 {
				t.Errorf("toDouble() = %.15f, want %.15f", result, tt.expected)
			}
		})
	}
}

func TestDataViewToFloat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected float32
	}{
		{
			name:     "zero",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0.0,
		},
		{
			name:     "one",
			data:     []byte{0x00, 0x00, 0x80, 0x3F},
			expected: 1.0,
		},
		{
			name:     "pi",
			data:     []byte{0xDB, 0x0F, 0x49, 0x40},
			expected: float32(math.Pi),
		},
		{
			name:     "negative one",
			data:     []byte{0x00, 0x00, 0x80, 0xBF},
			expected: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toFloat()
			if math.Abs(float64(result-tt.expected)) > 1e-7 {
				t.Errorf("toFloat() = %.7f, want %.7f", result, tt.expected)
			}
		})
	}
}

func TestDataViewToBool(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "false",
			data:     []byte{0x00},
			expected: false,
		},
		{
			name:     "true",
			data:     []byte{0x01},
			expected: true,
		},
		{
			name:     "non-zero as true",
			data:     []byte{0xFF},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			result := dv.toBool()
			if result != tt.expected {
				t.Errorf("toBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDataViewInsufficientData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		fn   func(*DataView)
	}{
		{
			name: "uint64 insufficient data",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, // 7 bytes, need 8
			fn:   func(dv *DataView) { dv.touint64() },
		},
		{
			name: "int64 insufficient data",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, // 7 bytes, need 8
			fn:   func(dv *DataView) { dv.toint64() },
		},
		{
			name: "int32 insufficient data",
			data: []byte{0x01, 0x02, 0x03}, // 3 bytes, need 4
			fn:   func(dv *DataView) { dv.toInt32() },
		},
		{
			name: "uint32 insufficient data",
			data: []byte{0x01, 0x02, 0x03}, // 3 bytes, need 4
			fn:   func(dv *DataView) { dv.toUint32() },
		},
		{
			name: "double insufficient data",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, // 7 bytes, need 8
			fn:   func(dv *DataView) { dv.toDouble() },
		},
		{
			name: "float insufficient data",
			data: []byte{0x01, 0x02, 0x03}, // 3 bytes, need 4
			fn:   func(dv *DataView) { dv.toFloat() },
		},
		{
			name: "bool insufficient data",
			data: []byte{}, // 0 bytes, need 1
			fn:   func(dv *DataView) { dv.toBool() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := NewDataView(string(tt.data))
			// These should not panic, even with insufficient data
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()
			tt.fn(dv)
		})
	}
}

func TestDataViewSeekReset(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	dv := NewDataView(string(data))

	// First read
	val1 := dv.touint64()
	expected := uint64(0x0807060504030201) // little endian
	if val1 != expected {
		t.Errorf("First read = %d, want %d", val1, expected)
	}

	// Second read (should reset position)
	val2 := dv.touint64()
	if val2 != expected {
		t.Errorf("Second read = %d, want %d", val2, expected)
	}
}

func TestDataViewEmpty(t *testing.T) {
	dv := NewDataView("")

	// Test all methods with empty data
	if dv.String() != "" {
		t.Error("Expected empty string")
	}

	if dv.touint64() != 0 {
		t.Error("Expected 0 for empty uint64")
	}

	if dv.toint64() != 0 {
		t.Error("Expected 0 for empty int64")
	}

	if dv.toInt32() != 0 {
		t.Error("Expected 0 for empty int32")
	}

	if dv.toUint32() != 0 {
		t.Error("Expected 0 for empty uint32")
	}

	if dv.toDouble() != 0 {
		t.Error("Expected 0 for empty double")
	}

	if dv.toFloat() != 0 {
		t.Error("Expected 0 for empty float")
	}

	if dv.toBool() != false {
		t.Error("Expected false for empty bool")
	}
}

// Benchmark tests
func BenchmarkDataViewString(b *testing.B) {
	dv := NewDataView("benchmark test string")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.String()
	}
}

func BenchmarkDataViewToUint64(b *testing.B) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, 123456789)
	dv := NewDataView(string(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.touint64()
	}
}

func BenchmarkDataViewToInt64(b *testing.B) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, 123456789)
	dv := NewDataView(string(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.toint64()
	}
}

func BenchmarkDataViewToDouble(b *testing.B) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, math.Float64bits(3.14159))
	dv := NewDataView(string(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.toDouble()
	}
}

func BenchmarkDataViewToFloat(b *testing.B) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(3.14159))
	dv := NewDataView(string(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dv.toFloat()
	}
}

func BenchmarkNewDataView(b *testing.B) {
	input := "benchmark input string"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDataView(input)
	}
}
