package ofbx

import (
	"io"

	"github.com/oakmound/oak/v2/alg/floatgeom"
)

// Object is the top level general class in fbx
type Object struct {
	id            uint64
	name          string
	element       *Element
	nodeAttribute Obj

	isNode bool
	scene  *Scene
}

// Obj interface version of Object
type Obj interface {
	ID() uint64
	SetID(uint64)
	Name() string
	Element() *Element
	NodeAttribute() Obj
	SetNodeAttribute(na Obj)
	IsNode() bool
	Scene() *Scene
	Type() Type
	String() string
	stringPrefix(string) string
}

// ID returns the Object's integer id value
func (o *Object) ID() uint64 {
	return o.id
}

// SetID sets the Objects ID
func (o *Object) SetID(i uint64) {
	o.id = i
}

// Name gets the Objects Name
func (o *Object) Name() string {
	return o.name
}

func (o *Object) Type() Type {
	return NOTYPE
}

// Element gets the Element on the Object
func (o *Object) Element() *Element {
	return o.element
}

// NodeAttribute should be deprecated and in favor of exporting the attribute
func (o *Object) NodeAttribute() Obj {
	return o.nodeAttribute
}

// SetNodeAttribute sets the attribute but should just exported field
func (o *Object) SetNodeAttribute(na Obj) {
	o.nodeAttribute = na
}

// IsNode ret[urns whether this is a node
func (o *Object) IsNode() bool {
	return o.isNode
}

// Scene returns the scene used for the object
func (o *Object) Scene() *Scene {
	return o.scene
}

func (o *Object) String() string {
	return o.stringPrefix("")
}
func (o *Object) stringPrefix(prefix string) string {
	s := "" //:= prefix //+ "Object: " + fmt.Sprintf("%d", o.id) + ", " + o.name
	if o.element != nil {
		s += o.element.stringPrefix(prefix)
	}
	if o.nodeAttribute != nil {
		if strn, ok := o.nodeAttribute.(stringPrefixer); ok {
			s += prefix + "node=\n" + strn.stringPrefix("\t"+prefix)
		}
	}
	// if o.is_node {
	// 	s += "(is_node)"
	// }
	return s
}

// NewObject creates a new object
func NewObject(scene *Scene, e *Element) *Object {
	o := &Object{
		scene:   scene,
		element: e,
	}
	if prop := e.getProperty(1); prop != nil {
		o.name = prop.value.String()
	}
	return o
}

func resolveAllObjectLinks(o Obj) []Obj {
	return resolveObjectLinks(o, NOTYPE, []string{""})
}

func resolveObjectLinkIndex(o Obj, idx int) Obj {
	return resolveObjectLink(o, NOTYPE, "", idx)
}

func resolveObjectLink(o Obj, typ Type, property string, idx int) Obj {
	id := o.ID()
	for _, conn := range o.Scene().Connections {
		if conn.to == id && conn.from != 0 {
			obj := o.Scene().ObjectMap[conn.from]
			if obj != nil && (obj.Type() == typ || typ == NOTYPE) {
				if property == "" || conn.property == property {
					if idx == 0 {
						return obj
					}
					idx--
				}
			}
		}
	}
	return nil
}

func resolveObjectLinks(o Obj, typ Type, properties []string) []Obj {
	id := o.ID()
	out := make([]Obj, 0)
	for _, conn := range o.Scene().Connections {
		if conn.to == id && conn.from != 0 {
			obj := o.Scene().ObjectMap[conn.from]
			if obj != nil && (obj.Type() == typ || typ == NOTYPE) {
				for _, prop2 := range properties {
					if prop2 == "" || conn.property == prop2 {
						out = append(out, obj)
						break
					}
				}
			}
		}
	}
	return out
}

func resolveObjectLinkReverse(o Obj, typ Type) Obj {
	var id uint64
	if prop := o.Element().getProperty(0); prop != nil {
		rdr := prop.value
		rdr.Seek(0, io.SeekStart)
		id = rdr.touint64()
	}
	for _, conn := range o.Scene().Connections {
		//fmt.Println("Connection iterated", id, conn.from, conn.to)
		if conn.from == id && conn.to != 0 {
			obj := o.Scene().ObjectMap[conn.to]
			if obj != nil && obj.Type() == typ {
				return obj
			}
		}
	}
	return nil
}

func getParent(o Obj) Obj {
	for _, con := range o.Scene().Connections {
		if con.from == o.ID() {
			obj := o.Scene().ObjectMap[con.to]
			if obj != nil && obj.IsNode() {
				return obj
			}
		}
	}
	return nil
}

func getRotationOrder(o Obj) RotationOrder {
	return RotationOrder(resolveEnumProperty(o, "RotationOrder", int(EulerZYX)))
}

func getRotationOffset(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "RotationOffset", floatgeom.Point3{})
}

func getRotationPivot(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "RotationPivot", floatgeom.Point3{})
}

func getPostRotation(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "PostRotation", floatgeom.Point3{})
}

func getScalingOffset(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "ScalingOffset", floatgeom.Point3{})
}

func getScalingPivot(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "ScalingPivot", floatgeom.Point3{})
}

func getPreRotation(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "PreRotation", floatgeom.Point3{})
}

func getLocalTranslation(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "Lcl Translation", floatgeom.Point3{})
}

func getLocalRotation(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "Lcl Rotation", floatgeom.Point3{})
}

func getLocalScaling(o Obj) floatgeom.Point3 {
	return resolveVec3Property(o, "Lcl Scaling", floatgeom.Point3{1, 1, 1})
}

func getGlobalTransform(o Obj) Matrix {
	parent := getParent(o)
	lmt := evalLocal(o, getLocalTranslation(o), getLocalRotation(o))
	if parent == nil {
		return lmt
	}
	mt := getGlobalTransform(parent)

	return mt.Mul(lmt)
}

func GetLocalTransform(o Obj) Matrix {
	return evalLocalScaling(o, getLocalTranslation(o), getLocalRotation(o), getLocalScaling(o))
}

func evalLocal(o Obj, translation, rotation floatgeom.Point3) Matrix {
	return evalLocalScaling(o, translation, rotation, getLocalScaling(o))
}

func evalLocalScaling(o Obj, translation, rotation, scaling floatgeom.Point3) Matrix {
	rotationPivot := getRotationPivot(o)
	scalingPivot := getScalingPivot(o)
	rotationOrder := getRotationOrder(o)

	s := makeIdentity()
	s.m[0] = scaling.X()
	s.m[5] = scaling.Y()
	s.m[10] = scaling.Z()

	t := makeIdentity()
	setTranslation(translation, &t)

	// 关键修复：使用相同的旋转顺序处理所有旋转
	r := rotationOrder.rotationMatrix(rotation)

	// 使用相同的旋转顺序处理preRotation
	pr := getPreRotation(o) // 度数转弧度
	rPre := rotationOrder.rotationMatrix(pr)

	// 使用相同的旋转顺序处理postRotation
	psr := getPostRotation(o)
	rPost := rotationOrder.rotationMatrix(psr)
	rPostInv := rPost.Transposed() // 逆矩阵

	rOff := makeIdentity()
	setTranslation(getRotationOffset(o), &rOff)

	rP := makeIdentity()
	setTranslation(rotationPivot, &rP)

	rPInv := makeIdentity()
	setTranslation(rotationPivot.MulConst(-1), &rPInv)

	sOff := makeIdentity()
	setTranslation(getScalingOffset(o), &sOff)

	sP := makeIdentity()
	setTranslation(scalingPivot, &sP)

	sPInv := makeIdentity()
	setTranslation(scalingPivot.MulConst(-1), &sPInv)

	// 修正后的变换顺序 - 与JS版本一致:
	// T * Roff * Rp * Rpre * R * RpostInv * RpInv * Soff * Sp * S * SpInv
	return t.
		Mul(rOff).
		Mul(rP).
		Mul(rPre).
		Mul(r).
		Mul(rPostInv).
		Mul(rPInv).
		Mul(sOff).
		Mul(sP).
		Mul(s).
		Mul(sPInv)
}

func GetGlobalMatrix(o Obj) Matrix {
	return getGlobalTransform(o)
}
