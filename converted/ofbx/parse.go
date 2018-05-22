package ofbx

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

func parseTemplates(root *Element) {
	const Element *defs = findChild(root, "Definitions")
	if defs == nil {
		return
	}

	templates := make(mao[string] * Element)
	Element * def = defs.child
	for def != nil {
		if def.id == "ObjectType" {
			prop1 := def.first_property.value
			prop1Data, err := ioutil.ReadAll(prop1)
			if err != nil && err != io.EOF {
				fmt.Println(err)
				def = def.sibling
				continue
			}
			Element * subdef = def.child
			for subdef != nil {
				if subdef.id == "PropertyTemplate" {
					prop2 := subdef.first_property.value
					prop2Data, err := ioutil.ReadAll(prop2)
					if err != nil && err != io.EOF {
						fmt.Println(err)
						subdef = subdef.sibling
						continue
					}
					templates[string(prop1Data)+string(prop2Data)] = subdef
				}
				subdef = subdef.sibling
			}
		}
		def = def.sibling
	}
}

func parseBinaryArrayInt(property *Property) ([]int, error) {
	count := property.getCount()
	if count == 0 {
		return []int{}, nil
	}
	elem_size := 1
	switch property.typ {
	case 'd':
		elem_size = 8
	case 'f':
		elem_size = 4
	case 'i':
		elem_size = 4
	default:
		return nil, false
	}
	elem_count := 4 / elem_size
	return parseArrayRawInt(property, count/elem_count)
}
func parseBinaryArrayFloat64(property *Property) ([]float64, error) {
	count := property.getCount()
	if count == 0 {
		return []float64{}, nil
	}
	elem_size := 1
	switch property.typ {
	case 'd':
		elem_size = 8
	case 'f':
		elem_size = 4
	case 'i':
		elem_size = 4
	default:
		return nil, false
	}
	elem_count := 4 / elem_size
	return parseArrayRawFloat64(property, count/elem_count)
}

func parseArrayRawInt(property *Property, max_size int) ([]int, error) {
	if property.typ == 'd' || property.typ == 'f' {
		return nil, errors.New("Invalid type, expected i or l")
	}
	elem_size := 4
	if property.typ == 'l' {
		elem_size = 8
	}
	count := property.getCount()
	var enc uint32
	binary.Read(property.value, binary.BigEndian, &enc)
	var ln uint32
	binary.Read(property.value, binary.BigEndian, &ln)

	if enc == 0 {
		if ln > max_size {
			return nil, errors.New("Max size too small for array")
		}
		return parseArrayRawIntEnd(property.value, ln, elem_size), nil
	} else if enc == 1 {
		if ln > max_size {
			return nil, errors.New("Max size too small for array")
		}
		zr := zip.NewReader(property.value, elem_size*count)
		defer zr.Close()
		// Assuming right now that zips only have one file to read
		fr, err := zr.Files[0].Open()
		if err != nil {
			return nil, err
		}
		defer fr.Close()
		return parseArrayRawIntEnd(property.value, ln, elem_size), nil
	}
	return nil, errors.New("Invalid encoding")
}

func parseArrayRawIntEnd(r io.Reader, ln, elem_size int) []int {
	out := make([]int, ln/elem_size)
	if elem_size == 4 {
		for i := 0; i < len(out); i++ {
			var v int
			binary.Read(property.value, binary.BigEndian, &v)
			out[i] = v
		}
	} else {
		for i := 0; i < len(out); i++ {
			var v int64
			binary.Read(property.value, binary.BigEndian, &v)
			out[i] = int(v)
		}
	}
	return out
}

func parseArrayRawFloat64(property *Property, max_size int) ([]float64, error) {
	if property.typ == 'i' || property.typ == 'l' {
		return nil, errors.New("Invalid type, expected d or f")
	}
	elem_size := 4
	if property.typ == 'd' {
		elem_size = 8
	}
	count := property.getCount()
	var enc uint32
	binary.Read(property.value, binary.BigEndian, &enc)
	var ln uint32
	binary.Read(property.value, binary.BigEndian, &ln)

	if enc == 0 {
		if ln > max_size {
			return nil, errors.New("Max size too small for array")
		}
		return parseArrayRawFloat64End(property.value, ln, elem_size), nil
	} else if enc == 1 {
		if ln > max_size {
			return nil, errors.New("Max size too small for array")
		}
		zr := zip.NewReader(property.value, elem_size*count)
		defer zr.Close()
		// Assuming right now that zips only have one file to read
		fr, err := zr.Files[0].Open()
		if err != nil {
			return nil, err
		}
		defer fr.Close()
		return parseArrayRawFloat64End(property.value, ln, elem_size), nil
	}
	return nil, errors.New("Invalid encoding")
}

func parseArrayRawFloat64End(r io.Reader, ln, elem_size int) []float64 {
	out := make([]int, ln/elem_size)
	if elem_size == 4 {
		for i := 0; i < len(out); i++ {
			var v float32
			binary.Read(property.value, binary.BigEndian, &v)
			out[i] = float64(v)
		}
	} else {
		for i := 0; i < len(out); i++ {
			var v float64
			binary.Read(property.value, binary.BigEndian, &v)
			out[i] = v
		}
	}
	return out
}

func parseDoubleVecDataVec2(property *Property) ([]Vec2, error) {
	if property.typ == 'd' {
		return parseBinaryArrayVec2(property, out_vec)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 2
	out_vec := make([]Vec2, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		out_vec[j].X = float64(tmp[i])
		out_vec[j].Y = float64(tmp[i+1])
	}
	return out_vec, nil
}

func parseDoubleVecDataVec3(property *Property) ([]Vec3, error) {
	if property.typ == 'd' {
		return parseBinaryArrayVec3(property, out_vec)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 3
	out_vec := make([]Vec3, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		out_vec[j].X = float64(tmp[i])
		out_vec[j].Y = float64(tmp[i+1])
		out_vec[j].Z = float64(tmp[i+2])
	}
	return out_vec, nil
}

func parseDoubleVecDataVec4(property *Property) ([]Vec4, error) {
	if property.typ == 'd' {
		return parseBinaryArrayVec4(property, out_vec)
	}
	tmp, err := parseBinaryArrayFloat32(property)
	if err != nil {
		return nil, err
	}
	size := 4
	out_vec := make([]Vec4, len(tmp)/size)
	for i := 0; i < len(tmp); i += size {
		j := i / size
		out_vec[j].X = float64(tmp[i])
		out_vec[j].Y = float64(tmp[i+1])
		out_vec[j].Z = float64(tmp[i+2])
		out_vec[j].W = float64(tmp[i+3])
	}
	return out_vec, nil
}

func parseVertexDataVec2(element *Element, name, index_name string) ([]Vec2, []int, VertexDataMapping, error) {
	data_element := findChild(element, name)
	if data_element == nil || data_element.first_property == nil {
		return nil, nil, 0, errors.New("Invalid data element")
	}
	idxs, mapping, err := parseVertexDataInner(element, name, index_name)
	vcs, err := parseDoubleVecDataVec2(*data_element.first_property)
	return vcs, idxs, mapping, err
}

func parseVertexDataVec3(element *Element, name, index_name string) ([]Vec2, []int, VertexDataMapping, error) {
	data_element := findChild(element, name)
	if data_element == nil || data_element.first_property == nil {
		return nil, nil, 0, errors.New("Invalid data element")
	}
	idxs, mapping, err := parseVertexDataInner(element, name, index_name)
	vcs, err := parseDoubleVecDataVec3(*data_element.first_property)
	return vcs, idxs, mapping, err
}

func parseVertexDataVec4(element *Element, name, index_name string) ([]Vec2, []int, VertexDataMapping, error) {
	data_element := findChild(element, name)
	if data_element == nil || data_element.first_property == nil {
		return nil, nil, 0, errors.New("Invalid data element")
	}
	idxs, mapping, err := parseVertexDataInner(element, name, index_name)
	vcs, err := parseDoubleVecDataVec4(*data_element.first_property)
	return vcs, idxs, mapping, err
}

func parseVertexDataInner(element *Element, name, index_name string) ([]int, VertexDataMapping, error) {
	mapping_element := findChild(element, "MappingInformationType")
	reference_element := findChild(element, "ReferenceInformationType")

	var mapping VertexDataMapping
	var err error

	if mapping_element != nil && mapping_element.first_property != nil {
		if mapping_element.first_property.value == "ByPolygonVertex" {
			mapping = BY_POLYGON_VERTEX
		} else if mapping_element.first_property.value == "ByPolygon" {
			mapping = BY_POLYGON
		} else if mapping_element.first_property.value == "ByVertice" || mapping_element.first_property.value == "ByVertex" {
			mapping = BY_VERTEX
		} else {
			return nil, 0, errors.New("Unable to parse mapping")
		}
	}
	if reference_element && reference_element.first_property {
		if reference_element.first_property.value == "IndexToDirect" {
			indices_element := findChild(element, index_name)
			if indices_element && indices_element.first_property {
				if idxs, err = parseBinaryArrayInt(*indices_element.first_property); err != nil {
					return nil, 0, errors.New("Unable to parse indices")
				}
			}
		} else if reference_element.first_property.value != "Direct" {
			return nil, 0, errors.New("Invalid properties")
		}
	}
	return idxs, mapping, nil
}

func parseTexture(scene *Scene, element *Element) *Object {
	texture := NewTexture(scene, element)
	texture_filename := findChild(element, "FileName")
	if texture_filename && texture_filename.first_property {
		texture.filename = texture_filename.first_property.value
	}
	texture_relative_filename := findChild(element, "RelativeFilename")
	if texture_relative_filename && texture_relative_filename.first_property {
		texture.relative_filename = texture_relative_filename.first_property.value
	}
	return texture
}

func parseLimbNode(scene *Scene, element *Element) (*Object, error) {
	if element.first_property == nil ||
		element.first_property.next == nil ||
		element.first_property.next.next == nil ||
		element.first_property.next.next.value != "LimbNode" {
		return nil, errors.New("Invalid limb node")
	}
	return NewLimpNode(scene, element)
}

func parseMesh(scene *Scene, element *Element) (*Object, error) {
	if element.first_property == nil ||
		element.first_property.next == nil ||
		element.first_property.next.next == nil ||
		element.first_property.next.next.value != "Mesh" {
		return nil, errors.New("Invalid mesh")
	}
	return NewMesh(scene, element)
}

func parseMaterial(scene *Scene, element *Element) *Object {
	material := NewMaterial(scene, element)
	prop := findChild(element, "Properties70")
	material.diffuse_color = Color{1, 1, 1}
	if prop != nil {
		prop = prop.child
	}
	for prop != nil {
		if prop.id == "P" && prop.first_property {
			if prop.first_property.value == "DiffuseColor" {
				material.diffuse_color.r = float32(prop.getProperty(4).getValue().toDouble())
				material.diffuse_color.g = float32(prop.getProperty(5).getValue().toDouble())
				material.diffuse_color.b = float32(prop.getProperty(6).getValue().toDouble())
			}
		}
		prop = prop.sibling
	}
	return material
}

func parseAnimationCurve(scene *Scene, element *Element) (*Object, error) {
	curve := &AnimationCurve{}

	times := findChild(element, "KeyTime")
	values := findChild(element, "KeyValueFloat")

	if times != nil && times.first_property != nil {
		curve.times = make([]int64, times.first_property.getCount())
		if !times.first_property.getValues(&curve.times[0], int(curve.times.size())*sizeof(curve.times[0])) {
			return nil, errors.New("Invalid animation curve")
		}
	}

	if values != nil && values.first_property != nil {
		curve.values = make([]float32, values.first_property.getCount())
		if !values.first_property.getValues(&curve.values[0], int(curve.values.size())*sizeof(curve.values[0])) {
			return nil, errors.New("Invalid animation curve")
		}
	}
	if curve.times.size() != curve.values.size() {
		return nil, errors.New("Invalid animation curve")
	}

	return nil, curve
}

func parseConnection(root *Element, scene *Scene) (bool, error) {
	connections := findChild(root, "Connections")
	if connections == nil {
		return true, nil
	}

	connection := connections.child
	for connection != nil {
		if !isString(connection.first_property) ||
			!isLong(connection.first_property.next) ||
			!isLong(connection.first_property.next.next) {
			return false, errors.New("Invalid connection")
		}

		var c Connection
		c.from = connection.first_property.next.value.touint64()
		c.to = connection.first_property.next.next.value.touint64()
		if connection.first_property.value == "OO" {
			c.typ = OBJECT_OBJECT
		} else if connection.first_property.value == "OP" {
			c.typ = OBJECT_PROPERTY
			if !connection.first_property.next.next.next {
				return false, errors.New("Invalid connection")
			}
			c.property = connection.first_property.next.next.next.value
		} else {
			return false, errors.New("Not supported")
		}
		scene.m_connections.push_back(c)

		connection = connection.sibling
	}
	return true, nil
}

func parseTakes(scene *Scene) (bool, error) {
	takes := findChild(scene.getRootElement(), "Takes")
	if takes == nil {
		return false, true
	}

	object := takes.child
	for object != nil {
		if object.id == "Take" {
			if !isString(object.first_property) {
				return false, errors.New("Invalid name in take")
			}

			var take TakeInfo
			take.name = object.first_property.value
			filename := findChild(*object, "FileName")
			if filename {
				if !isString(filename.first_property) {
					return false, errors.New("Invalid filename in take")
				}
				take.filename = filename.first_property.value
			}
			local_time := findChild(*object, "LocalTime")
			if local_time {
				if !isLong(local_time.first_property) || !isLong(local_time.first_property.next) {
					return false, errors.New("Invalid local time in take")
				}

				take.local_time_from = fbxTimeToSeconds(local_time.first_property.value.toint64())
				take.local_time_to = fbxTimeToSeconds(local_time.first_property.next.value.toint64())
			}
			reference_time := findChild(*object, "ReferenceTime")
			if reference_time != nil {
				if !isLong(reference_time.first_property) || !isLong(reference_time.first_property.next) {
					return false, errors.New("Invalid reference time in take")
				}

				take.reference_time_from = fbxTimeToSeconds(reference_time.first_property.value.toint64())
				take.reference_time_to = fbxTimeToSeconds(reference_time.first_property.next.value.toint64())
			}

			scene.m_take_infos.push_back(take)
		}

		object = object.sibling
	}

	return true, nil
}

func parseGlobalSettings(root *Element, scene *Scene) {
	for settings := root.child; settings != nil; settings = settings.sibling {
		if settings.id == "GlobalSettings" {
			for props70 := settings.child; props70 != nil; props70 = props70.sibling {
				if props70.id == "Properties70" {
					for node := props70.child; node; node = node.sibling {
						if !node.first_property {
							continue
						}

						if node.first_property.value == "UpAxis" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.UpAxis = value.begin.(UpVector)
							}
						}

						if node.first_property.value == "UpAxisSign" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.UpAxisSign = value.begin.(int)
							}
						}

						if node.first_property.value == "FrontAxis" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.FrontAxis = value.begin.(FrontVector)
							}
						}

						if node.first_property.value == "FrontAxisSign" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.FrontAxisSign = value.begin.(int)
							}
						}

						if node.first_property.value == "CoordAxis" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.CoordAxis = value.begin.(CoordSystem)
							}
						}

						if node.first_property.value == "CoordAxisSign" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.CoordAxisSign = value.begin.(int)
							}
						}

						if node.first_property.value == "OriginalUpAxis" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.OriginalUpAxis = value.begin.(int)
							}
						}

						if node.first_property.value == "OriginalUpAxisSign" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.OriginalUpAxisSign = value.begin.(int)
							}
						}

						if node.first_property.value == "UnitScaleFactor" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.UnitScaleFactor = value.begin.(float)
							}
						}

						if node.first_property.value == "OriginalUnitScaleFactor" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.OriginalUnitScaleFactor = value.begin.(float)
							}
						}

						if node.first_property.value == "TimeSpanStart" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.TimeSpanStart = value.begin.(uint64)
							}
						}

						if node.first_property.value == "TimeSpanStop" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.TimeSpanStop = value.begin.(uint64)
							}
						}

						if node.first_property.value == "TimeMode" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.TimeMode = value.begin.(FrameRate)
							}
						}

						if node.first_property.value == "CustomFrameRate" {
							prop := node.getProperty(4)
							if prop != nil {
								value := prop.getValue()
								scene.m_settings.CustomFrameRate = value.begin.(float)
							}
						}

						scene.m_scene_frame_rate = getFramerateFromTimeMode(scene.m_settings.TimeMode, scene.m_settings.CustomFrameRate)
					}
					break
				}
			}
			break
		}
	}
}

func parseObjects(root *Element, scene *Scene) (bool, error) {
	objs := findChild(root, "Objects")
	if objs == nil {
		return true, nil
	}

	scene.m_root = NewRoot(*scene, root)
	scene.m_object_map[0] = ObjectPair{&root, scene.m_root}

	object := objs.child
	for object != nil {
		if !isLong(object.first_property) {
			return false, errors.New("Invalid")
		}

		id := object.first_property.value.touint64()
		scene.m_object_map[id] = ObjectPair{object, nullptr}
		object = object.sibling
	}

	for _, iter := range scene.m_object_map {
		var obj *Object

		if iter.second.object == scene.m_root {
			continue
		}

		if iter.second.element.id == "Geometry" {
			last_prop := iter.second.element.first_property
			for last_prop.next != nil {
				last_prop = last_prop.next
			}
			if last_prop != nil && last_prop.value == "Mesh" {
				obj = parseGeometry(*scene, *iter.second.element)
			}
		} else if iter.second.element.id == "Material" {
			obj = parseMaterial(*scene, *iter.second.element)
		} else if iter.second.element.id == "AnimationStack" {
			obj = NewAnimationStack(*scene, *iter.second.element)
			if !obj.isError() {
				stack := obj.getValue().(*AnimationStackImpl)
				scene.m_animation_stacks.push_back(stack)
			}
		} else if iter.second.element.id == "AnimationLayer" {
			obj = NewAnimationLayer(*scene, *iter.second.element)
		} else if iter.second.element.id == "AnimationCurve" {
			obj = parseAnimationCurve(*scene, *iter.second.element)
		} else if iter.second.element.id == "AnimationCurveNode" {
			obj = NewAnimationCurveNode(*scene, *iter.second.element)
		} else if iter.second.element.id == "Deformer" {
			class_prop = iter.second.element.getProperty(2)
			if class_prop != nil {
				v := class_prop.getValue()
				if v == "Cluster" {
					obj = parseCluster(*scene, *iter.second.element)
				} else if v == "Skin" {
					obj = NewSkin(*scene, *iter.second.element)
				}
			}
		} else if iter.second.element.id == "NodeAttribute" {
			obj = parseNodeAttribute(*scene, *iter.second.element)
		} else if iter.second.element.id == "Model" {
			iter.second.element.getProperty(2)
			if class_prop != nil {
				v := class_prop.getValue()
				if v == "Mesh" {
					obj = parseMesh(*scene, *iter.second.element)
					if !obj.isError() {
						mesh = obj.getValue().(*Mesh)
						scene.m_meshes.push_back(mesh)
						obj = mesh
					}
				} else if v == "LimbNode" {
					obj, err = parseLimbNode(*scene, *iter.second.element)
					if err != nil {
						return false, err
					}
				} else if v == "Null" || v == "Root" {
					obj = NewNull(*scene, *iter.second.element)
				}
			}
		} else if iter.second.element.id == "Texture" {
			obj = parseTexture(*scene, *iter.second.element)
		}

		if obj.isError() {
			return false, nil // error?
		}

		val := obj.getValue()
		scene.m_object_map[iter.first].object = val
		if val != nil {
			scene.m_all_objects.push_back(val)
			val.id = iter.first
		}
	}
	for _, con := range scene.m_connections {
		parent := scene.m_object_map[con.to].object
		child := scene.m_object_map[con.from].object
		if child == nil || parent == nil {
			continue
		}

		ctyp := child.getType()

		switch ctyp {
		case NODE_ATTRIBUTE:
			if parent.node_attribute {
				return false, errors.New("Invalid node attribute")
			}
			parent.node_attribute = child.(*NodeAttribute)
		case ANIMATION_CURVE_NODE:
			if parent.isNode() {
				node := child.(*AnimationCurveNode)
				node.bone = parent
				node.bone_link_property = con.property
			}
		}

		switch parent.getType() {
		case MESH:
			{
				mesh := parent.(*MeshImpl)
				switch ctyp {
				case GEOMETRY:
					if mesh.geometry != nil {
						return false, errors.New("Invalid mesh")
					}
					mesh.geometry = child.(*Geometry)
				case MATERIAL:
					mesh.materials.push_back(child.(*Material))
				}
			}
		case SKIN:
			{
				skin := parent.(*Skin)
				if ctyp == CLUSTER {
					cluster := child.(*Cluster)
					skin.clusters.push_back(cluster)
					if cluster.skin != nil {
						return false, errors.New("Invalid cluster")
					}
					cluster.skin = skin
				}
			}
		case MATERIAL:
			mat := parent.(*Material)
			if ctyp == TEXTURE {
				ttyp = COUNT
				if con.property == "NormalMap" {
					ttyp = NORMAL
				} else if con.property == "DiffuseColor" {
					ttyp = DIFFUSE
				}
				if ttyp == COUNT {
					break
				}
				if mat.textures[ttyp] != nil {
					break
				}
				mat.textures[ttyp] = child.(*Texture)
			}
		case GEOMETRY:
			geom := parent.(*Geometry)
			if ctyp == SKIN {
				geom.skin = child.(*Skin)
			}
		case CLUSTER:
			cluster := parent.(*Cluster)
			if ctyp == LIMB_NODE || ctyp == MESH || ctyp == NULL_NODE {
				if cluster.link != nil {
					return false, errors.New("Invalid cluster")
				}
				cluster.link = child
			}

		case ANIMATION_LAYER:
			if ctyp == ANIMATION_CURVE_NODE {
				parent.(*AnimationLayer).curve_nodes.push_back(child.(*AnimationCurveNode))
			}

		case ANIMATION_CURVE_NODE:
			node = parent.(*AnimationCurveNode)
			if ctyp == ANIMATION_CURVE {
				if !node.curves[0].curve == nil {
					node.curves[0].connection = &con
					node.curves[0].curve = child.(*AnimationCurve)
				} else if node.curves[1].curve == nil {
					node.curves[1].connection = &con
					node.curves[1].curve = child.(*AnimationCurve)
				} else if !node.curves[2].curve == nil {
					node.curves[2].connection = &con
					node.curves[2].curve = child.(*AnimationCurve)
				} else {
					return false, errors.New("Invalid animation node")
				}
			}
		}
	}

	for _, iter := range scene.m_object_map {
		obj := iter.second.object
		if obj == nil {
			continue
		}
		if obj.getType() == CLUSTER {
			if !iter.second.object.(*ClusterImpl).postprocess() {
				return false, errors.New("Failed to postprocess cluster")
			}
		}
	}

	return false, true
}
