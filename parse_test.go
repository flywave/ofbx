package ofbx

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/oakmound/oak/v2/alg/floatgeom"
	"github.com/stretchr/testify/assert"
)

func TestParseTemplates(t *testing.T) {
	// 创建测试用的Element结构
	root := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Definitions"))},
				Children: []*Element{
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("ObjectType"))},
						Properties: []*Property{
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Geometry"))}},
						},
						Children: []*Element{
							{
								ID: &DataView{Reader: *bytes.NewReader([]byte("PropertyTemplate"))},
								Properties: []*Property{
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("FbxMesh"))}},
								},
							},
						},
					},
				},
			},
		},
	}

	// 测试ParseTemplates不会panic
	assert.NotPanics(t, func() {
		ParseTemplates(root)
	})
}

func TestParseBinaryArrayInt(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'i', Encoding: 0}
	result, err := parseBinaryArrayInt(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试无效类型
	prop = &Property{Count: 1, Type: 'd', Encoding: 0}
	_, err = parseBinaryArrayInt(prop)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid type")
}

func TestParseBinaryArrayFloat64(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'd', Encoding: 0}
	result, err := parseBinaryArrayFloat64(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试无效类型
	prop = &Property{Count: 1, Type: 'i', Encoding: 0}
	_, err = parseBinaryArrayFloat64(prop)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid type")
}

func TestParseBinaryArrayFloat32(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'f', Encoding: 0}
	result, err := parseBinaryArrayFloat32(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试从float64转换
	data := []byte{0, 0, 0, 0, 0, 0, 240, 63} // 1.0 in little-endian float64
	prop = &Property{
		Count:    1,
		Type:     'd',
		Encoding: 0,
		value:    &DataView{Reader: *bytes.NewReader(data)},
	}
	result, err = parseBinaryArrayFloat32(prop)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.InDelta(t, 1.0, result[0], 0.001)
}

func TestParseBinaryArrayVec2(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'd', Encoding: 0}
	result, err := parseBinaryArrayVec2(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试Vec2数据
	data := []byte{
		0, 0, 0, 0, 0, 0, 240, 63, // 1.0
		0, 0, 0, 0, 0, 0, 0, 64, // 2.0
	}
	prop = &Property{
		Count:    2,
		Type:     'd',
		Encoding: 0,
		value:    &DataView{Reader: *bytes.NewReader(data)},
	}
	result, err = parseBinaryArrayVec2(prop)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, floatgeom.Point2{1.0, 2.0}, result[0])
}

func TestParseBinaryArrayVec3(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'd', Encoding: 0}
	result, err := parseBinaryArrayVec3(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试Vec3数据
	data := []byte{
		0, 0, 0, 0, 0, 0, 240, 63, // 1.0
		0, 0, 0, 0, 0, 0, 0, 64, // 2.0
		0, 0, 0, 0, 0, 0, 8, 64, // 3.0
	}
	prop = &Property{
		Count:    3,
		Type:     'd',
		Encoding: 0,
		value:    &DataView{Reader: *bytes.NewReader(data)},
	}
	result, err = parseBinaryArrayVec3(prop)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, floatgeom.Point3{1.0, 2.0, 3.0}, result[0])
}

func TestParseBinaryArrayVec4(t *testing.T) {
	// 测试空数组
	prop := &Property{Count: 0, Type: 'd', Encoding: 0}
	result, err := parseBinaryArrayVec4(prop)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试Vec4数据
	data := []byte{
		0, 0, 0, 0, 0, 0, 240, 63, // 1.0
		0, 0, 0, 0, 0, 0, 0, 64, // 2.0
		0, 0, 0, 0, 0, 0, 8, 64, // 3.0
		0, 0, 0, 0, 0, 0, 16, 64, // 4.0
	}
	prop = &Property{
		Count:    4,
		Type:     'd',
		Encoding: 0,
		value:    &DataView{Reader: *bytes.NewReader(data)},
	}
	result, err = parseBinaryArrayVec4(prop)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, floatgeom.Point4{1.0, 2.0, 3.0, 4.0}, result[0])
}

func TestParseDoubleVecDataVec2(t *testing.T) {
	// 创建测试用的Element结构
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("UV"))},
				Properties: []*Property{
					{Type: 'd', Count: 2, value: &DataView{Reader: *bytes.NewReader([]byte{
						0, 0, 0, 0, 0, 0, 240, 63, // 1.0
						0, 0, 0, 0, 0, 0, 0, 64, // 2.0
					})}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("MappingInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("ByVertex"))}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("ReferenceInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Direct"))}},
				},
			},
		},
	}

	result, idxs, mapping, err := parseVertexDataVec2(element, "UV", "UVIndex")
	assert.NoError(t, err)
	fmt.Printf("result: %v, idxs: %v, mapping: %v\n", result, idxs, mapping)
	assert.NotNil(t, result)
	assert.Nil(t, idxs) // Direct模式下没有索引
	assert.NotZero(t, mapping)
}

func TestParseDoubleVecDataVec3(t *testing.T) {
	// 创建测试用的Element结构
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Vertices"))},
				Properties: []*Property{
					{Type: 'd', Count: 3, value: &DataView{Reader: *bytes.NewReader([]byte{
						0, 0, 0, 0, 0, 0, 240, 63, // 1.0
						0, 0, 0, 0, 0, 0, 0, 64, // 2.0
						0, 0, 0, 0, 0, 0, 8, 64, // 3.0
					})}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("MappingInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("ByVertex"))}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("ReferenceInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Direct"))}},
				},
			},
		},
	}

	result, idxs, mapping, err := parseVertexDataVec3(element, "Vertices", "PolygonVertexIndex")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, idxs) // Direct模式下没有索引
	assert.NotZero(t, mapping)
}

func TestParseDoubleVecDataVec4(t *testing.T) {
	// 创建测试用的Element结构
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Colors"))},
				Properties: []*Property{
					{Type: 'd', Count: 4, value: &DataView{Reader: *bytes.NewReader([]byte{
						0, 0, 0, 0, 0, 0, 240, 63, // 1.0
						0, 0, 0, 0, 0, 0, 0, 64, // 2.0
						0, 0, 0, 0, 0, 0, 8, 64, // 3.0
						0, 0, 0, 0, 0, 0, 16, 64, // 4.0
					})}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("MappingInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("ByVertex"))}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("ReferenceInformationType"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Direct"))}},
				},
			},
		},
	}

	result, idxs, mapping, err := parseVertexDataVec4(element, "Colors", "ColorIndex")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, idxs) // Direct模式下没有索引
	assert.NotZero(t, mapping)
}

func TestParseVertexDataInnerErrorCases(t *testing.T) {
	// 测试无效数据元素
	element := &Element{}
	idxs, mapping, result, err := parseVertexDataInner(element, "NonExistent", "Index")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid data element")
	assert.Nil(t, result)
	assert.Nil(t, idxs)
	assert.Zero(t, mapping)
}

func TestParseTexture(t *testing.T) {
	scene := &Scene{}
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("FileName"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test_texture.jpg"))}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("RelativeFilename"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("textures/test_texture.jpg"))}},
				},
			},
		},
	}

	texture := parseTexture(scene, element)
	assert.NotNil(t, texture)
	assert.Equal(t, "test_texture.jpg", texture.filename.String())
	assert.Equal(t, "textures/test_texture.jpg", texture.relativeFilename.String())
}

func TestParseLimbNode(t *testing.T) {
	scene := &Scene{}

	// 测试有效的LimbNode
	element := &Element{
		Properties: []*Property{
			{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test"))}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("LimbNode"))}},
		},
	}

	node, err := parseLimbNode(scene, element)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// 测试无效的LimbNode
	element = &Element{
		Properties: []*Property{
			{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test"))}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("NotLimbNode"))}},
		},
	}

	node, err = parseLimbNode(scene, element)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid limb node")
	assert.Nil(t, node)
}

func TestParseMesh(t *testing.T) {
	scene := &Scene{}

	// 测试有效的Mesh
	element := &Element{
		Properties: []*Property{
			{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test"))}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Mesh"))}},
		},
	}

	mesh, err := parseMesh(scene, element)
	assert.NoError(t, err)
	assert.NotNil(t, mesh)

	// 测试无效的Mesh
	element = &Element{
		Properties: []*Property{
			{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test"))}},
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("NotMesh"))}},
		},
	}

	mesh, err = parseMesh(scene, element)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid mesh")
	assert.Nil(t, mesh)
}

func TestParseMaterial(t *testing.T) {
	scene := &Scene{}
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
				Children: []*Element{
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
						Properties: []*Property{
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("DiffuseColor"))}},
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Color"))}},
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
							{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 240, 63})}}, // 1.0
							{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 64})}},   // 2.0
							{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 8, 64})}},   // 3.0
						},
					},
				},
			},
		},
	}

	material := parseMaterial(scene, element)
	assert.NotNil(t, material)
	assert.Equal(t, float32(1.0), material.DiffuseColor.R)
	assert.Equal(t, float32(2.0), material.DiffuseColor.G)
	assert.Equal(t, float32(3.0), material.DiffuseColor.B)
}

func TestParseMaterialDefaultValues(t *testing.T) {
	scene := &Scene{}
	element := &Element{} // 空元素，没有Properties70

	material := parseMaterial(scene, element)
	assert.NotNil(t, material)
	assert.Equal(t, float32(1.0), material.DiffuseColor.R)
	assert.Equal(t, float32(1.0), material.DiffuseColor.G)
	assert.Equal(t, float32(1.0), material.DiffuseColor.B)
}

func TestParseAnimationCurve(t *testing.T) {
	scene := &Scene{}
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("KeyTime"))},
				Properties: []*Property{
					{Type: 'l', Count: 2, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("KeyValueFloat"))},
				Properties: []*Property{
					{Type: 'f', Count: 2, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 128, 63, 0, 0, 0, 64})}}, // [1.0, 2.0]
				},
			},
		},
	}

	curve, err := parseAnimationCurve(scene, element)
	assert.NoError(t, err)
	assert.NotNil(t, curve)
	assert.Len(t, curve.Times, 2)
	assert.Len(t, curve.Values, 2)
}

func TestParseAnimationCurveInvalid(t *testing.T) {
	scene := &Scene{}

	// 测试长度不匹配
	element := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("KeyTime"))},
				Properties: []*Property{
					{Type: 'l', Count: 2, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})}},
				},
			},
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("KeyValueFloat"))},
				Properties: []*Property{
					{Type: 'f', Count: 1, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 128, 63})}}, // [1.0]
				},
			},
		},
	}

	curve, err := parseAnimationCurve(scene, element)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "len error")
	assert.Nil(t, curve)
}

func TestParseConnection(t *testing.T) {
	scene := &Scene{}
	root := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Connections"))},
				Children: []*Element{
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("C"))},
						Properties: []*Property{
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("OO"))}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{2, 0, 0, 0, 0, 0, 0, 0})}},
						},
					},
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("C"))},
						Properties: []*Property{
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("OP"))}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{3, 0, 0, 0, 0, 0, 0, 0})}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{4, 0, 0, 0, 0, 0, 0, 0})}},
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("property_name"))}},
						},
					},
				},
			},
		},
	}

	success, err := parseConnection(root, scene)
	assert.NoError(t, err)
	assert.True(t, success)
	assert.Len(t, scene.Connections, 2)
	assert.Equal(t, uint64(1), scene.Connections[0].from)
	assert.Equal(t, uint64(2), scene.Connections[0].to)
	assert.Equal(t, ObjectConn, scene.Connections[0].typ)
	assert.Equal(t, uint64(3), scene.Connections[1].from)
	assert.Equal(t, uint64(4), scene.Connections[1].to)
	assert.Equal(t, PropConn, scene.Connections[1].typ)
	assert.Equal(t, "property_name", scene.Connections[1].property)
}

func TestParseConnectionInvalid(t *testing.T) {
	scene := &Scene{}
	root := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("Connections"))},
				Children: []*Element{
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("C"))},
						Properties: []*Property{
							{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("INVALID"))}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})}},
							{Type: LONG, value: &DataView{Reader: *bytes.NewReader([]byte{2, 0, 0, 0, 0, 0, 0, 0})}},
						},
					},
				},
			},
		},
	}

	_, err := parseConnection(root, scene)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not supported")
}

func TestParseGlobalSettings(t *testing.T) {
	scene := &Scene{}
	elem := &Element{
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("GlobalSettings"))},
				Children: []*Element{
					{
						ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
						Children: []*Element{
							{
								ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
								Properties: []*Property{
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("UpAxis"))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("int"))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
									{Type: INTEGER, value: &DataView{Reader: *bytes.NewReader([]byte{1, 0, 0, 0})}},
								},
							},
							{
								ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
								Properties: []*Property{
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("UnitScaleFactor"))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("double"))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
									{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
									{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 240, 63})}}, // 1.0
								},
							},
						},
					},
				},
			},
		},
	}

	parseGlobalSettings(elem, scene)
	assert.Equal(t, UpVector(1), scene.Settings.UpAxis)
	fmt.Printf("UnitScaleFactor: %v\n", scene.Settings.UnitScaleFactor)
	assert.Equal(t, float32(1.0), scene.Settings.UnitScaleFactor)
}

func TestParseObjectsEmpty(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	root := &Element{
		Children: []*Element{
			{
				ID:       &DataView{Reader: *bytes.NewReader([]byte("Objects"))},
				Children: []*Element{}, // 空的Objects节点
			},
		},
	} // 包含空Objects节点的根元素

	success, err := parseObjects(root, scene)
	assert.NoError(t, err)
	assert.True(t, success)
	assert.NotNil(t, scene.RootNode)
	assert.Len(t, scene.ObjectMap, 1) // 只有RootNode
}

func TestParseArrayRawIntEnd(t *testing.T) {
	// 测试32位整数
	data := []byte{1, 0, 0, 0, 2, 0, 0, 0} // [1, 2] in little-endian int32
	result := parseArrayRawIntEnd(bytes.NewReader(data), 2, 4)
	assert.Equal(t, []int{1, 2}, result)

	// 测试64位整数
	data = []byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0} // [1, 2] in little-endian int64
	result = parseArrayRawIntEnd(bytes.NewReader(data), 2, 8)
	assert.Equal(t, []int{1, 2}, result)
}

func TestParseArrayRawFloat64End(t *testing.T) {
	data := []byte{0, 0, 0, 0, 0, 0, 240, 63, 0, 0, 0, 0, 0, 0, 0, 64} // [1.0, 2.0] in little-endian float64
	result := parseArrayRawFloat64End(bytes.NewReader(data), 2, 8)
	assert.Len(t, result, 2)
	assert.InDelta(t, 1.0, result[0], 0.001)
	assert.InDelta(t, 2.0, result[1], 0.001)
}

func TestParseArrayRawFloat32End(t *testing.T) {
	data := []byte{0, 0, 128, 63, 0, 0, 0, 64} // [1.0, 2.0] in little-endian float32
	result := parseArrayRawFloat32End(bytes.NewReader(data), 2, 4)
	assert.Len(t, result, 2)
	assert.InDelta(t, 1.0, result[0], 0.001)
	assert.InDelta(t, 2.0, result[1], 0.001)
}

func TestParseArrayRawInt64End(t *testing.T) {
	data := []byte{1, 0, 0, 0, 2, 0, 0, 0} // [1, 2] in little-endian int32
	result := parseArrayRawInt64End(bytes.NewReader(data), 2, 4)
	assert.Equal(t, []int64{1, 2}, result)

	data = []byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0} // [1, 2] in little-endian int64
	result = parseArrayRawInt64End(bytes.NewReader(data), 2, 8)
	assert.Equal(t, []int64{1, 2}, result)
}
