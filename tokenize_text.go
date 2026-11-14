package ofbx

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
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

func (c *Cursor) skipWhitespaces() error {
	for {
		by, _, err := c.ReadRune()
		if err != nil {
			return err
		}
		if unicode.IsSpace(by) {
			continue
		}
		c.UnreadRune()
		break
	}
	for {
		by, _, err := c.ReadRune()
		if err != nil {
			return err
		}
		if by == ';' {
			c.skipLine()
			continue
		}
		c.UnreadRune()
		break
	}
	return nil
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
			return nil, errors.Wrap(err, "ofbx: parsing text element")
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
				default:
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

	if unicode.IsDigit(r) || r == '-' || r == '+' {
		c.UnreadRune()
		return c.readTextNumber()
	}

	if r == '*' {
		return c.readTextArray()
	}
	return nil, errors.Errorf("ofbx: unexpected character %q in text property", string(r))
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
			return nil, errors.Wrapf(perr, "ofbx: bad float %q", text)
		}
		return NewFloat64Property(f), nil
	}
	i, perr := strconv.ParseInt(text, 10, 64)
	if perr != nil {
		return nil, errors.Wrapf(perr, "ofbx: bad integer %q", text)
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
		return nil, errors.Wrap(err, "ofbx: malformed array count")
	}
	if count < 0 || int64(count)*8 > maxArrayBytes {
		return nil, errors.Errorf("ofbx: text array count %d exceeds limit", count)
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
				return errors.Wrapf(perr, "ofbx: bad array integer %q", text)
			}
			ints = append(ints, v)
			doubles = append(doubles, float64(v)) // kept in sync in case a later value is a float
			return nil
		}
		isDouble = true
		v, perr := strconv.ParseFloat(text, 64)
		if perr != nil {
			return errors.Wrapf(perr, "ofbx: bad array float %q", text)
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
			return nil, errors.Errorf("ofbx: unexpected character %q in array literal", string(rr))
		}
		if len(ints) > count && len(doubles) > count {
			return nil, errors.Errorf("ofbx: array literal holds more than %d values", count)
		}
	}
done:
	if len(ints) != count && len(doubles) != count {
		return nil, errors.Errorf("ofbx: array claims %d values, found %d", count, len(doubles))
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
		if by[0] == '\n' || by[0] == '{' {
			break
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
