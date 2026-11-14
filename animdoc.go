package ofbx

import (
	"errors"
	"fmt"
	"time"
)

// Animation and skinning constructors for Document. The FBX animation graph
// is Objects { AnimationStack > AnimationLayer > AnimationCurveNode >
// AnimationCurve } wired together with connections:
//
//	curve    --OP "d|X"-->  curveNode --OP "Lcl Translation"--> animated node
//	curveNode --OO-->       layer     --OO-->                  stack

// animationKeyAttrFlags is the standard key attribute bitmask
// (cubic interpolation, auto tangents); see the FBX SDK KFCurve flags.
const animationKeyAttrFlags = 24836

// takeEntry remembers a created animation stack for the Takes section.
type takeEntry struct {
	name                  string
	localStart, localStop int64
}

// NewAnimationStack creates an animation stack. LocalStart/LocalStop bound
// the stack in FBX time (KTime units).
func (d *Document) NewAnimationStack(name string, localStart, localStop int64) *AnimationStack {
	el := NewElement("AnimationStack")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01AnimStack"))
	el.AddProperty(NewStringProperty(""))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pKTime("LocalStart", localStart))
	p70.AddChild(pKTime("LocalStop", localStop))
	p70.AddChild(pKTime("ReferenceStart", localStart))
	p70.AddChild(pKTime("ReferenceStop", localStop))

	stack := NewAnimationStack(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, stack)
	d.objs.AddChild(el)
	d.scene.AnimationStacks = append(d.scene.AnimationStacks, stack)
	d.animTakes = append(d.animTakes, takeEntry{name: name, localStart: localStart, localStop: localStop})
	d.countType("AnimationStack")
	return stack
}

// syncTakes writes the legacy Takes section for the created animation
// stacks (still read by many importers to enumerate takes). Scenes exported
// through SaveScene keep their original Takes subtree.
func (d *Document) syncTakes() {
	if len(d.animTakes) == 0 {
		return
	}
	d.root.RemoveChild("Takes")
	takes := d.root.AddChildElement("Takes")
	takes.AddChildElement("Current", NewStringProperty(d.animTakes[0].name))
	for _, tk := range d.animTakes {
		take := takes.AddChildElement("Take", NewStringProperty(tk.name))
		take.AddChildElement("FileName", NewStringProperty(takeFileName(tk.name)))
		take.AddChildElement("LocalTime", NewInt64Property(tk.localStart), NewInt64Property(tk.localStop))
		take.AddChildElement("ReferenceTime", NewInt64Property(tk.localStart), NewInt64Property(tk.localStop))
	}
}

// takeFileName derives the conventional .tak file name for a take.
func takeFileName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		if r == ' ' {
			r = '_'
		}
		out = append(out, r)
	}
	return string(out) + ".tak"
}

// NewAnimationLayer creates an animation layer inside the stack.
func (d *Document) NewAnimationLayer(name string, stack *AnimationStack) *AnimationLayer {
	el := NewElement("AnimationLayer")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01AnimLayer"))
	el.AddProperty(NewStringProperty(""))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pNumber("Weight", 100))
	p70.AddChild(pInt("Mute", 0))

	layer := NewAnimationLayer(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, layer)
	d.objs.AddChild(el)
	stack.Layers = append(stack.Layers, layer)
	d.connect(layer, stack, "")
	d.countType("AnimationLayer")
	return layer
}

// NewAnimationCurve creates a curve from (time, value) keys. Times may use
// any duration unit; they are stored as FBX KTime (46186158000 per second).
func (d *Document) NewAnimationCurve(name string, times []time.Duration, values []float32) (*AnimationCurve, error) {
	if len(times) != len(values) {
		return nil, fmt.Errorf("ofbx: %d times but %d values", len(times), len(values))
	}
	if len(times) == 0 {
		return nil, errors.New("ofbx: empty animation curve")
	}

	keyTimes := make([]int64, len(times))
	for i, t := range times {
		keyTimes[i] = stdTimeToFbxTime(t)
	}
	keyAttrData := make([]float32, 9*len(times)) // zeroed tangents; importers recompute
	keyAttrRefCount := make([]int64, len(times))
	for i := range keyAttrRefCount {
		keyAttrRefCount[i] = 1
	}

	timeProp, err := NewInt64ArrayProperty(keyTimes, d.CompressArrays)
	if err != nil {
		return nil, err
	}
	valueProp, err := NewFloatArrayProperty(values, d.CompressArrays)
	if err != nil {
		return nil, err
	}
	flagsProp, err := NewInt64ArrayProperty(repeatInt64(animationKeyAttrFlags, len(times)), d.CompressArrays)
	if err != nil {
		return nil, err
	}
	attrDataProp, err := NewFloatArrayProperty(keyAttrData, d.CompressArrays)
	if err != nil {
		return nil, err
	}
	refCountProp, err := NewInt64ArrayProperty(keyAttrRefCount, d.CompressArrays)
	if err != nil {
		return nil, err
	}

	el := NewElement("AnimationCurve")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01AnimCurve"))
	el.AddProperty(NewStringProperty(""))
	el.AddChildElement("KeyTime", timeProp)
	el.AddChildElement("KeyValueFloat", valueProp)
	el.AddChildElement("KeyAttrFlags", flagsProp)
	el.AddChildElement("KeyAttrDataFloat", attrDataProp)
	el.AddChildElement("KeyAttrRefCount", refCountProp)

	curve, err := parseAnimationCurve(d.scene, el)
	if err != nil {
		return nil, err
	}
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, curve)
	d.objs.AddChild(el)
	d.countType("AnimationCurve")
	return curve, nil
}

// NewAnimationCurveNode binds up to three curves (X/Y/Z) to one property of
// an animated object. property is the FBX property name, e.g.
// "Lcl Translation". Curves may be nil. The special property "FocalLength"
// binds a single curve instead of an xyz triple.
func (d *Document) NewAnimationCurveNode(name string, layer *AnimationLayer, target Obj, property string, curves [3]*AnimationCurve) (*AnimationCurveNode, error) {
	if target == nil || target.Element() == nil {
		return nil, errors.New("ofbx: animation curve node needs a target object")
	}

	el := NewElement("AnimationCurveNode")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01AnimCurveNode"))
	el.AddProperty(NewStringProperty(""))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pNumber("d|X", 1))
	p70.AddChild(pNumber("d|Y", 1))
	p70.AddChild(pNumber("d|Z", 1))

	node := NewAnimationCurveNode(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, node)
	d.objs.AddChild(el)

	node.Bone = target
	node.BoneLinkProp = property
	if property == "FocalLength" {
		if curves[0] != nil {
			node.Curves[0].Curve = curves[0]
			d.connect(curves[0], node, "FocalLength")
		}
	} else {
		for i, curve := range curves {
			if curve == nil {
				continue
			}
			node.Curves[i].Curve = curve
			d.connect(curve, node, [3]string{"d|X", "d|Y", "d|Z"}[i])
		}
	}
	d.connect(node, layer, "")
	d.connect(node, target, property)
	d.countType("AnimationCurveNode")
	return node, nil
}

// ---------- skinning ----------

// NewSkin creates a skin deformer bound to a mesh's geometry.
func (d *Document) NewSkin(m *Mesh) (*Skin, error) {
	if m == nil || m.Geometry == nil {
		return nil, errors.New("ofbx: skin needs a mesh with geometry")
	}
	el := NewElement("Deformer")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(m.Name() + "\x00\x01Deformer"))
	el.AddProperty(NewStringProperty("Skin"))
	el.AddChildElement("Version", NewInt32Property(101))

	skin := NewSkin(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, skin)
	d.objs.AddChild(el)
	m.Geometry.Skin = skin
	d.connect(skin, m.Geometry, "")
	d.countType("Deformer")
	return skin, nil
}

// NewCluster adds a cluster to a skin. indices reference control points of
// the skinned geometry; weights are parallel to indices. transformLink is
// the bone's bind pose matrix in mesh space (row-major 4x4); an optional
// transform argument supplies the mesh's bind pose matrix.
func (d *Document) NewCluster(skin *Skin, link Obj, indices []int, weights []float64, transformLink Matrix, transform ...Matrix) (*Cluster, error) {
	if skin == nil {
		return nil, errors.New("ofbx: cluster needs a skin")
	}
	if link == nil || link.Element() == nil {
		return nil, errors.New("ofbx: cluster needs a link (bone) object")
	}
	if len(indices) != len(weights) {
		return nil, fmt.Errorf("ofbx: %d indices but %d weights", len(indices), len(weights))
	}

	idxProp, err := NewInt32ArrayProperty(ints32(indices), d.CompressArrays)
	if err != nil {
		return nil, err
	}
	weightProp, err := NewDoubleArrayProperty(weights, d.CompressArrays)
	if err != nil {
		return nil, err
	}
	tlArr := transformLink.ToArray()
	linkProp, err := NewDoubleArrayProperty(tlArr[:], d.CompressArrays)
	if err != nil {
		return nil, err
	}

	el := NewElement("Deformer")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty("Cluster\x00\x01Cluster"))
	el.AddProperty(NewStringProperty("Cluster"))
	el.AddChildElement("Indexes", idxProp)
	el.AddChildElement("Weights", weightProp)
	el.AddChildElement("TransformLink", linkProp)
	if len(transform) > 0 {
		tArr := transform[0].ToArray()
		tProp, err := NewDoubleArrayProperty(tArr[:], d.CompressArrays)
		if err != nil {
			return nil, err
		}
		el.AddChildElement("Transform", tProp)
	}

	cluster, err := parseCluster(d.scene, el)
	if err != nil {
		return nil, err
	}
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, cluster)
	d.objs.AddChild(el)

	cluster.Skin = skin
	cluster.Link = link
	skin.Clusters = append(skin.Clusters, cluster)
	d.connect(cluster, skin, "")
	d.connect(link, cluster, "")

	// run the parser's post-processing (index remapping) now that the
	// connections are registered
	if !cluster.postProcess() {
		return nil, errors.New("ofbx: cluster postprocessing failed")
	}
	d.countType("Deformer")
	return cluster, nil
}

// ---------- time helpers ----------

// stdTimeToFbxTime converts a duration to FBX KTime units
// (46186158000 ticks per second).
func stdTimeToFbxTime(t time.Duration) int64 {
	return int64(float64(t) * 46186158000 / float64(time.Second))
}

func repeatInt64(v int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func ints32(in []int) []int32 {
	out := make([]int32, len(in))
	for i, v := range in {
		out[i] = int32(v)
	}
	return out
}
