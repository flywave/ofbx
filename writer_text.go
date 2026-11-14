package ofbx

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

// Text (ASCII) FBX writer. EncodeText serializes an element tree in the
// ASCII FBX dialect; Load parses it back, so text output can be round-tripped
// and is convenient for debugging and diffs.

// EncodeText writes the element tree rooted at root as an ASCII FBX file.
// The version is recorded in the header comment only; the text format has no
// binary offsets.
func EncodeText(w io.Writer, root *Element, version uint32) error {
	if root == nil {
		return errors.New("ofbx: cannot encode nil element tree")
	}
	if version == 0 {
		version = DefaultWriteVersion
	}
	tw := &textWriter{out: bufio.NewWriter(w)}
	major := int(version) / 1000
	minor := (int(version) / 100) % 10
	patch := int(version) % 100
	tw.out.WriteString("; FBX " + strconv.Itoa(major) + "." + strconv.Itoa(minor) + "." + strconv.Itoa(patch) + " project file\n")
	tw.out.WriteString(";--------------------------------------------------------------------\n")

	for _, child := range root.Children {
		if err := tw.writeElement(child, 0); err != nil {
			return err
		}
	}
	return tw.out.Flush()
}

type textWriter struct {
	out *bufio.Writer
}

func (tw *textWriter) indent(depth int) {
	for i := 0; i < depth; i++ {
		tw.out.WriteByte('\t')
	}
}

// textSafeID sanitizes an element id for the text dialect, whose tokens are
// restricted to letters, digits and underscores. Byte soup names from
// corrupt files would otherwise truncate tokens and produce unparseable
// output; the sanitized id keeps the file loadable.
func textSafeID(id string) string {
	clean := make([]byte, len(id))
	for i := 0; i < len(id); i++ {
		b := id[i]
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9', b == '_':
			clean[i] = b
		default:
			clean[i] = '_'
		}
	}
	return string(clean)
}

func (tw *textWriter) writeElement(e *Element, depth int) error {
	if e == nil || e.ID == nil {
		return errors.New("ofbx: cannot encode element without ID")
	}
	tw.indent(depth)
	tw.out.WriteString(textSafeID(e.IDString()))
	tw.out.WriteString(":")

	for i, p := range e.Properties {
		if i == 0 {
			tw.out.WriteByte(' ')
		} else {
			tw.out.WriteString(", ")
		}
		if err := tw.writeProperty(p); err != nil {
			return fmt.Errorf("encoding property of element %q: %w", e.IDString(), err)
		}
	}

	if len(e.Children) == 0 {
		tw.out.WriteByte('\n')
		return nil
	}
	tw.out.WriteString(" {\n")
	for _, c := range e.Children {
		if err := tw.writeElement(c, depth+1); err != nil {
			return err
		}
	}
	tw.indent(depth)
	tw.out.WriteString("}\n")
	return nil
}

func (tw *textWriter) writeProperty(p *Property) error {
	if p == nil || p.value == nil {
		return errors.New("ofbx: property without value")
	}
	data := p.rawBytes()

	switch p.Type {
	case STRING: // 'S'
		return tw.writeString(data)
	case RAWSTRING: // 'R'
		// binary payloads hex-escape everything but plain printable bytes:
		// third-party ASCII parsers do not process escapes, so quote and
		// backslash bytes inside the payload would break their tokenization
		return tw.writeRawString(data)
	case BOOL: // 'C'
		if len(data) > 0 && data[0] != 0 {
			tw.out.WriteString("T")
		} else {
			tw.out.WriteString("F")
		}
	case INT16: // 'Y'
		tw.out.WriteString(strconv.FormatInt(intN(leUint(data, 2), 16), 10))
	case INTEGER: // 'I'
		v := int32(leUint(data, 4))
		tw.out.WriteString(strconv.FormatInt(int64(v), 10))
	case LONG: // 'L'
		tw.out.WriteString(strconv.FormatInt(int64(leUint(data, 8)), 10))
	case FLOAT: // 'F'
		f := leFloat32(data)
		tw.out.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
	case DOUBLE: // 'D'
		tw.out.WriteString(strconv.FormatFloat(leFloat64(data), 'g', -1, 64))
	case ArrayDOUBLE, ArrayFLOAT: // 'd', 'f'
		return tw.writeArray(p, data, false)
	case ArrayINT, ArrayLONG: // 'i', 'l'
		return tw.writeArray(p, data, true)
	case ArrayBOOL, ArrayBYTE:
		return tw.writeArray(p, data, true)
	default:
		return fmt.Errorf("ofbx: unsupported property type %q", string(p.Type))
	}
	return nil
}

// writeRawString emits a binary payload as a quoted string, hex-escaping
// every byte that is not plain printable ASCII.
func (tw *textWriter) writeRawString(data []byte) error {
	tw.out.WriteByte('"')
	for _, b := range data {
		if b >= 0x20 && b < 0x7f && b != '"' && b != '\\' {
			tw.out.WriteByte(b)
		} else {
			tw.out.WriteString(fmt.Sprintf("\\x%02x", b))
		}
	}
	tw.out.WriteByte('"')
	return nil
}

func (tw *textWriter) writeString(data []byte) error {
	tw.out.WriteByte('"')
	for _, b := range data {
		switch b {
		case '"':
			tw.out.WriteString("\\\"")
		case '\\':
			tw.out.WriteString("\\\\")
		case '\n':
			tw.out.WriteString("\\n")
		case '\r':
			tw.out.WriteString("\\r")
		case '\t':
			tw.out.WriteString("\\t")
		default:
			if b >= 0x20 && b < 0x7f {
				tw.out.WriteByte(b)
			} else {
				// control bytes (the \x00\x01 namespace separator and
				// friends) must be hex-escaped: raw NUL terminates C-string
				// based parsers
				tw.out.WriteString(fmt.Sprintf("\\x%02x", b))
			}
		}
	}
	tw.out.WriteByte('"')
	return nil
}

// writeArray emits `*N { a: v1,v2,... }`, breaking long arrays into lines.
func (tw *textWriter) writeArray(p *Property, data []byte, ints bool) error {
	count, encoding, err := arrayInfo(p, len(data))
	if err != nil {
		return err
	}
	if encoding == 1 {
		// decompress into raw bytes first
		raw, derr := decompressArray(p)
		if derr != nil {
			return derr
		}
		data = raw
	}
	tw.out.WriteString("*")
	tw.out.WriteString(strconv.Itoa(count))
	tw.out.WriteString(" {\n\t\ta: ")
	const perLine = 20
	size := p.Type.Size()
	for i := 0; i < count; i++ {
		if i > 0 {
			tw.out.WriteByte(',')
		}
		if i > 0 && i%perLine == 0 {
			tw.out.WriteString("\n\t\t")
		}
		switch {
		case ints && size == 1:
			tw.out.WriteString(strconv.Itoa(int(data[i])))
		case ints && size == 4:
			tw.out.WriteString(strconv.Itoa(int(int32(leUint(data[i*4:], 4)))))
		case ints && size == 8:
			tw.out.WriteString(strconv.FormatInt(int64(leUint(data[i*8:], 8)), 10))
		case !ints && size == 4:
			tw.out.WriteString(strconv.FormatFloat(float64(leFloat32(data[i*4:])), 'g', -1, 32))
		default:
			tw.out.WriteString(strconv.FormatFloat(leFloat64(data[i*8:]), 'g', -1, 64))
		}
	}
	tw.out.WriteString("\n\t}")
	return nil
}

func leUint(b []byte, n int) uint64 {
	var v uint64
	for i := 0; i < n && i < len(b); i++ {
		v |= uint64(b[i]) << (8 * uint(i))
	}
	return v
}

func leFloat32(b []byte) float32 {
	return math.Float32frombits(uint32(leUint(b, 4)))
}

func leFloat64(b []byte) float64 {
	return math.Float64frombits(leUint(b, 8))
}

func intN(v uint64, bits int) int64 {
	shift := uint(64 - bits)
	return int64(v<<shift) >> shift
}

// decompressArray inflates a zlib encoded array property's payload.
func decompressArray(p *Property) ([]byte, error) {
	if p.value == nil {
		return nil, errors.New("ofbx: property without value")
	}
	p.value.Seek(0, io.SeekStart)
	zr, err := zlib.NewReader(&p.value.Reader)
	if err != nil {
		return nil, fmt.Errorf("ofbx: bad compressed array"+": %w", err)
	}
	defer zr.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		return nil, fmt.Errorf("ofbx: decompressing array"+": %w", err)
	}
	p.value.Seek(0, io.SeekStart)
	expected := p.Count * p.Type.Size()
	if buf.Len() != expected {
		return nil, fmt.Errorf("ofbx: compressed array inflated to %d bytes, want %d", buf.Len(), expected)
	}
	return buf.Bytes(), nil
}
