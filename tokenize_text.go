package ofbx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"
)

func (c *Cursor) IsEndLine() bool {
	by, err := c.Peek(1)
	if err != nil {
		//fmt.Println(err)
	}
	return by[0] == '\n'
}

// TODO: Make a isspace for bytes not unicode
func (c *Cursor) skipInsignificantWhitespaces() error {
	for {
		by, _, err := c.ReadRune()
		if err != nil {
			return err
		}
		if unicode.IsSpace(by) && by != '\n' {
			continue
		}
		c.UnreadRune()
		break
	}
	return nil
}

func (c *Cursor) skipLine() {
	c.ReadBytes('\n')
	c.skipInsignificantWhitespaces()
}

// skipWhitespaces advances to the next significant token character,
// treating whitespace (newlines included) and ';'-comment lines as noise.
func (c *Cursor) skipWhitespaces() error {
	for {
		by, _, err := c.ReadRune()
		if err != nil {
			return err
		}
		if by == ';' {
			c.skipLine()
			continue
		}
		if unicode.IsSpace(by) {
			continue
		}
		c.UnreadRune()
		return nil
	}
}

func isTextTokenChar(c rune) bool {
	return unicode.IsDigit(c) || unicode.IsLetter(c) || c == '_'
}

// tokenizeText parses a whole ASCII FBX stream into an element tree.
func tokenizeText(cursor *Cursor) (*Element, error) {
	root := &Element{}
	for {
		if err := cursor.skipWhitespaces(); err != nil {
			// EOF (or a stream of trailing whitespace) ends the document
			break
		}
		child, err := cursor.ReadTextElement()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("ofbx: parsing text element"+": %w", err)
		}
		if child == nil {
			break
		}
		root.Children = append(root.Children, child)
	}
	if len(root.Children) == 0 {
		return nil, errors.New("ofbx: no elements found in text FBX input")
	}
	return root, nil
}

func (c *Cursor) readTextToken() (*DataView, error) {
	out := bytes.NewBuffer([]byte{})
	for {
		r, _, err := c.ReadRune()
		if err != nil {
			return nil, err
		}
		if isTextTokenChar(r) {
			out.WriteRune(r)
			continue
		}
		c.UnreadRune()
		break
	}
	return BufferDataView(out), nil
}

func (c *Cursor) readTextProperty() (*Property, error) {
	prop := &Property{}

	r, _, err := c.ReadRune()
	if err != nil {
		return nil, err
	}
	if r == '"' {
		prop.Type = STRING
		val := bytes.NewBuffer([]byte{})
		for {
			rr, _, cerr := c.ReadRune()
			if cerr != nil {
				if cerr == io.EOF {
					break //?
				}
				return nil, cerr
			}
			if rr == '\\' {
				// escape sequence inside a string
				esc, _, eerr := c.ReadRune()
				if eerr != nil {
					return nil, eerr
				}
				switch esc {
				case 'n':
					val.WriteByte('\n')
				case 'r':
					val.WriteByte('\r')
				case 't':
					val.WriteByte('\t')
				case '"':
					val.WriteByte('"')
				case '\\':
					val.WriteByte('\\')
				case 'x':
					// hex escape \xHH for control bytes (our own writer's
					// convention). Autodesk writes unescaped backslashes, so
					// only commit to the escape when two hex digits follow.
					h1, _, e1 := c.ReadRune()
					if e1 != nil {
						return nil, e1
					}
					h2, _, e2 := c.ReadRune()
					if e2 != nil {
						return nil, e2
					}
					hi, ok1 := hexDigit(byte(h1))
					lo, ok2 := hexDigit(byte(h2))
					if !ok1 || !ok2 {
						// not an escape after all: literal backslash + x
						val.WriteByte('\\')
						val.WriteByte('x')
						val.WriteRune(h1)
						val.WriteRune(h2)
						continue
					}
					val.WriteByte(hi<<4 | lo)
				default:
					// Autodesk does not escape backslashes in paths:
					// keep them verbatim
					val.WriteByte('\\')
					val.WriteRune(esc)
				}
				continue
			}
			if rr == '"' {
				break
			}
			val.WriteRune(rr)
		}
		prop.value = BufferDataView(val)
		return prop, nil
	}

	if r == 'T' || r == 'Y' {
		// ASCII boolean literals (True/Yes); False/No are rejected below in
		// their own branch
		prop.Type = BOOL
		prop.value = NewDataView("\x01")
		return prop, nil
	}
	if r == 'F' || r == 'N' {
		prop.Type = BOOL
		prop.value = NewDataView("\x00")
		return prop, nil
	}

	if r == ',' {
		// Autodesk writes `Content: ,"<base64>"`: the bare comma is an
		// empty placeholder property (an int 0 in binary files; some
		// third-party importers reject empty byte arrays)
		return NewInt32Property(0), nil
	}
	if unicode.IsDigit(r) || r == '-' || r == '+' {
		c.UnreadRune()
		return c.readTextNumber()
	}

	if r == '*' {
		return c.readTextArray()
	}
	return nil, fmt.Errorf("ofbx: unexpected character %q in text property", string(r))
}

// readTextNumber parses an optionally signed decimal with optional fraction
// and exponent into an 'L' (integer) or 'D' (double) property.
func (c *Cursor) readTextNumber() (*Property, error) {
	val := bytes.NewBuffer([]byte{})
	isDouble := false

	digits := func() error {
		for {
			rr, _, err := c.ReadRune()
			if err != nil {
				if err == io.EOF {
					return io.EOF
				}
				return err
			}
			if !unicode.IsDigit(rr) {
				c.UnreadRune()
				return nil
			}
			val.WriteRune(rr)
		}
	}

	rr, _, err := c.ReadRune()
	if err != nil {
		return nil, err
	}
	if rr != '-' && rr != '+' {
		c.UnreadRune()
	} else {
		val.WriteRune(rr)
	}
	if err := digits(); err == io.EOF && val.Len() <= 1 {
		return nil, errors.New("ofbx: malformed number in text property")
	}

	rr, _, err = c.ReadRune()
	if err == nil && rr == '.' {
		isDouble = true
		val.WriteRune('.')
		if err := digits(); err == io.EOF {
			return nil, io.EOF
		}
		rr, _, err = c.ReadRune()
	}
	if err == nil && (rr == 'e' || rr == 'E') {
		// exponent: e[+-]digits
		isDouble = true
		val.WriteRune('e')
		nr, _, eerr := c.ReadRune()
		if eerr != nil {
			return nil, eerr
		}
		if nr == '-' || nr == '+' {
			val.WriteRune(nr)
			nr, _, eerr = c.ReadRune()
			if eerr != nil {
				return nil, eerr
			}
		}
		c.UnreadRune()
		if err := digits(); err == io.EOF {
			return nil, io.EOF
		}
	} else if err == nil {
		c.UnreadRune()
	}

	text := val.String()
	if isDouble {
		f, perr := strconv.ParseFloat(text, 64)
		if perr != nil {
			return nil, fmt.Errorf("ofbx: bad float %q"+": %w", text, perr)
		}
		return NewFloat64Property(f), nil
	}
	i, perr := strconv.ParseInt(text, 10, 64)
	if perr != nil {
		return nil, fmt.Errorf("ofbx: bad integer %q"+": %w", text, perr)
	}
	// FBX SDK convention: ints within 32-bit range serialize as 'I'; ids
	// (restored to 'L' by fixupIDs below) and KTime values exceed it
	if i >= math.MinInt32 && i <= math.MaxInt32 {
		return NewInt32Property(int32(i)), nil
	}
	return NewInt64Property(i), nil
}

// readTextArray parses `*N { a: v1,v2,... }` into a binary-backed array
// property ('i' or 'd') exactly like the binary tokenizer would produce.
func (c *Cursor) readTextArray() (*Property, error) {
	// element count
	countBuf := bytes.NewBuffer([]byte{})
	for {
		rr, _, err := c.ReadRune()
		if err != nil {
			return nil, err
		}
		if unicode.IsDigit(rr) {
			countBuf.WriteRune(rr)
			continue
		}
		c.UnreadRune()
		break
	}
	count, err := strconv.Atoi(countBuf.String())
	if err != nil {
		return nil, fmt.Errorf("ofbx: malformed array count"+": %w", err)
	}
	if count < 0 || int64(count)*8 > maxArrayBytes {
		return nil, fmt.Errorf("ofbx: text array count %d exceeds limit", count)
	}

	if err := c.skipWhitespaces(); err != nil {
		return nil, err
	}
	rr, _, err := c.ReadRune()
	if err != nil {
		return nil, err
	}
	if rr != '{' {
		return nil, errors.New("ofbx: expected '{' after array count")
	}
	if err := c.skipWhitespaces(); err != nil {
		return nil, err
	}
	// "a:" prefix (some writers use "a:" only)
	rr, _, err = c.ReadRune()
	if err != nil {
		return nil, err
	}
	if rr != 'a' {
		return nil, errors.New("ofbx: expected 'a:' in array literal")
	}
	rr, _, err = c.ReadRune()
	if err != nil {
		return nil, err
	}
	if rr != ':' {
		return nil, errors.New("ofbx: expected 'a:' in array literal")
	}
	c.skipInsignificantWhitespaces()

	ints := make([]int64, 0, count)
	doubles := make([]float64, 0, count)
	isDouble := false
	numBuf := bytes.NewBuffer([]byte{})
	flush := func() error {
		defer numBuf.Reset()
		if numBuf.Len() == 0 {
			return nil
		}
		text := numBuf.String()
		if !isDouble && !strings.ContainsAny(text, ".eE") {
			v, perr := strconv.ParseInt(text, 10, 64)
			if perr != nil {
				return fmt.Errorf("ofbx: bad array integer %q"+": %w", text, perr)
			}
			ints = append(ints, v)
			doubles = append(doubles, float64(v)) // kept in sync in case a later value is a float
			return nil
		}
		isDouble = true
		v, perr := strconv.ParseFloat(text, 64)
		if perr != nil {
			return fmt.Errorf("ofbx: bad array float %q"+": %w", text, perr)
		}
		doubles = append(doubles, v)
		return nil
	}

	for {
		rr, _, err := c.ReadRune()
		if err != nil {
			return nil, err
		}
		switch {
		case rr == ',' || unicode.IsSpace(rr):
			if err := flush(); err != nil {
				return nil, err
			}
		case rr == '}':
			if err := flush(); err != nil {
				return nil, err
			}
			goto done
		case rr == '-' || rr == '+' || unicode.IsDigit(rr) || rr == '.' || rr == 'e' || rr == 'E':
			numBuf.WriteRune(rr)
		default:
			return nil, fmt.Errorf("ofbx: unexpected character %q in array literal", string(rr))
		}
		if len(ints) > count && len(doubles) > count {
			return nil, fmt.Errorf("ofbx: array literal holds more than %d values", count)
		}
	}
done:
	if len(ints) != count && len(doubles) != count {
		return nil, fmt.Errorf("ofbx: array claims %d values, found %d", count, len(doubles))
	}
	if isDouble {
		return NewDoubleArrayProperty(doubles, false)
	}
	i64s := make([]int64, count)
	copy(i64s, ints)
	return NewInt64ArrayProperty(i64s, false)
}

func (c *Cursor) ReadTextElement() (*Element, error) {
	//fmt.Println("Read text token start")
	id, err := c.readTextToken()
	if err != nil {
		return nil, err
	}
	//fmt.Println("Read rune start")
	r, _, err := c.ReadRune()
	if err != nil {
		return nil, err
	}
	//fmt.Println("Read rune complete")
	if r != ':' {
		return nil, errors.New("Unexpected end of file")
	}
	// skip spaces, tabs and comment lines after the colon, but keep the
	// newline significant: `Name:\n` terminates a property-less element
	c.skipSpacesAndComments()

	element := &Element{}
	element.ID = id

	//fmt.Println("Looping over properties")
	for {
		//fmt.Println("Property loop")
		by, cerr := c.Peek(1)
		if cerr != nil {
			if cerr == io.EOF {
				break
			}
			return nil, cerr
		}
		if by[0] == '{' {
			break
		}
		if by[0] == '\n' {
			// a newline ends the property list unless a value continues on
			// the next line (Autodesk writes `Content: ,` + a base64
			// string on the following line)
			if !c.textPropsContinue() {
				break
			}
		}
		prop, cerr := c.readTextProperty()
		if cerr != nil {
			return nil, cerr
		}
		by, err = c.Peek(1)
		if err != io.EOF {
			if err != nil {
				return nil, err
			}
			if by[0] == ',' {
				c.Discard(1)
				c.skipWhitespaces()
			}
		}
		c.skipInsignificantWhitespaces()

		element.Properties = append(element.Properties, prop)
	}

	fixupIDs(element)
	rematerializeP(element)

	r, _, err = c.ReadRune()
	if err != nil {
		return nil, err
	}
	switch r {
	case '{':
		c.skipWhitespaces()
		for {
			by, err := c.Peek(1)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			if by[0] == '}' {
				c.Discard(1)
				break
			}
			child, err := c.ReadTextElement()
			if err != nil {
				return nil, err
			}
			c.skipWhitespaces()

			element.Children = append(element.Children, child)
		}
	case '\n':
		// newline terminates a property-less element; consume it
	default:
		c.UnreadRune()
	}
	return element, nil
}

// skipSpacesAndComments consumes blanks, tabs and ';'-comment lines without
// touching newlines, which stay meaningful as element terminators.
func (c *Cursor) skipSpacesAndComments() {
	for {
		by, err := c.Peek(1)
		if err != nil {
			return
		}
		switch by[0] {
		case ' ', '\t', '\r':
			c.Discard(1)
		case ';':
			c.skipLine()
		default:
			return
		}
	}
}

// textPropsContinue reports whether a property value continues after the
// line break at the cursor, consuming the intervening whitespace when it
// does. Peek never advances, so a negative answer leaves the cursor on the
// newline itself.
func (c *Cursor) textPropsContinue() bool {
	for k := 2; ; k++ {
		ahead, err := c.Peek(k)
		if err != nil || len(ahead) < k {
			return false
		}
		last := ahead[k-1]
		switch {
		case unicode.IsSpace(rune(last)):
			continue // keep looking past the whitespace
		case last == '"' || last == '*' || last == ',' || last == '-' || last == '+' || unicode.IsDigit(rune(last)):
			// value continues: consume the peeked whitespace (but not last)
			for i := 0; i < k-1; i++ {
				c.Discard(1)
			}
			return true
		default:
			return false
		}
	}
}

// hexDigit decodes one hex digit.
func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// fixupIDs restores 64-bit 'L' materialization for object and connection
// ids: importers parse ids through 64-bit-only readers while every other
// int is expected as 'I'.
func fixupIDs(element *Element) {
	widen := func(p *Property) *Property {
		if p != nil && p.Type == INTEGER {
			return NewInt64Property(int64(p.ValueI64()))
		}
		return p
	}
	switch element.IDString() {
	case "C":
		// C: "OO", from, to[, property]
		if len(element.Properties) >= 3 {
			element.Properties[1] = widen(element.Properties[1])
			element.Properties[2] = widen(element.Properties[2])
		}
	default:
		if textObjectElements[element.IDString()] && len(element.Properties) > 0 {
			// first property of an object element is its id
			element.Properties[0] = widen(element.Properties[0])
		}
	}
}

// rematerializeP rewrites the value properties of a "P" entry according to
// its type descriptor: importers parse P values with width-specific
// readers, and the ASCII dialect loses that information (an integer
// spelled "1" may be an int or a double).
func rematerializeP(element *Element) {
	if element.IDString() != "P" || len(element.Properties) < 5 {
		return
	}
	if element.Properties[0].Type != STRING || element.Properties[1].Type != STRING {
		return
	}
	pt := element.Properties[1].value.String()
	var narrow, ktime bool
	switch pt {
	case "int", "enum", "bool", "Visibility":
		narrow = true // 'I' within 32 bits
	case "KTime", "Time", "DateTime":
		ktime = true // keep 'L' materialization
	default:
		// every other numeric-ish type ("double", "Number", "Vector3D",
		// "Lcl Translation", "Color", ...) reads as F/D in importers,
		// including integer literals like a plain 0
		narrow = false
	}
	widenFloat := !narrow && !ktime
	for i := 4; i < len(element.Properties); i++ {
		p := element.Properties[i]
		switch {
		case narrow && p.Type == LONG:
			if v := p.ValueI64(); v >= math.MinInt32 && v <= math.MaxInt32 {
				element.Properties[i] = NewInt32Property(int32(v))
			}
		case widenFloat && (p.Type == LONG || p.Type == INTEGER):
			element.Properties[i] = NewFloat64Property(p.ValueF64())
		case ktime && p.Type == INTEGER:
			// KTime values parse as 64-bit in importers
			element.Properties[i] = NewInt64Property(p.ValueI64())
		}
	}
}

// textObjectElements lists Objects-section element names whose first
// property is an object id.
var textObjectElements = map[string]bool{
	"Geometry":           true,
	"Material":           true,
	"Model":              true,
	"Texture":            true,
	"Video":              true,
	"NodeAttribute":      true,
	"Deformer":           true,
	"AnimationStack":     true,
	"AnimationLayer":     true,
	"AnimationCurve":     true,
	"AnimationCurveNode": true,
}
