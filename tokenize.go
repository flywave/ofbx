package ofbx

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Header is a magic set of ints
type Header struct {
	Reserved [2]uint8
	Version  uint32
}

func (h Header) String() string {
	b := []byte{}
	for _, n := range h.Reserved {
		b = append(b, byte(n))
	}
	s := string(b)
	s += " " + strconv.Itoa(int(h.Version))
	return s
}

// Cursor is a rapper for a reader
type Cursor struct {
	*bufio.Reader
	cr *CountReader
}

// ReadSoFar returns how much of the data has been read
func (c *Cursor) ReadSoFar() int {
	return c.cr.ReadSoFar - c.Reader.Buffered()
}

func (c *Cursor) readShortString() (string, error) {
	length, err := c.ReadByte()
	if err != nil {
		return "", err
	}
	//fmt.Print("slength is ", length)
	byt := make([]byte, int(length))
	_, err = io.ReadFull(c, byt)
	if err != nil {
		return "", err
	}
	return string(byt), nil
}

// readLongStringBytes reads a u32-length byte payload without copying
// through a string.
func (c *Cursor) readLongStringBytes() ([]byte, error) {
	var length uint32
	if err := binary.Read(c, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if int64(length) > maxPropertyBytes {
		return nil, fmt.Errorf("ofbx: string property length %d exceeds limit", length)
	}
	byt := make([]byte, length)
	if _, err := io.ReadFull(c, byt); err != nil {
		return nil, err
	}
	return byt, nil
}

func (c *Cursor) readLongString() (string, error) {
	var length uint32
	err := binary.Read(c, binary.LittleEndian, &length)
	if err != nil {
		return "", err
	}
	if int64(length) > maxPropertyBytes {
		return "", fmt.Errorf("ofbx: string property length %d exceeds limit", length)
	}
	byt := make([]byte, length)
	_, err = io.ReadFull(c, byt)
	if err != nil {
		return "", err
	}
	//fmt.Print(" |S Prop had ", length, "|")
	return string(byt), nil
}

// isBinaryFBX peeks at the buffered reader (without consuming bytes) and
// discards the binary magic when present, reporting whether the stream is a
// binary FBX file.
func isBinaryFBX(br *bufio.Reader) bool {
	magic := append([]byte("Kaydara FBX Binary  "), 0)
	header, err := br.Peek(len(magic))
	if err != nil || !bytes.Equal(magic, header) {
		return false
	}
	br.Discard(len(magic))
	return true
}

func (c *Cursor) readElementOffset(version uint16) (uint64, error) {
	if version >= 7500 {
		var i uint64
		err := binary.Read(c, binary.LittleEndian, &i)
		return i, err
	}
	var i uint32
	err := binary.Read(c, binary.LittleEndian, &i)
	return uint64(i), err
}

func (c *Cursor) readBytes(length int) []byte {
	if length < 0 || int64(length) > maxPropertyBytes {
		return nil
	}
	tempArr := make([]byte, length)
	n, err := io.ReadFull(c, tempArr)
	if err != nil {
		// truncated stream: return only the bytes that exist so downstream
		// length validations reject the property instead of retaining a
		// giant zero-filled allocation
		return tempArr[:n]
	}
	return tempArr
}

func (c *Cursor) readProperty() (*Property, error) {
	if _, err := c.Peek(1); err != nil {
		return nil, errors.New("Reading Past End")
	}
	prop := Property{}
	typ, _, err := c.ReadRune()
	if err != nil {
		//fmt.Println(err)
	}
	prop.Type = PropertyType(typ)
	var val []byte
	// readFixed reads n bytes, rejecting truncated streams so the encoder
	// never sees a scalar property with a short payload
	readFixed := func(n int) ([]byte, error) {
		b := c.readBytes(n)
		if len(b) != n {
			return nil, fmt.Errorf("truncated %c property payload", typ)
		}
		return b, nil
	}
	switch prop.Type {
	case 'S':
		val, err = c.readLongStringBytes()
	case 'Y':
		val, err = readFixed(2)
	case 'C':
		val, err = readFixed(1)
	case 'I':
		val, err = readFixed(4)
	case 'F':
		val, err = readFixed(4)
	case 'D':
		val, err = readFixed(8)
	case 'L':
		val, err = readFixed(8)
	case 'R':
		tmp := c.readBytes(4)
		if len(tmp) != 4 {
			return nil, errors.New("truncated raw property header")
		}
		length := int(binary.LittleEndian.Uint32(tmp))
		if int64(length) > maxPropertyBytes {
			return nil, fmt.Errorf("ofbx: raw property length %d exceeds limit", length)
		}
		val = append(tmp, c.readBytes(length)...)
	case 'b', 'f', 'd', 'l', 'i', 'c':
		unCompressedLength := c.readBytes(4)
		encoding := c.readBytes(4)
		compressedLength := c.readBytes(4)
		if len(unCompressedLength) != 4 || len(encoding) != 4 || len(compressedLength) != 4 {
			return nil, errors.New("truncated array header")
		}
		prop.Encoding = binary.LittleEndian.Uint32(encoding)
		prop.compressedLength = binary.LittleEndian.Uint32(compressedLength)
		prop.Count = int(binary.LittleEndian.Uint32(unCompressedLength))
		length := int(prop.compressedLength)
		if prop.Encoding == 0 {
			// uncompressed: validate the claimed size before allocating
			if int64(prop.Count)*int64(prop.Type.Size()) > maxArrayBytes {
				return nil, fmt.Errorf("array of %d x %d bytes exceeds limit", prop.Count, prop.Type.Size())
			}
			length = prop.Count * prop.Type.Size()
		} else if prop.Encoding > 1 {
			return nil, fmt.Errorf("invalid array encoding %d", prop.Encoding)
		}
		val = c.readBytes(length)
		if len(val) != length {
			return nil, errors.New("truncated array payload")
		}
	default:
		return nil, errors.New("Did not know this property:" + string(prop.Type))
	}
	if err != nil {
		return nil, err
	}

	prop.value = NewDataViewBytes(val)

	return &prop, nil
}

func (c *Cursor) readElement(version uint16) (*Element, error) {
	v, _ := c.Peek(12)
	footer := true
	for _, b := range v {
		if b != 0 {
			footer = false
			break
		}
	}
	if footer {
		// Note we don't actually read the footer contents yet,
		// as far as we know the footer holds no useful information
		//fmt.Println("Returning footer")
		return nil, nil
	}

	endOffset, err := c.readElementOffset(version)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Obtained element end offset", endOffset)
	propCt, err := c.readElementOffset(version)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Obtained element prop count", propCt)
	_, err = c.readElementOffset(version)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Obtained property list length", prop_list_length)
	id, err := c.readShortString()
	if err != nil {
		return nil, err
	}
	//fmt.Println("Read short string", id)

	// sanity: a record must end at or after its header, and must not claim
	// an absurd number of properties; otherwise corrupt files cause huge
	// allocations or unterminated child loops.
	if propCt > maxElementProps {
		return nil, fmt.Errorf("ofbx: element %q claims %d properties", id, propCt)
	}
	if endOffset < uint64(c.ReadSoFar()) {
		return nil, fmt.Errorf("ofbx: element %q end offset %d precedes its header", id, endOffset)
	}

	element := Element{}
	element.ID = NewDataView(id)

	element.Properties = make([]*Property, propCt)
	for i := uint64(0); i < propCt; i++ {
		element.Properties[i], err = c.readProperty()
		if err != nil {
			return nil, err
		}
	}

	if uint64(c.ReadSoFar()) >= endOffset {
		//fmt.Println("NO Sentinel sizes ", c.ReadSoFar(), endOffset)
		return &element, nil
	}
	blockSentinelLength := 13
	if version >= 7500 {
		blockSentinelLength = 25
	}

	//fmt.Print("sizes pre children ", c.ReadSoFar(), endOffset, uint64(blockSentinelLength))
	for uint64(c.ReadSoFar()) < endOffset-uint64(blockSentinelLength) {
		child, err := c.readElement(version)
		if err != nil {
			return nil, fmt.Errorf("ReadingChild element failed"+": %w", err)
		}
		if child == nil {
			// stream ended inside this element's children
			return nil, fmt.Errorf("ofbx: element %q ends past end of data", id)
		}
		element.Children = append(element.Children, child)
		if uint64(c.ReadSoFar()) > endOffset {
			//fmt.Println("Read past where we were supposed to!!", c.ReadSoFar(), endOffset)
		}
	}
	if uint64(c.ReadSoFar()) > endOffset {
		//fmt.Println("Read past where we were supposed to!!", c.ReadSoFar(), endOffset)
	}
	c.Discard(blockSentinelLength)
	//fmt.Println("With Sentinel", uint64(c.ReadSoFar()), "versus", endOffset)
	return &element, nil
}

// sanity limits guarding against corrupt or hostile files: allocations are
// capped so that absurd length/count fields fail with an error instead of
// exhausting memory.
const (
	maxPropertyBytes = 1 << 28 // 256 MiB payload per property (raw/string)
	maxElementProps  = 1 << 24 // 16M properties per element
)

func tokenize(r io.Reader) (*Element, error) {
	countReader := NewCountReader(r)
	r2 := bufio.NewReader(countReader)
	cursor := &Cursor{r2, countReader}
	//fmt.Println("initial stats: ", r2.Buffered(), cursor.ReadSoFar())

	if !isBinaryFBX(r2) {
		return tokenizeText(cursor)
	}

	var header Header
	err := binary.Read(cursor, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	//fmt.Println(header)

	//fmt.Println("Header:", header)

	root := &Element{}

	for {
		//fmt.Println("Reading element")
		child, err := cursor.readElement(uint16(header.Version))
		if err != nil {
			//fmt.Println("Read element failure", err)
			return nil, err
		}

		if child == nil {
			return root, nil
		}
		root.Children = append(root.Children, child)
	}
}
