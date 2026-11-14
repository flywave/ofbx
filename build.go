package ofbx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
)

// This file provides constructors for Properties and Elements so that FBX
// documents can be assembled programmatically and handed to the encoder
// (see Encode / Document).

// ---------- scalar properties ----------

// NewStringProperty creates an 'S' (string) property.
func NewStringProperty(s string) *Property {
	return &Property{Type: STRING, value: NewDataView(s)}
}

// NewRawProperty creates an 'R' (raw byte) property.
func NewRawProperty(b []byte) *Property {
	return &Property{Type: RAWSTRING, value: NewDataView(string(b))}
}

// NewBoolProperty creates a 'C' (1 byte boolean) property.
func NewBoolProperty(v bool) *Property {
	b := []byte{0}
	if v {
		b[0] = 1
	}
	return &Property{Type: BOOL, value: NewDataView(string(b))}
}

// NewInt16Property creates a 'Y' property.
func NewInt16Property(v int16) *Property {
	return newFixedProperty(INT16, uint64(uint16(v)), 2)
}

// NewInt32Property creates an 'I' property.
func NewInt32Property(v int32) *Property {
	return newFixedProperty(INTEGER, uint64(uint32(v)), 4)
}

// NewInt64Property creates an 'L' property.
func NewInt64Property(v int64) *Property {
	return newFixedProperty(LONG, uint64(v), 8)
}

// NewFloat32Property creates an 'F' property.
func NewFloat32Property(v float32) *Property {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
	return &Property{Type: FLOAT, value: NewDataView(string(buf))}
}

// NewFloat64Property creates a 'D' property.
func NewFloat64Property(v float64) *Property {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(v))
	return &Property{Type: DOUBLE, value: NewDataView(string(buf))}
}

func newFixedProperty(t PropertyType, v uint64, size int) *Property {
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(v >> (8 * uint(i)))
	}
	return &Property{Type: t, value: NewDataView(string(buf))}
}

// ---------- array properties ----------

// NewDoubleArrayProperty creates a 'd' array property. When compress is true
// the payload is zlib encoded (Encoding=1) the way most exporters do.
func NewDoubleArrayProperty(vals []float64, compress bool) (*Property, error) {
	buf := make([]byte, 8*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[8*i:], math.Float64bits(v))
	}
	return newArrayProperty(ArrayDOUBLE, buf, len(vals), compress)
}

// NewFloatArrayProperty creates an 'f' array property.
func NewFloatArrayProperty(vals []float32, compress bool) (*Property, error) {
	buf := make([]byte, 4*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint32(buf[4*i:], math.Float32bits(v))
	}
	return newArrayProperty(ArrayFLOAT, buf, len(vals), compress)
}

// NewInt32ArrayProperty creates an 'i' array property.
func NewInt32ArrayProperty(vals []int32, compress bool) (*Property, error) {
	buf := make([]byte, 4*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint32(buf[4*i:], uint32(v))
	}
	return newArrayProperty(ArrayINT, buf, len(vals), compress)
}

// NewInt64ArrayProperty creates an 'l' array property.
func NewInt64ArrayProperty(vals []int64, compress bool) (*Property, error) {
	buf := make([]byte, 8*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[8*i:], uint64(v))
	}
	return newArrayProperty(ArrayLONG, buf, len(vals), compress)
}

// NewByteArrayProperty creates a 'c' (byte) array property.
func NewByteArrayProperty(vals []byte, compress bool) (*Property, error) {
	cp := make([]byte, len(vals))
	copy(cp, vals)
	return newArrayProperty(ArrayBYTE, cp, len(vals), compress)
}

// NewBoolArrayProperty creates a 'b' (bool) array property.
func NewBoolArrayProperty(vals []bool, compress bool) (*Property, error) {
	buf := make([]byte, len(vals))
	for i, v := range vals {
		if v {
			buf[i] = 1
		}
	}
	return newArrayProperty(ArrayBOOL, buf, len(vals), compress)
}

func newArrayProperty(t PropertyType, payload []byte, count int, compress bool) (*Property, error) {
	p := &Property{Type: t, Count: count}
	if compress && len(payload) > 0 {
		var zbuf bytes.Buffer
		zw := zlib.NewWriter(&zbuf)
		if _, err := zw.Write(payload); err != nil {
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		p.Encoding = 1
		p.compressedLength = uint32(zbuf.Len())
		p.value = NewDataView(zbuf.String())
		return p, nil
	}
	p.Encoding = 0
	p.compressedLength = uint32(len(payload))
	p.value = NewDataView(string(payload))
	return p, nil
}

// ---------- element construction ----------

// NewElement creates an empty element with the given ID.
func NewElement(id string) *Element {
	return &Element{ID: NewDataView(id)}
}

// AddProperty appends a property and returns the element so calls can chain.
func (e *Element) AddProperty(p *Property) *Element {
	e.Properties = append(e.Properties, p)
	return e
}

// AddChild appends a child element and returns the child.
func (e *Element) AddChild(c *Element) *Element {
	e.Children = append(e.Children, c)
	return c
}

// AddChildElement creates a child with the given ID and properties, appends
// it and returns it.
func (e *Element) AddChildElement(id string, props ...*Property) *Element {
	c := NewElement(id)
	for _, p := range props {
		if p != nil {
			c.Properties = append(c.Properties, p)
		}
	}
	e.Children = append(e.Children, c)
	return c
}

// Child returns the first child with the given ID, or nil.
func (e *Element) Child(id string) *Element {
	for _, c := range e.Children {
		if c.ID != nil && c.IDString() == id {
			return c
		}
	}
	return nil
}

// ChildrenOf returns all children with the given ID.
func (e *Element) ChildrenOf(id string) []*Element {
	out := []*Element{}
	for _, c := range e.Children {
		if c.ID != nil && c.IDString() == id {
			out = append(out, c)
		}
	}
	return out
}

// ReplaceChild substitutes the first child with the given ID, or appends the
// new child when no such child exists yet.
func (e *Element) ReplaceChild(id string, replacement *Element) {
	for i, c := range e.Children {
		if c.ID != nil && c.IDString() == id {
			e.Children[i] = replacement
			return
		}
	}
	e.Children = append(e.Children, replacement)
}

// RemoveChild removes the first child with the given ID, reporting whether a
// child was removed.
func (e *Element) RemoveChild(id string) bool {
	for i, c := range e.Children {
		if c.ID != nil && c.IDString() == id {
			e.Children = append(e.Children[:i], e.Children[i+1:]...)
			return true
		}
	}
	return false
}
