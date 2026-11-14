package ofbx

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

// Binary FBX writer. Encode serializes an Element tree (as produced by
// tokenize / Load, or assembled via the builders in build.go and document.go)
// back into the Kaydara binary FBX container.

// FBX binary versions accepted by Encode.
const (
	FBXVersion7100 uint32 = 7100
	FBXVersion7400 uint32 = 7400
	FBXVersion7500 uint32 = 7500
	FBXVersion7700 uint32 = 7700
)

// DefaultWriteVersion is used when Encode is called with version 0. FBX 7400
// is readable by every FBX importer in the wild.
const DefaultWriteVersion = FBXVersion7400

// footer magic observed in Autodesk written files. The block before the
// trailing version field varies per file revision; readers do not validate it
// but the FBX SDK expects its presence.
var (
	footerIDPre7500 = []byte{
		0xfa, 0xbc, 0xab, 0x09, 0xd0, 0xc8, 0xd4, 0x66,
		0xb1, 0x76, 0xfb, 0x83, 0x1c, 0xf7, 0x26, 0x7e,
	}
	footerID7500Plus = []byte{
		0xfa, 0xbc, 0xab, 0x0a, 0xd3, 0xcf, 0xd3, 0x61,
		0xb6, 0x79, 0xf5, 0x82, 0x1f, 0xf2, 0x24, 0x7b,
	}
	footerTrailingMagic = []byte{
		0xf8, 0x5a, 0x8c, 0x6a, 0xde, 0xf5, 0xd9, 0x7e,
		0xec, 0xe9, 0x0c, 0xe3, 0x75, 0x8f, 0x29, 0x0b,
	}
)

type encoder struct {
	version    uint32
	bigOffsets bool // version >= 7500 uses 64-bit record offsets
}

// Encode writes the element tree rooted at root as a binary FBX document.
// version selects the container revision (0 means DefaultWriteVersion);
// versions >= 7500 switch to 64-bit node offsets.
func Encode(w io.Writer, root *Element, version uint32) error {
	switch version {
	case 0:
		version = DefaultWriteVersion
	case FBXVersion7100, FBXVersion7400, FBXVersion7500, FBXVersion7700:
	default:
		return errors.Errorf("ofbx: unsupported write version %d", version)
	}
	if root == nil {
		return errors.New("ofbx: cannot encode nil element tree")
	}

	enc := &encoder{version: version, bigOffsets: version >= FBXVersion7500}

	var buf bytes.Buffer
	// 21 byte magic, then 0x1a 0x00, then u32 version.
	buf.WriteString("Kaydara FBX Binary  ")
	buf.WriteByte(0)
	buf.WriteByte(0x1a)
	buf.WriteByte(0)
	writeUint32(&buf, int(version))

	for _, child := range root.Children {
		if _, err := enc.writeNode(&buf, child, buf.Len()); err != nil {
			return err
		}
	}
	// top level NULL record terminates the node stream
	enc.writeZeros(&buf, enc.sentinelSize())
	enc.writeFooter(&buf)

	_, err := w.Write(buf.Bytes())
	return err
}

func (enc *encoder) offsetSize() int {
	if enc.bigOffsets {
		return 8
	}
	return 4
}

func (enc *encoder) sentinelSize() int {
	if enc.bigOffsets {
		return 25
	}
	return 13
}

func (enc *encoder) headerSize(nameLen int) int {
	return 3*enc.offsetSize() + 1 + nameLen
}

func (enc *encoder) writeOffset(buf *bytes.Buffer, v int) {
	if enc.bigOffsets {
		var tmp [8]byte
		binary.LittleEndian.PutUint64(tmp[:], uint64(v))
		buf.Write(tmp[:])
		return
	}
	writeUint32(buf, v)
}

// writeNode serializes one node record starting at the absolute offset start
// and returns the number of bytes written. Properties and children are first
// serialized into scratch buffers so the record header (including the
// endOffset, which must be absolute) can be computed before anything is
// appended to buf.
func (enc *encoder) writeNode(buf *bytes.Buffer, e *Element, start int) (int, error) {
	if e == nil || e.ID == nil {
		return 0, errors.New("ofbx: cannot encode element without ID")
	}
	name := e.ID.String()
	if len(name) > 255 {
		return 0, errors.Errorf("ofbx: element name %q exceeds 255 bytes", name)
	}

	props := &bytes.Buffer{}
	for _, p := range e.Properties {
		if err := enc.writeProperty(props, p); err != nil {
			return 0, errors.Wrapf(err, "encoding property of element %q", name)
		}
	}

	kids := &bytes.Buffer{}
	off := start + enc.headerSize(len(name)) + props.Len()
	for _, c := range e.Children {
		n, err := enc.writeNode(kids, c, off)
		if err != nil {
			return 0, err
		}
		off += n
	}

	// nodes that carry children are terminated by a NULL record
	end := off
	if len(e.Children) > 0 {
		end += enc.sentinelSize()
	}
	if !enc.bigOffsets && end > math.MaxUint32 {
		return 0, errors.Errorf("ofbx: node %q ends past 4GB; write with version >= 7500 (64-bit offsets)", name)
	}

	enc.writeOffset(buf, end)
	enc.writeOffset(buf, len(e.Properties))
	enc.writeOffset(buf, props.Len())
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	buf.Write(props.Bytes())
	buf.Write(kids.Bytes())
	if len(e.Children) > 0 {
		enc.writeZeros(buf, enc.sentinelSize())
	}
	return end - start, nil
}

func (enc *encoder) writeProperty(buf *bytes.Buffer, p *Property) error {
	if p == nil || p.value == nil {
		return errors.New("ofbx: property without value")
	}
	data := p.rawBytes()

	buf.WriteByte(byte(p.Type))
	switch p.Type {
	case STRING, RAWSTRING: // 'S', 'R'
		writeUint32(buf, len(data))
		buf.Write(data)
	case BOOL, INT16, INTEGER, FLOAT, DOUBLE, LONG: // 'C','Y','I','F','D','L'
		if len(data) != p.Type.Size() {
			return errors.Errorf("ofbx: scalar property %q payload is %d bytes, want %d",
				string(p.Type), len(data), p.Type.Size())
		}
		buf.Write(data)
	case ArrayDOUBLE, ArrayFLOAT, ArrayINT, ArrayLONG, ArrayBOOL, ArrayBYTE: // 'd','f','i','l','b','c'
		count := p.Count
		if count == 0 && len(data) > 0 {
			// tolerate hand-built properties without a count
			count = len(data) / p.Type.Size()
		}
		if p.Encoding == 0 && count*p.Type.Size() != len(data) {
			return errors.Errorf("ofbx: uncompressed array %q holds %d bytes for %d elements of size %d",
				string(p.Type), len(data), count, p.Type.Size())
		}
		if p.Encoding > 1 {
			return errors.Errorf("ofbx: invalid array encoding %d", p.Encoding)
		}
		writeUint32(buf, count)
		writeUint32(buf, int(p.Encoding))
		writeUint32(buf, len(data))
		buf.Write(data)
	default:
		return errors.Errorf("ofbx: unsupported property type %q", string(p.Type))
	}
	return nil
}

func (enc *encoder) writeFooter(buf *bytes.Buffer) {
	enc.padTo16(buf)
	if enc.bigOffsets {
		buf.Write(footerID7500Plus)
	} else {
		buf.Write(footerIDPre7500)
	}
	enc.padTo16(buf)
	writeUint32(buf, int(enc.version))
	enc.writeZeros(buf, 120)
	buf.Write(footerTrailingMagic)
}

func (enc *encoder) padTo16(buf *bytes.Buffer) {
	if rem := buf.Len() % 16; rem != 0 {
		enc.writeZeros(buf, 16-rem)
	}
}

func (enc *encoder) writeZeros(buf *bytes.Buffer, n int) {
	for i := 0; i < n; i++ {
		buf.WriteByte(0)
	}
}

func writeUint32(buf *bytes.Buffer, v int) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], uint32(v))
	buf.Write(tmp[:])
}

// rawBytes returns the property's payload bytes and rewinds the underlying
// DataView so it can be read again.
func (p *Property) rawBytes() []byte {
	if p.value == nil {
		return nil
	}
	p.value.Seek(0, io.SeekStart)
	data, err := io.ReadAll(p.value)
	if err != nil {
		return nil
	}
	p.value.Seek(0, io.SeekStart)
	return data
}
