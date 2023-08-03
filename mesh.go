package ofbx

import (
	"fmt"
)

// Mesh is a geometry made of polygon
// https://help.autodesk.com/view/FBX/2017/ENU/?guid=__cpp_ref_class_fbx_mesh_html
type Mesh struct {
	Object
	Geometry  *Geometry
	Materials []*Material
}

// NewMesh cretes a new stub Object
func NewMesh(scene *Scene, element *Element) *Mesh {
	m := &Mesh{}
	m.Object = *NewObject(scene, element)
	m.Object.isNode = true
	return m
}

// Animations returns the Animation Stacks connected to this mesh
func (m *Mesh) Animations() []*AnimationStack {
	anims := m.scene.AnimationStacks
	out := []*AnimationStack{}

	animatableIds := map[uint64]bool{}
	animatableIds[m.ID()] = true

	for _, cluster := range m.Geometry.Skin.Clusters {

		animatableIds[cluster.Link.ID()] = true

	}

ANIMLOOP:
	for _, a := range anims {
		for _, l := range a.Layers {
			fmt.Println("in Layer ", l.id)
			for _, c := range l.CurveNodes {
				if _, ok := animatableIds[c.Bone.ID()]; ok {
					out = append(out, a)
					continue ANIMLOOP
				}
			}
		}
	}
	return out
}

// Type returns MESH
func (m *Mesh) Type() Type {
	return MESH
}

func (m *Mesh) GetGeometricMatrix() Matrix {
	translation := getLocalTranslation(m)
	rotation := getLocalRotation(m)
	scale := getLocalScaling(m)

	scaleMtx := makeIdentity()
	scaleMtx.m[0] = scale.X()
	scaleMtx.m[5] = scale.Y()
	scaleMtx.m[10] = scale.Z()
	mtx := EulerXYZ.rotationMatrix(rotation)
	setTranslation(translation, &mtx)

	return scaleMtx.Mul(mtx)
}

func (m *Mesh) String() string {
	return m.stringPrefix("")
}

func (m *Mesh) stringPrefix(prefix string) string {
	s := prefix + "Mesh:" + fmt.Sprintf("%v", m.ID()) + "\n"
	s += m.Geometry.stringPrefix(prefix + "\t")
	for _, mat := range m.Materials {
		s += "\n"
		s += mat.stringPrefix(prefix + "\t")
	}
	return s
}
