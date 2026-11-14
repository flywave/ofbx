package ofbx

import (
	"fmt"
)

// Blend shapes (morph targets): a BlendShape hangs off a geometry, holds
// BlendShapeChannels, and each channel holds Shape deltas (control point
// indices plus position offsets applied at the channel's weight).

// Shape is one morph target: sparse control point position deltas.
type Shape struct {
	Object
	// Indices lists the affected control points.
	Indices []int
	// Vertices holds one position delta per index.
	Vertices []Point3
}

// NewShape makes a stub Shape object.
func NewShape(scene *Scene, element *Element) *Shape {
	s := &Shape{}
	s.Object = *NewObject(scene, element)
	return s
}

// Type returns SHAPE.
func (s *Shape) Type() Type { return SHAPE }

func (s *Shape) String() string { return s.stringPrefix("") }

func (s *Shape) stringPrefix(prefix string) string {
	return fmt.Sprintf("%sShape: %s (%d deltas)\n", prefix, s.Name(), len(s.Indices))
}

func parseShape(scene *Scene, element *Element) (*Shape, error) {
	shape := NewShape(scene, element)
	if p := findChildProperty(element, "Indexes"); len(p) > 0 {
		idx, err := parseBinaryArrayInt(p[0])
		if err != nil {
			return nil, fmt.Errorf("ofbx: shape indexes: %w", err)
		}
		shape.Indices = idx
	}
	if p := findChildProperty(element, "Vertices"); len(p) > 0 {
		verts, err := parseDoubleVecDataVec3(p[0])
		if err != nil {
			return nil, fmt.Errorf("ofbx: shape vertices: %w", err)
		}
		shape.Vertices = verts
	}
	return shape, nil
}

// BlendShapeChannel weights a group of shapes.
type BlendShapeChannel struct {
	Object
	// Weight is the current deform percent (0-100).
	Weight float64
	Shapes []*Shape
}

// NewBlendShapeChannel makes a stub channel object.
func NewBlendShapeChannel(scene *Scene, element *Element) *BlendShapeChannel {
	c := &BlendShapeChannel{}
	c.Object = *NewObject(scene, element)
	return c
}

// Type returns BLEND_SHAPE_CHANNEL.
func (c *BlendShapeChannel) Type() Type { return BLEND_SHAPE_CHANNEL }

func (c *BlendShapeChannel) String() string { return c.stringPrefix("") }

func (c *BlendShapeChannel) stringPrefix(prefix string) string {
	s := fmt.Sprintf("%sBlendShapeChannel: %s weight=%v (%d shapes)\n",
		prefix, c.Name(), c.Weight, len(c.Shapes))
	for _, sh := range c.Shapes {
		s += sh.stringPrefix(prefix + "\t")
	}
	return s
}

func parseBlendShapeChannel(scene *Scene, element *Element) *BlendShapeChannel {
	ch := NewBlendShapeChannel(scene, element)
	ch.Weight = propFloat(resolvePropertyP(element, "DeformPercent"))
	return ch
}

// resolvePropertyP finds a named P entry's value property in an element's
// Properties70 (element-level variant of resolveProperty).
func resolvePropertyP(element *Element, name string) *Property {
	if element == nil {
		return nil
	}
	for _, p70 := range element.ChildrenOf("Properties70") {
		for _, p := range p70.Children {
			if prop := p.getProperty(0); prop != nil && prop.value.String() == name {
				if len(p.Properties) > 4 {
					return p.getProperty(4)
				}
			}
		}
	}
	return nil
}

// BlendShape is a morph deformer attached to a geometry.
type BlendShape struct {
	Object
	Channels []*BlendShapeChannel
}

// NewBlendShape makes a stub BlendShape object.
func NewBlendShape(scene *Scene, element *Element) *BlendShape {
	b := &BlendShape{}
	b.Object = *NewObject(scene, element)
	return b
}

// Type returns BLEND_SHAPE.
func (b *BlendShape) Type() Type { return BLEND_SHAPE }

func (b *BlendShape) String() string { return b.stringPrefix("") }

func (b *BlendShape) stringPrefix(prefix string) string {
	s := fmt.Sprintf("%sBlendShape: %s (%d channels)\n", prefix, b.Name(), len(b.Channels))
	for _, c := range b.Channels {
		s += c.stringPrefix(prefix + "\t")
	}
	return s
}

// ---------- document builders ----------

// NewBlendShape attaches an empty blend shape deformer to a mesh's geometry.
func (d *Document) NewBlendShape(m *Mesh, name string) (*BlendShape, error) {
	if m == nil || m.Geometry == nil {
		return nil, fmt.Errorf("ofbx: blend shape needs a mesh with geometry")
	}
	el := NewElement("Deformer")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Deformer"))
	el.AddProperty(NewStringProperty("BlendShape"))

	bs := NewBlendShape(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, bs)
	d.objs.AddChild(el)
	m.Geometry.BlendShapes = append(m.Geometry.BlendShapes, bs)
	d.connect(bs, m.Geometry, "")
	d.countType("Deformer")
	return bs, nil
}

// NewBlendShapeChannel adds a channel with the given default weight to a
// blend shape.
func (d *Document) NewBlendShapeChannel(bs *BlendShape, name string, weight float64) *BlendShapeChannel {
	el := NewElement("Deformer")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Deformer"))
	el.AddProperty(NewStringProperty("BlendShapeChannel"))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pNumber("DeformPercent", weight))

	ch := NewBlendShapeChannel(d.scene, el)
	ch.Weight = weight
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, ch)
	d.objs.AddChild(el)
	bs.Channels = append(bs.Channels, ch)
	d.connect(ch, bs, "")
	d.countType("Deformer")
	return ch
}

// AddShape adds a morph target to a channel. indices reference control
// points; deltas holds one position offset per index.
func (d *Document) AddShape(ch *BlendShapeChannel, name string, indices []int, deltas []Point3) (*Shape, error) {
	if len(indices) != len(deltas) {
		return nil, fmt.Errorf("ofbx: %d indices but %d deltas", len(indices), len(deltas))
	}
	idxProp, err := NewInt32ArrayProperty(ints32(indices), d.CompressArrays)
	if err != nil {
		return nil, err
	}
	vertsProp, err := NewDoubleArrayProperty(flattenPoint3s(deltas), d.CompressArrays)
	if err != nil {
		return nil, err
	}
	el := NewElement("Geometry")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Geometry"))
	el.AddProperty(NewStringProperty("Shape"))
	el.AddChildElement("Indexes", idxProp)
	el.AddChildElement("Vertices", vertsProp)
	el.AddChildElement("GeometryVersion", NewInt32Property(124))

	shape, err := parseShape(d.scene, el)
	if err != nil {
		return nil, err
	}
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, shape)
	d.objs.AddChild(el)
	ch.Shapes = append(ch.Shapes, shape)
	d.connect(shape, ch, "")
	d.countType("Geometry")
	return shape, nil
}
