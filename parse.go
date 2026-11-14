package ofbx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// maxArrayBytes caps total decoded array payload; corrupt count fields fail
// with an error instead of a huge allocation. 16 MiB accommodates meshes of
// roughly 600k vertices (xyz doubles) with headroom.
const maxArrayBytes = 1 << 24 // 16 MiB

// validateArrayCount guards the decoded size of an array property.
func validateArrayCount(property *Property) error {
	if property.Count < 0 {
		return errors.New("ofbx: negative array count")
	}
	size := int64(property.Count) * int64(property.Type.Size())
	if size > maxArrayBytes {
		return fmt.Errorf("ofbx: array of %d x %d bytes exceeds limit", property.Count, property.Type.Size())
	}
	if property.Encoding == 0 {
		// uncompressed arrays must be fully backed by their payload; a lying
		// count would otherwise trigger a huge zero-filled allocation
		if size != int64(property.value.Len()) {
			return fmt.Errorf("ofbx: uncompressed array of %d elements is backed by %d payload bytes",
				property.Count, property.value.Len())
		}
	}
	return nil
}

func ParseTemplates(root *Element) {
	defs := findChildren(root, "Definitions")
	if len(defs) == 0 {
		return
	}

	templates := make(map[string]*Element)
	defs = defs[0].Children
	for _, def := range defs {
		if def.IDString() == "ObjectType" {
			prop1 := def.getProperty(0).value
			prop1Data, err := io.ReadAll(prop1)
			if err != nil && err != io.EOF {
				//fmt.Println(err)
				continue
			}
			subdefs := def.Children
			for _, subdef := range subdefs {
				if subdef.IDString() == "PropertyTemplate" {
					prop2 := subdef.getProperty(0).value
					prop2Data, err := io.ReadAll(prop2)
					if err != nil && err != io.EOF {
						//fmt.Println(err)
						continue
					}
					templates[string(prop1Data)+string(prop2Data)] = subdef
				}
			}

		}
	}
}

func parseBinaryArrayInt(property *Property) ([]int, error) {
	count := property.Count
	if count == 0 {
		return []int{}, nil
	}
	if !property.Type.IsArray() {
		return nil, errors.New("Invalid type")
	}
	return parseArrayRawInt(property)
}

func parseBinaryArrayFloat64(property *Property) ([]float64, error) {
	count := property.Count
	if count == 0 {
		return []float64{}, nil
	}
	if !property.Type.IsArray() {
		return nil, errors.New("Invalid type")
	}
	return parseArrayRawFloat64(property)
}

func parseBinaryArrayFloat32(property *Property) ([]float32, error) {
	f64s, err := parseBinaryArrayFloat64(property)
	if err != nil {
		return nil, err
	}
	f32s := make([]float32, len(f64s))
	for i, f64 := range f64s {
		f32s[i] = float32(f64)
	}
	return f32s, nil
}

func parseBinaryArrayVec2(property *Property) ([]Point2, error) {
	f64s, err := parseBinaryArrayFloat64(property)
	if err != nil {
		return nil, err
	}
	vs := make([]Point2, len(f64s)/2)
	for i := 0; i < len(f64s); i += 2 {
		vs[i/2][0] = f64s[i]
		vs[i/2][1] = f64s[i+1]
	}
	return vs, nil
}

func parseBinaryArrayVec3(property *Property) ([]Point3, error) {
	f64s, err := parseBinaryArrayFloat64(property)
	if err != nil {
		return nil, err
	}
	vs := make([]Point3, len(f64s)/3)
	// len(f64s) should probably be divisible by 3
	//if len(f64s)%3 != 0 {
	//fmt.Println("Vec3 binary array not made up of Point3s")
	//}
	for i := 0; (i + 2) < len(f64s); i += 3 {
		vs[i/3][0] = f64s[i]
		vs[i/3][1] = f64s[i+1]
		vs[i/3][2] = f64s[i+2]
	}
	return vs, nil
}
func parseBinaryArrayVec4(property *Property) ([]Point4, error) {
	f64s, err := parseBinaryArrayFloat64(property)
	if err != nil {
		return nil, err
	}
	vs := make([]Point4, len(f64s)/4)
	for i := 0; i < len(f64s); i += 4 {
		vs[i/4][0] = f64s[i]
		vs[i/4][1] = f64s[i+1]
		vs[i/4][2] = f64s[i+2]
		vs[i/4][3] = f64s[i+3]
	}
	return vs, nil
}

func parseArrayRawInt(property *Property) ([]int, error) {
	if property.Type == 'd' || property.Type == 'f' {
		return nil, errors.New("Invalid type, expected i or l")
	}
	if err := validateArrayCount(property); err != nil {
		return nil, err
	}
	data, err := arrayPayload(property)
	if err != nil {
		return nil, err
	}
	i64s := decodeInt64s(data, property.Type.Size())
	out := make([]int, len(i64s))
	for i, v := range i64s {
		out[i] = int(v)
	}
	return out, nil
}

// arrayPayload returns the raw (or inflated) payload bytes for an array
// property whose count was already validated.
func arrayPayload(property *Property) ([]byte, error) {
	if property.Encoding == 1 {
		return inflateArray(property)
	}
	return property.rawBytes(), nil
}

// decodeInt64s decodes little-endian integers from an exact payload.
func decodeInt64s(data []byte, elemSize int) []int64 {
	ln := len(data) / elemSize
	out := make([]int64, ln)
	if elemSize <= 1 {
		// bool/byte arrays widen each byte to one element
		for i := 0; i < ln; i++ {
			out[i] = int64(data[i])
		}
		return out
	}
	if elemSize == 4 {
		for i := 0; i < ln; i++ {
			out[i] = int64(int32(binary.LittleEndian.Uint32(data[i*4:])))
		}
		return out
	}
	for i := 0; i < ln; i++ {
		out[i] = int64(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return out
}

// decodeFloat64s decodes little-endian doubles/floats from an exact payload.
func decodeFloat64s(data []byte, elemSize int) []float64 {
	ln := len(data) / elemSize
	out := make([]float64, ln)
	if elemSize <= 1 {
		for i := 0; i < ln; i++ {
			out[i] = float64(data[i])
		}
		return out
	}
	if elemSize == 4 {
		for i := 0; i < ln; i++ {
			out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:])))
		}
		return out
	}
	for i := 0; i < ln; i++ {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return out
}

// decodeFloat32s decodes little-endian floats/doubles from an exact payload.
func decodeFloat32s(data []byte, elemSize int) []float32 {
	ln := len(data) / elemSize
	out := make([]float32, ln)
	if elemSize <= 1 {
		for i := 0; i < ln; i++ {
			out[i] = float32(data[i])
		}
		return out
	}
	if elemSize == 4 {
		for i := 0; i < ln; i++ {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
		}
		return out
	}
	for i := 0; i < ln; i++ {
		out[i] = float32(math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:])))
	}
	return out
}

// inflateArray decompresses a zlib array payload into a buffer sized for
// count elements; a short stream leaves the tail zeroed like a short read.
func inflateArray(p *Property) ([]byte, error) {
	out := make([]byte, p.Count*p.Type.Size())
	zr, err := zlib.NewReader(&p.value.Reader)
	if err != nil {
		return nil, fmt.Errorf("ofbx: inflating array: %w", err)
	}
	defer zr.Close()
	_, _ = io.ReadFull(zr, out)
	p.value.Seek(0, io.SeekStart)
	return out, nil
}

func parseArrayRawIntEnd(r io.Reader, ln int, elemSize int) []int {
	data, _ := io.ReadAll(r)
	i64s := decodeInt64s(exactBytes(data, ln, elemSize), elemSize)
	out := make([]int, len(i64s))
	for i, v := range i64s {
		out[i] = int(v)
	}
	return out
}

// exactBytes pads or trims data to exactly n elements.
func exactBytes(data []byte, ln, elemSize int) []byte {
	want := ln * elemSize
	if len(data) >= want {
		return data[:want]
	}
	padded := make([]byte, want)
	copy(padded, data)
	return padded
}

func parseArrayRawInt64(property *Property) ([]int64, error) {
	if property.Type == 'd' || property.Type == 'f' {
		return nil, errors.New("Invalid type, expected i or l")
	}
	if err := validateArrayCount(property); err != nil {
		return nil, err
	}
	data, err := arrayPayload(property)
	if err != nil {
		return nil, err
	}
	return decodeInt64s(data, property.Type.Size()), nil
}

func parseArrayRawInt64End(r io.Reader, ln int, elemSize int) []int64 {
	data, _ := io.ReadAll(r)
	return decodeInt64s(exactBytes(data, ln, elemSize), elemSize)
}

func parseArrayRawFloat32(property *Property) ([]float32, error) {
	if property.Type == 'i' || property.Type == 'l' {
		// see parseArrayRawFloat64: integer-valued float arrays re-read from
		// text FBX come back typed as ints
		ints, err := parseArrayRawInt64(property)
		if err != nil {
			return nil, err
		}
		out := make([]float32, len(ints))
		for i, v := range ints {
			out[i] = float32(v)
		}
		return out, nil
	}
	if err := validateArrayCount(property); err != nil {
		return nil, err
	}
	data, err := arrayPayload(property)
	if err != nil {
		return nil, err
	}
	return decodeFloat32s(data, property.Type.Size()), nil
}

func parseArrayRawFloat32End(r io.Reader, ln int, elemSize int) []float32 {
	data, _ := io.ReadAll(r)
	return decodeFloat32s(exactBytes(data, ln, elemSize), elemSize)
}

func parseArrayRawFloat64(property *Property) ([]float64, error) {
	if property.Type == 'i' || property.Type == 'l' {
		// text FBX re-parsing loses type hints: integer-valued float arrays
		// come back as int arrays; converting is lossless
		ints, err := parseArrayRawInt64(property)
		if err != nil {
			return nil, err
		}
		out := make([]float64, len(ints))
		for i, v := range ints {
			out[i] = float64(v)
		}
		return out, nil
	}
	if err := validateArrayCount(property); err != nil {
		return nil, err
	}
	data, err := arrayPayload(property)
	if err != nil {
		return nil, err
	}
	return decodeFloat64s(data, property.Type.Size()), nil
}

func parseArrayRawFloat64End(r io.Reader, ln int, elemSize int) []float64 {
	data, _ := io.ReadAll(r)
	return decodeFloat64s(exactBytes(data, ln, elemSize), elemSize)
}

func parseDoubleVecDataVec2(property *Property) ([]Point2, error) {
	if property.Type == 'd' {
		return parseBinaryArrayVec2(property)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 2
	outVec := make([]Point2, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		outVec[j][0] = float64(tmp[i])
		outVec[j][1] = float64(tmp[i+1])
	}
	return outVec, nil
}

func parseDoubleVecDataVec3(property *Property) ([]Point3, error) {
	if property.Type == 'd' {
		return parseBinaryArrayVec3(property)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 3
	outVec := make([]Point3, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		outVec[j][0] = float64(tmp[i])
		outVec[j][1] = float64(tmp[i+1])
		outVec[j][2] = float64(tmp[i+2])
	}
	return outVec, nil
}

func parseDoubleVecDataVec4(property *Property) ([]Point4, error) {
	if property.Type == 'd' {
		return parseBinaryArrayVec4(property)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 4
	outVec := make([]Point4, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		outVec[j][0] = float64(tmp[i])
		outVec[j][1] = float64(tmp[i+1])
		outVec[j][2] = float64(tmp[i+2])
		outVec[j][3] = float64(tmp[i+3])
	}
	return outVec, nil
}

func parseVertexDataVec2(element *Element, name, idxName string) ([]Point2, []int, VertexDataMapping, error) {
	idxs, mapping, dataProp, err := parseVertexDataInner(element, name, idxName)
	if err != nil {
		return nil, nil, mapping, err
	}
	vcs, err := parseDoubleVecDataVec2(dataProp)
	return vcs, idxs, mapping, err
}

func parseVertexDataVec3(element *Element, name, idxName string) ([]Point3, []int, VertexDataMapping, error) {
	idxs, mapping, dataProp, err := parseVertexDataInner(element, name, idxName)
	if err != nil {
		return nil, nil, mapping, err
	}
	vcs, err := parseDoubleVecDataVec3(dataProp)
	return vcs, idxs, mapping, err
}

func parseVertexDataVec4(element *Element, name, idxName string) ([]Point4, []int, VertexDataMapping, error) {
	idxs, mapping, dataProp, err := parseVertexDataInner(element, name, idxName)
	if err != nil {
		return nil, nil, mapping, err
	}
	vcs, err := parseDoubleVecDataVec4(dataProp)
	return vcs, idxs, mapping, err
}

func parseVertexDataInner(element *Element, name, idxName string) ([]int, VertexDataMapping, *Property, error) {
	dataProp := findChildProperty(element, name)
	if len(dataProp) == 0 {
		return nil, 0, nil, errors.New("Invalid data element")
	}
	mappingProp := findChildProperty(element, "MappingInformationType")
	referenceProp := findChildProperty(element, "ReferenceInformationType")

	var idxs []int
	var mapping VertexDataMapping
	var err error

	if len(mappingProp) != 0 {
		var ok bool
		mapping, ok = vtxDataMapFromStrs[mappingProp[0].value.String()]
		if !ok {
			return nil, 0, nil, errors.New("Unable to parse mapping")
		}
	}
	if len(referenceProp) != 0 {
		if referenceProp[0].value.String() == "IndexToDirect" {
			indicesProp := findChildProperty(element, idxName)
			if len(indicesProp) != 0 {
				if idxs, err = parseBinaryArrayInt(indicesProp[0]); err != nil {
					return nil, 0, nil, errors.New("Unable to parse indices")
				}
			} else {
				// just use indicies in order.
				idxs = nil
			}
		} else if referenceProp[0].value.String() != "Direct" {
			return nil, 0, nil, errors.New("Invalid properties")
		}
	}
	return idxs, mapping, dataProp[0], nil
}

func parseTexture(scene *Scene, element *Element) *Texture {
	texture := NewTexture(scene, element)
	assignSingleChildProperty(element, "FileName", &texture.filename)
	assignSingleChildProperty(element, "RelativeFilename", &texture.relativeFilename)
	return texture
}

func parseLimbNode(scene *Scene, element *Element) (*Node, error) {
	if prop := element.getProperty(2); prop == nil || prop.value.String() != "LimbNode" {
		return nil, errors.New("Invalid limb node")
	}
	return NewNode(scene, element, LIMB_NODE), nil
}

func parseMesh(scene *Scene, element *Element) (*Mesh, error) {
	if prop := element.getProperty(2); prop == nil || prop.value.String() != "Mesh" {
		return nil, errors.New("Invalid mesh")
	}
	return NewMesh(scene, element), nil
}

func parseMaterial(scene *Scene, element *Element) *Material {
	material := NewMaterial(scene, element)
	elems := findChildren(element, "Properties70")
	material.DiffuseColor = Color{1, 1, 1}
	if len(elems) == 0 {
		return material
	}
	elems = elems[0].Children
	// Todo: reflection / struct tags for these types of values
	for _, elem := range elems {
		if elem.getProperty(0) == nil {
			continue
		}
		v := elem.getProperty(0).value.String()
		// Commented out cases are things I (200sc) think might exist
		// but haven't seen
		switch v {
		case "EmissiveColor":
			material.EmissiveColor.R = float32(propFloat(elem.getProperty(4)))
			material.EmissiveColor.G = float32(propFloat(elem.getProperty(5)))
			material.EmissiveColor.B = float32(propFloat(elem.getProperty(6)))
		case "AmbientColor":
			material.AmbientColor.R = float32(propFloat(elem.getProperty(4)))
			material.AmbientColor.G = float32(propFloat(elem.getProperty(5)))
			material.AmbientColor.B = float32(propFloat(elem.getProperty(6)))
		case "DiffuseColor":
			material.DiffuseColor.R = float32(propFloat(elem.getProperty(4)))
			material.DiffuseColor.G = float32(propFloat(elem.getProperty(5)))
			material.DiffuseColor.B = float32(propFloat(elem.getProperty(6)))
		case "TransparentColor":
			material.TransparentColor.R = float32(propFloat(elem.getProperty(4)))
			material.TransparentColor.G = float32(propFloat(elem.getProperty(5)))
			material.TransparentColor.B = float32(propFloat(elem.getProperty(6)))
		case "SpecularColor":
			material.SpecularColor.R = float32(propFloat(elem.getProperty(4)))
			material.SpecularColor.G = float32(propFloat(elem.getProperty(5)))
			material.SpecularColor.B = float32(propFloat(elem.getProperty(6)))
		case "ReflectionColor":
			material.ReflectionColor.R = float32(propFloat(elem.getProperty(4)))
			material.ReflectionColor.G = float32(propFloat(elem.getProperty(5)))
			material.ReflectionColor.B = float32(propFloat(elem.getProperty(6)))
		case "EmissiveFactor":
			material.EmissiveFactor = propFloat(elem.getProperty(4))
		// case "AmbientFactor":
		case "DiffuseFactor":
			material.DiffuseFactor = propFloat(elem.getProperty(4))
		// case "TransparentFactor":
		case "SpecularFactor":
			material.SpecularFactor = propFloat(elem.getProperty(4))
		case "ReflectionFactor":
			material.ReflectionFactor = propFloat(elem.getProperty(4))
		case "Shininess":
			material.Shininess = propFloat(elem.getProperty(4))
		case "ShininessExponent":
			material.ShininessExponent = propFloat(elem.getProperty(4))
		}
	}
	return material
}

func parseAnimationCurve(scene *Scene, element *Element) (*AnimationCurve, error) {
	curve := NewAnimationCurve(scene, element)
	var err error
	if attrFlags := findSingleChildProperty(element, "KeyAttrFlags"); attrFlags != nil {
		curve.AttrFlags, err = attrFlags.getValuesInt64()
		if err != nil {
			return nil, fmt.Errorf("Invalid animation curve: attrFlags error"+": %w", err)
		}
	}
	if attrData := findSingleChildProperty(element, "KeyAttrDataFloat"); attrData != nil {
		curve.AttrData, err = attrData.getValuesF32()
		if err != nil {
			return nil, fmt.Errorf("Invalid animation curve: attrFlags error"+": %w", err)
		}
	}
	if attrRefCt := findSingleChildProperty(element, "KeyAttrRefCount"); attrRefCt != nil {
		curve.AttrRefCount, err = attrRefCt.getValuesInt64()
		if err != nil {
			return nil, fmt.Errorf("Invalid animation curve: attrFlags error"+": %w", err)
		}
	}
	if times := findSingleChildProperty(element, "KeyTime"); times != nil {
		intTimes, cerr := times.getValuesInt64()
		if cerr != nil {
			return nil, fmt.Errorf("Invalid animation curve: times error"+": %w", cerr)
		}
		curve.Times = make([]time.Duration, len(intTimes))
		for i, v := range intTimes {
			curve.Times[i] = fbxTimetoStdTime(v)
		}
	}
	if values := findSingleChildProperty(element, "KeyValueFloat"); values != nil {
		curve.Values, err = values.getValuesF32()
		if err != nil {
			return nil, errors.New("Invalid animation curve: values error")
		}
	}
	if len(curve.Times) != len(curve.Values) {
		return nil, errors.New("Invalid animation curve: len error")
	}
	return curve, nil
}

func parseConnection(root *Element, scene *Scene) (bool, error) {
	connections := findChildren(root, "Connections")
	if len(connections) == 0 {
		return true, nil
	}

	connections = connections[0].Children
	for _, connection := range connections {
		prop0 := connection.getProperty(0)
		prop1 := connection.getProperty(1)
		prop2 := connection.getProperty(2)
		if !isString(prop0) ||
			!isLong(prop1) ||
			!isLong(prop2) {
			return false, errors.New("Invalid connection")
		}
		var c Connection
		c.from = prop1.value.touint64()
		c.to = prop2.value.touint64()
		if prop0.value.String() == "OO" {
			c.typ = ObjectConn
		} else if prop0.value.String() == "OP" {
			c.typ = PropConn
			if prop3 := connection.getProperty(3); prop3 != nil {
				c.property = prop3.value.String()
			} else {
				return false, errors.New("Invalid connection")
			}
		} else {
			return false, errors.New("Not supported")
		}
		scene.Connections = append(scene.Connections, c)
	}
	return true, nil
}

func parseTakes(scene *Scene) (bool, error) {
	takes := findChildren(scene.RootElement, "Takes")
	if len(takes) == 0 {
		return true, nil
	}

	objects := takes[0].Children

	for _, object := range objects {
		if object.IDString() != "Take" {
			continue
		}
		if !isString(object.getProperty(0)) {
			return false, errors.New("Invalid name in take")
		}
		var take TakeInfo
		take.name = object.getProperty(0).value
		filename := findSingleChildProperty(object, "FileName")
		if filename != nil {
			if !isString(filename) {
				return false, errors.New("Invalid filename in take")
			}
			take.filename = filename.value
		}
		localTime := findChildProperty(object, "LocalTime")
		if len(localTime) != 0 {
			if !isLong(localTime[0]) || len(localTime) < 2 || !isLong(localTime[1]) {
				return false, errors.New("Invalid local time in take")
			}

			take.localTimeFrom = fbxTimeToSeconds(localTime[0].value.toint64())
			take.localTimeTo = fbxTimeToSeconds(localTime[1].value.toint64())
		}
		refTime := findChildProperty(object, "ReferenceTime")
		if len(refTime) != 0 {
			if !isLong(refTime[0]) || len(refTime) < 2 || !isLong(refTime[1]) {
				return false, errors.New("Invalid reference time in take")
			}
			take.refTimeFrom = fbxTimeToSeconds(refTime[0].value.toint64())
			take.refTimeTo = fbxTimeToSeconds(refTime[1].value.toint64())
		}
		scene.TakeInfos = append(scene.TakeInfos, take)
	}
	return true, nil
}

func parseGlobalSettings(root *Element, scene *Scene) {

	for _, settings := range root.Children {
		if settings.IDString() != "GlobalSettings" {
			continue
		}
		for _, props70 := range settings.Children {
			if props70.IDString() != "Properties70" {
				continue
			}
			for _, node := range props70.Children {
				p := node.getProperty(0)
				if p == nil {
					continue
				}
				prop4 := node.getProperty(4)
				if prop4 == nil {
					continue
				}
				value := prop4.value

				switch p.value.String() {
				case "UpAxis":
					scene.Settings.UpAxis = UpVector(propInt(prop4))
				case "UpAxisSign":
					scene.Settings.UpAxisSign = propInt(prop4)
				case "FrontAxis":
					scene.Settings.FrontAxis = FrontVector(propInt(prop4))
				case "FrontAxisSign":
					scene.Settings.FrontAxisSign = propInt(prop4)
				case "CoordAxis":
					scene.Settings.CoordAxis = CoordSystem(propInt(prop4))
				case "CoordAxisSign":
					scene.Settings.CoordAxisSign = propInt(prop4)
				case "OriginalUpAxis":
					scene.Settings.OriginalUpAxis = propInt(prop4)
				case "OriginalUpAxisSign":
					scene.Settings.OriginalUpAxisSign = propInt(prop4)
				case "UnitScaleFactor":
					scene.Settings.UnitScaleFactor = float32(propFloat(prop4))
				case "OriginalUnitScaleFactor":
					scene.Settings.OriginalUnitScaleFactor = float32(propFloat(prop4))
				case "TimeSpanStart":
					scene.Settings.TimeSpanStart = value.touint64()
				case "TimeSpanStop":
					scene.Settings.TimeSpanStop = value.touint64()
				case "TimeMode":
					scene.Settings.TimeMode = FrameRate(propInt(prop4))
				case "CustomFrameRate":
					scene.Settings.CustomFrameRate = float32(propFloat(prop4))
				}
			}
			break
		}
		break
	}
	scene.FrameRate = GetFramerateFromTimeMode(scene.Settings.TimeMode, scene.Settings.CustomFrameRate)
}

func parseObjects(root *Element, scene *Scene) (bool, error) {
	//fmt.Println("Starting object Parse")
	objs := findChildren(root, "Objects")
	if len(objs) == 0 {
		return true, nil
	}
	scene.RootNode = NewNode(scene, root, ROOT)
	scene.ObjectMap[0] = scene.RootNode

	objs = objs[0].Children
	for _, elem := range objs {
		if !isLong(elem.getProperty(0)) {
			return false, errors.New("Invalid")
		}
		id := elem.getProperty(0).value.touint64()

		var obj Obj
		var err error
		// This shouldn't happen?
		// Original library had a check like this but it seems nonsensical
		if id == 0 {
			continue
		}
		switch elem.IDString() {
		case "Geometry":
			lastProp := elem.getProperty(len(elem.Properties) - 1)
			if lastProp != nil && lastProp.value.String() == "Mesh" {
				obj, err = parseGeometry(scene, elem)
				if err != nil {
					return false, err
				}
			} else if lastProp != nil && lastProp.value.String() == "Shape" {
				obj, err = parseShape(scene, elem)
				if err != nil {
					return false, err
				}
			}
		case "Material":
			obj = parseMaterial(scene, elem)
		case "AnimationStack":
			obj = NewAnimationStack(scene, elem)
			stack := obj.(*AnimationStack)
			scene.AnimationStacks = append(scene.AnimationStacks, stack)
		case "AnimationLayer":
			obj = NewAnimationLayer(scene, elem)
		case "AnimationCurve":
			obj, err = parseAnimationCurve(scene, elem)
			if err != nil {
				return false, err
			}
		case "AnimationCurveNode":
			obj = NewAnimationCurveNode(scene, elem)
		case "Deformer":
			classProp := elem.getProperty(2)
			if classProp != nil {
				v := classProp.value.String()
				switch v {
				case "Cluster":
					obj, err = parseCluster(scene, elem)
					if err != nil {
						return false, err
					}
				case "Skin":
					obj = NewSkin(scene, elem)
				case "BlendShape":
					obj = NewBlendShape(scene, elem)
				case "BlendShapeChannel":
					obj = parseBlendShapeChannel(scene, elem)
				}
			}
		case "NodeAttribute":
			obj, err = parseNodeAttribute(scene, elem)
			if err != nil {
				return false, err
			}
		case "Model":
			classProp := elem.getProperty(2)
			if classProp != nil {
				v := classProp.value.String()
				switch v {
				case "Mesh":
					obj, err = parseMesh(scene, elem)
					if err == nil {
						mesh := obj.(*Mesh)
						scene.Meshes = append(scene.Meshes, mesh)
						obj = mesh
					}
				case "LimbNode":
					obj, err = parseLimbNode(scene, elem)
					if err != nil {
						return false, err
					}
				case "Null", "Root":
					obj = NewNode(scene, elem, NULL_NODE)
				case "Camera":
					obj = NewCamera(scene, elem)
					scene.Cameras = append(scene.Cameras, obj.(*Camera))
				case "Light":
					obj = NewLight(scene, elem)
					scene.Lights = append(scene.Lights, obj.(*Light))
				}
			}
		case "Texture":
			obj = parseTexture(scene, elem)
		case "Video":
			obj = parseVideo(scene, elem)
			if obj != nil {
				scene.Videos = append(scene.Videos, obj.(*Video))
			}
		}

		scene.ObjectMap[id] = obj
		if obj != nil {
			obj.SetID(id)
		}
	}

	//fmt.Println("Parsing connections")
	for _, con := range scene.Connections {
		con := con
		parent := scene.ObjectMap[con.to]
		child := scene.ObjectMap[con.from]
		if child == nil || parent == nil {
			continue
		}

		ctyp := child.Type()

		switch ctyp {
		case NODE_ATTRIBUTE:
			if parent.NodeAttribute() != nil {
				return false, errors.New("Invalid node attribute")
			}
			parent.SetNodeAttribute(child) //previously asserted that the child was a nodeattribute
		case ANIMATION_CURVE_NODE:
			if parent.IsNode() {
				node := child.(*AnimationCurveNode)
				node.Bone = parent
				node.BoneLinkProp = con.property
			}
		}

		switch parent.Type() {
		case MESH:
			mesh := parent.(*Mesh)
			switch ctyp {
			case GEOMETRY:
				if mesh.Geometry != nil {
					return false, errors.New("Invalid mesh")
				}
				mesh.Geometry = child.(*Geometry)
			case MATERIAL:
				mesh.Materials = append(mesh.Materials, (child.(*Material)))
			}
		case GEOMETRY:
			geom := parent.(*Geometry)
			switch ctyp {
			case SKIN:
				geom.Skin = child.(*Skin)
			case BLEND_SHAPE:
				geom.BlendShapes = append(geom.BlendShapes, child.(*BlendShape))
			}
		case BLEND_SHAPE:
			if ctyp == BLEND_SHAPE_CHANNEL {
				bs := parent.(*BlendShape)
				bs.Channels = append(bs.Channels, child.(*BlendShapeChannel))
			}
		case BLEND_SHAPE_CHANNEL:
			if ctyp == SHAPE {
				ch := parent.(*BlendShapeChannel)
				ch.Shapes = append(ch.Shapes, child.(*Shape))
			}
		case TEXTURE:
			if ctyp == VIDEO {
				tex := parent.(*Texture)
				tex.Media = child.(*Video)
			}
		case SKIN:
			skin := parent.(*Skin)
			if ctyp == CLUSTER {
				cluster := child.(*Cluster)
				skin.Clusters = append(skin.Clusters, cluster)
				if cluster.Skin != nil {
					return false, errors.New("Cluster assigned to multiple skins")
				}
				cluster.Skin = skin
			}
		case MATERIAL:
			mat := parent.(*Material)
			if ctyp == TEXTURE {
				ttyp := TextureCOUNT
				switch con.property {
				case "NormalMap":
					ttyp = NORMAL
				case "DiffuseColor":
					ttyp = DIFFUSE
				}
				if ttyp == TextureCOUNT {
					break
				}
				if mat.Textures[ttyp] != nil {
					break
				}
				mat.Textures[ttyp] = child.(*Texture)
			}
		case CLUSTER:
			cluster := parent.(*Cluster)
			if ctyp == LIMB_NODE || ctyp == MESH || ctyp == NULL_NODE {
				if cluster.Link != nil {
					return false, errors.New("Invalid cluster")
				}
				cluster.Link = child
			}

		case ANIMATION_LAYER:
			if ctyp == ANIMATION_CURVE_NODE {
				p := parent.(*AnimationLayer)
				p.CurveNodes = append(p.CurveNodes, child.(*AnimationCurveNode))
			}

		case ANIMATION_CURVE_NODE:
			node := parent.(*AnimationCurveNode)
			if ctyp == ANIMATION_CURVE {
				if node.Curves[0].Curve == nil {
					node.Curves[0].connection = &con
					node.Curves[0].Curve = child.(*AnimationCurve)
				} else if node.Curves[1].Curve == nil {
					node.Curves[1].connection = &con
					node.Curves[1].Curve = child.(*AnimationCurve)
				} else if node.Curves[2].Curve == nil {
					node.Curves[2].connection = &con
					node.Curves[2].Curve = child.(*AnimationCurve)
				} else {
					return false, errors.New("Invalid animation node")
				}
			}
		}
	}

	for _, obj := range scene.ObjectMap {
		if obj == nil {
			continue
		}
		if ppr, ok := obj.(NeedsPostProcessing); ok {
			if !ppr.postProcess() {
				return false, errors.New("Failed to postprocess object" + fmt.Sprintf("%v", obj.ID()))
			}
		}
	}

	return true, nil
}

// NeedsPostProcessing note objects that require post processing
type NeedsPostProcessing interface {
	postProcess() bool
}

// parseArrayRawBytes decodes a 'c' (byte) array property, inflating zlib
// payloads when encoded.
func parseArrayRawBytes(property *Property) ([]byte, error) {
	if property.Type != ArrayBYTE {
		return nil, fmt.Errorf("ofbx: byte array expected, got %q", string(property.Type))
	}
	if err := validateArrayCount(property); err != nil {
		return nil, err
	}
	data := property.rawBytes()
	if property.Encoding == 1 {
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("ofbx: inflating byte array: %w", err)
		}
		defer zr.Close()
		out, err := io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("ofbx: inflating byte array: %w", err)
		}
		if len(out) != property.Count {
			return nil, fmt.Errorf("ofbx: byte array inflated to %d bytes, want %d", len(out), property.Count)
		}
		return out, nil
	}
	if len(data) != property.Count {
		return nil, fmt.Errorf("ofbx: byte array holds %d bytes, want %d", len(data), property.Count)
	}
	return data, nil
}
