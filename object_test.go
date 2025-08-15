package ofbx

import (
	"bytes"
	"testing"

	"github.com/oakmound/oak/v2/alg/floatgeom"
)

func TestObjectID(t *testing.T) {
	obj := &Object{id: 12345}

	if got := obj.ID(); got != 12345 {
		t.Errorf("Object.ID() = %d, want %d", got, 12345)
	}
}

func TestObjectSetID(t *testing.T) {
	obj := &Object{}
	obj.SetID(54321)

	if got := obj.ID(); got != 54321 {
		t.Errorf("Object.SetID() failed, got %d, want %d", got, 54321)
	}
}

func TestObjectName(t *testing.T) {
	obj := &Object{name: "test_object"}

	if got := obj.Name(); got != "test_object" {
		t.Errorf("Object.Name() = %q, want %q", got, "test_object")
	}
}

func TestObjectType(t *testing.T) {
	obj := &Object{}

	if got := obj.Type(); got != NOTYPE {
		t.Errorf("Object.Type() = %v, want %v", got, NOTYPE)
	}
}

func TestObjectElement(t *testing.T) {
	elem := &Element{ID: &DataView{}}
	obj := &Object{element: elem}

	if got := obj.Element(); got != elem {
		t.Errorf("Object.Element() returned wrong element")
	}
}

func TestObjectNodeAttribute(t *testing.T) {
	attr := &Object{name: "test_attr"}
	obj := &Object{nodeAttribute: attr}

	if got := obj.NodeAttribute(); got != attr {
		t.Errorf("Object.NodeAttribute() returned wrong attribute")
	}
}

func TestObjectSetNodeAttribute(t *testing.T) {
	attr := &Object{name: "new_attr"}
	obj := &Object{}
	obj.SetNodeAttribute(attr)

	if got := obj.NodeAttribute(); got != attr {
		t.Errorf("Object.SetNodeAttribute() failed")
	}
}

func TestObjectIsNode(t *testing.T) {
	obj := &Object{isNode: true}

	if got := obj.IsNode(); got != true {
		t.Errorf("Object.IsNode() = %v, want %v", got, true)
	}

	obj.isNode = false
	if got := obj.IsNode(); got != false {
		t.Errorf("Object.IsNode() = %v, want %v", got, false)
	}
}

func TestObjectScene(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	obj := &Object{scene: scene}

	if got := obj.Scene(); got != scene {
		t.Errorf("Object.Scene() returned wrong scene")
	}
}

func TestGetParent(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// 创建测试对象
	child := &Object{id: 1, isNode: true}
	parent := &Object{id: 2, isNode: true}

	// 创建连接关系
	scene.Connections = []Connection{
		{from: 1, to: 2}, // child -> parent
	}

	// 设置对象映射
	scene.ObjectMap[1] = child
	scene.ObjectMap[2] = parent

	// 设置场景
	child.scene = scene

	if got := getParent(child); got != parent {
		t.Errorf("getParent() returned wrong parent")
	}
}

func TestGetParentNoParent(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	child := &Object{id: 1, isNode: true}

	// 没有连接关系
	scene.Connections = []Connection{}
	scene.ObjectMap[1] = child
	child.scene = scene

	if got := getParent(child); got != nil {
		t.Errorf("getParent() = %v, want nil when no parent", got)
	}
}

func TestGetRotationOrder(t *testing.T) {
	// 创建带有旋转顺序属性的对象
	properties70 := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("RotationOrder"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("RotationOrder"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: INTEGER, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0})}}, // EulerXYZ = 0
				},
			},
		},
	}
	elem := &Element{
		Children: []*Element{properties70},
	}

	obj := &Object{element: elem}

	if got := getRotationOrder(obj); got != EulerXYZ {
		t.Errorf("getRotationOrder() = %v, want %v", got, EulerXYZ)
	}
}

func TestGetRotationOrderDefault(t *testing.T) {
	// 没有旋转顺序属性的对象
	elem := &Element{Properties: []*Property{}}
	obj := &Object{element: elem}

	if got := getRotationOrder(obj); got != EulerZYX {
		t.Errorf("getRotationOrder() default = %v, want %v", got, EulerZYX)
	}
}

func TestGetRotationOffset(t *testing.T) {
	// 创建带有旋转偏移属性的对象
	properties70 := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("RotationOffset"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("RotationOffset"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 240, 63})}}, // 1.0 in IEEE 754
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 64})}},   // 2.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 8, 64})}},   // 3.0
				},
			},
		},
	}
	elem := &Element{
		Children: []*Element{properties70},
	}

	obj := &Object{element: elem}
	expected := floatgeom.Point3{1, 2, 3}

	if got := getRotationOffset(obj); got != expected {
		t.Errorf("getRotationOffset() = %v, want %v", got, expected)
	}
}

func TestGetRotationOffsetDefault(t *testing.T) {
	// 没有旋转偏移属性的对象
	elem := &Element{Properties: []*Property{}}
	obj := &Object{element: elem}

	expected := floatgeom.Point3{}
	if got := getRotationOffset(obj); got != expected {
		t.Errorf("getRotationOffset() default = %v, want %v", got, expected)
	}
}

func TestGetLocalTranslation(t *testing.T) {
	// 创建带有本地平移属性的对象
	properties70 := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 240, 63})}}, // 1.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 64})}},   // 2.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 8, 64})}},   // 3.0
				},
			},
		},
	}
	elem := &Element{
		Children: []*Element{properties70},
	}

	obj := &Object{element: elem}
	expected := floatgeom.Point3{1, 2, 3}

	if got := getLocalTranslation(obj); got != expected {
		t.Errorf("getLocalTranslation() = %v, want %v", got, expected)
	}
}

func TestGetLocalScalingDefault(t *testing.T) {
	// 没有本地缩放属性的对象
	elem := &Element{Properties: []*Property{}}
	obj := &Object{element: elem}

	expected := floatgeom.Point3{1, 1, 1}
	if got := getLocalScaling(obj); got != expected {
		t.Errorf("getLocalScaling() default = %v, want %v", got, expected)
	}
}

func TestGetGlobalTransform(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// 创建根节点
	properties70Root := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
				},
			},
		},
	}
	root := &Object{id: 1, isNode: true, scene: scene, element: &Element{Children: []*Element{properties70Root}}}

	// 创建子节点
	properties70Child := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
				},
			},
		},
	}
	child := &Object{id: 2, isNode: true, scene: scene, element: &Element{Children: []*Element{properties70Child}}}

	// 创建连接关系
	scene.Connections = []Connection{
		{from: 2, to: 1}, // child -> root
	}

	// 设置对象映射
	scene.ObjectMap[1] = root
	scene.ObjectMap[2] = child

	// 测试全局变换
	transform := getGlobalTransform(child)
	if transform.isZero() {
		t.Errorf("getGlobalTransform() returned zero matrix")
	}
}

func TestGetLocalTransform(t *testing.T) {
	properties70 := &Element{
		ID: &DataView{Reader: *bytes.NewReader([]byte("Properties70"))},
		Children: []*Element{
			{
				ID: &DataView{Reader: *bytes.NewReader([]byte("P"))},
				Properties: []*Property{
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("Lcl Translation"))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},
					{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("A"))}},
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
					{Type: DOUBLE, value: &DataView{Reader: *bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}}, // 0.0
				},
			},
		},
	}
	obj := &Object{element: &Element{Children: []*Element{properties70}}}

	// 测试本地变换
	transform := GetLocalTransform(obj)
	if transform.isZero() {
		t.Errorf("GetLocalTransform() returned zero matrix")
	}
}

func TestNewObject(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	elem := &Element{
		Properties: []*Property{
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte(""))}},          // 索引0通常是空字符串
			{Type: STRING, value: &DataView{Reader: *bytes.NewReader([]byte("test_name"))}}, // 索引1是对象名称
		},
	}

	obj := NewObject(scene, elem)

	if obj.Scene() != scene {
		t.Errorf("NewObject() scene not set correctly")
	}
	if obj.Element() != elem {
		t.Errorf("NewObject() element not set correctly")
	}
	if obj.Name() != "test_name" {
		t.Errorf("NewObject() name = %q, want %q", obj.Name(), "test_name")
	}
}
