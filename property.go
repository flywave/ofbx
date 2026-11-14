package ofbx

import (
	"fmt"
	"io"
)

// PropertyType is a mapping of letter to data type
type PropertyType rune

// Property types block
const (
	BOOL        PropertyType = 'C'
	INT16       PropertyType = 'Y'
	LONG        PropertyType = 'L'
	INTEGER     PropertyType = 'I'
	STRING      PropertyType = 'S'
	RAWSTRING   PropertyType = 'R'
	FLOAT       PropertyType = 'F'
	DOUBLE      PropertyType = 'D'
	ArrayDOUBLE PropertyType = 'd'
	ArrayINT    PropertyType = 'i'
	ArrayLONG   PropertyType = 'l'
	ArrayFLOAT  PropertyType = 'f'
	ArrayBOOL   PropertyType = 'b'
	ArrayBYTE   PropertyType = 'c'
)

var (
	propertyTypeSizes = map[PropertyType]int{
		BOOL:        1,
		INT16:       2,
		DOUBLE:      8,
		INTEGER:     4,
		LONG:        8,
		FLOAT:       4,
		ArrayDOUBLE: 8,
		ArrayINT:    4,
		ArrayLONG:   8,
		ArrayFLOAT:  4,
		ArrayBOOL:   1,
		ArrayBYTE:   1,
	}
)

func (p *Property) stringValue() string {

	switch p.Type {
	case BOOL:
		return fmt.Sprintf("%v", p.value.toBool())
	case LONG:
		return fmt.Sprintf("%d", p.value.toint64())
	case INTEGER:
		return fmt.Sprintf("%d", p.value.toInt32())
	case STRING:
		return p.value.String()
	case RAWSTRING:
		return p.value.String()
	case FLOAT:
		return fmt.Sprintf("%f", p.value.toFloat())
	case DOUBLE:
		return fmt.Sprintf("%f", p.value.toDouble())
	case ArrayDOUBLE:
		sli, err := parseArrayRawFloat64(p)
		if err != nil {
			return "Bad Format F64s " + err.Error()
		}
		return fmt.Sprintf("%v", sli)
	case ArrayINT:
		sli, err := parseArrayRawInt(p)
		if err != nil {
			return "Bad Format Ints " + err.Error()
		}
		return fmt.Sprintf("%v", sli)
	case ArrayLONG:
		sli, err := parseArrayRawInt64(p)
		if err != nil {
			return "Bad Format I64s " + err.Error()
		}
		return fmt.Sprintf("%v", sli)
	case ArrayFLOAT:
		sli, err := parseArrayRawFloat32(p)
		if err != nil {
			return "Bad Format F32s " + err.Error()
		}
		return fmt.Sprintf("%v", sli)
	case ArrayBOOL:
		return "Bool array not implemented"
	case ArrayBYTE:
		return "Byte array not implemented"
	}

	return "Error: Not a known property Type " + string(p.Type)
}

// Size returns the current property type's size
func (pt PropertyType) Size() int {
	return propertyTypeSizes[pt]
}

// IsArray checks whether the property is an array
func (pt PropertyType) IsArray() bool {
	switch pt {
	case ArrayDOUBLE, ArrayFLOAT, ArrayINT, ArrayLONG, ArrayBOOL, ArrayBYTE:
		return true
	}
	return false
}

// A Property is template class is used to ensure that the data of a FbxObject is strongly typed
type Property struct {
	Count            int
	Type             PropertyType
	value            *DataView
	Encoding         uint32
	compressedLength uint32
}

func (p *Property) getValuesF32() ([]float32, error) {
	return parseArrayRawFloat32(p)
}

func (p *Property) getValuesInt64() ([]int64, error) {
	return parseArrayRawInt64(p)
}

// propFloat reads a numeric property as float64 regardless of the concrete
// type it was stored as. Text FBX loses type information (an integer "1" and
// a double "1" are indistinguishable), and exporters also disagree about
// whether P values are ints or doubles, so consumers must not assume.
func propFloat(p *Property) float64 {
	if p == nil || p.value == nil {
		return 0
	}
	switch p.Type {
	case BOOL:
		if b := p.rawBytes(); len(b) > 0 && b[0] != 0 {
			return 1
		}
		return 0
	case INT16:
		return float64(intN(leUint(p.rawBytes(), 2), 16))
	case INTEGER:
		return float64(p.value.toInt32())
	case LONG:
		return float64(p.value.toint64())
	case FLOAT:
		return float64(p.value.toFloat())
	case DOUBLE:
		return p.value.toDouble()
	}
	return 0
}

// propInt reads a numeric property as int regardless of the concrete type.
func propInt(p *Property) int {
	if p == nil || p.value == nil {
		return 0
	}
	switch p.Type {
	case BOOL:
		if b := p.rawBytes(); len(b) > 0 && b[0] != 0 {
			return 1
		}
		return 0
	case INT16:
		return int(intN(leUint(p.rawBytes(), 2), 16))
	case INTEGER:
		return int(p.value.toInt32())
	case LONG:
		return int(p.value.toint64())
	case FLOAT:
		return int(p.value.toFloat())
	case DOUBLE:
		return int(p.value.toDouble())
	}
	return 0
}

func findChildren(element *Element, id string) []*Element {
	iterables := element.Children
	for idx, val := range iterables {
		if val.IDString() == id {
			return iterables[idx:]
		}
	}
	return []*Element{}
}

func assignSingleChildProperty(element *Element, id string, dv **DataView) bool {
	prop := findSingleChildProperty(element, id)
	if prop != nil {
		*dv = prop.value
		return true
	}
	return false
}

func findSingleChildProperty(element *Element, id string) *Property {
	iterables := element.Children
	for idx, val := range iterables {
		if val.IDString() == id {
			if len(iterables[idx].Properties) > 0 {
				return iterables[idx].Properties[0]
			}
		}
	}
	return nil
}

func findChildProperty(element *Element, id string) []*Property {
	iterables := element.Children
	for idx, val := range iterables {
		if val.IDString() == id {
			return iterables[idx].Properties
		}
	}
	return nil
}

func resolveProperty(obj Obj, name string) *Element {
	elems := findChildren(obj.Element(), "Properties70")
	if len(elems) == 0 || elems == nil {
		return nil
	}

	elems = elems[0].Children
	for _, elem := range elems {
		if prop := elem.getProperty(0); prop != nil && prop.value.String() == name {
			return elem
		}
	}
	return nil
}

func isString(prop *Property) bool {
	if prop == nil {
		return false
	}
	return prop.Type == STRING
}

func isLong(prop *Property) bool {
	if prop == nil {
		return false
	}
	// integer ids arrive as 'L' from binary files and as 'I' when the
	// value fits in 32 bits (the text reader materializes small ints that
	// way, matching the FBX SDK)
	return prop.Type == LONG || prop.Type == INTEGER
}

func (p *Property) String() string {
	return p.stringPrefix("")
}

func (p *Property) stringPrefix(prefix string) string {
	p.value.Seek(0, io.SeekStart)
	if p.value.Len() == 0 {
		return ""
	}
	s := prefix + p.stringValue()
	// s += ", proptype= " + fmt.Sprintf("%q", p.typ)
	// s += "count=" + fmt.Sprintf("%d", p.count)
	// s += ", encoding=" + fmt.Sprintf("%d", p.encoding)
	// s += ", compressedLen=" + fmt.Sprintf("%d", p.compressedLength)
	return s
}

// Type exposes the property's FBX type character.
func (p *Property) GetType() PropertyType { return p.Type }

// Count returns the element count for array properties.
func (p *Property) GetCount() int { return p.Count }

// ValueU64 interprets the property payload as an unsigned integer
// regardless of the stored width ('I' or 'L').
func (p *Property) ValueU64() uint64 { return uint64(propInt(p)) }

// ValueI64 interprets the property payload as an integer regardless of the
// stored width ('I' or 'L').
func (p *Property) ValueI64() int64 { return int64(propInt(p)) }

// ValueF64 interprets the property payload as a double.
func (p *Property) ValueF64() float64 { return propFloat(p) }

// ValueString returns string/raw payloads as text.
func (p *Property) ValueString() string { return p.value.String() }

// ValueBytes returns raw payload bytes.
func (p *Property) ValueBytes() []byte { return p.rawBytes() }

// ArrayF64 decodes float array payloads.
func (p *Property) ArrayF64() ([]float64, error) { return parseArrayRawFloat64(p) }

// ArrayI64 decodes integer array payloads.
func (p *Property) ArrayI64() ([]int64, error) { return parseArrayRawInt64(p) }
