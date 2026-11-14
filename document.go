package ofbx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"
)

// A Document assembles a complete FBX element tree for writing. Objects are
// created through the New* constructors which allocate element IDs, register
// typed objects and record Connections; Save serializes the finished
// document.
//
//	doc := ofbx.NewDocument()
//	mesh := doc.NewMesh("Cube", verts, faces)
//	doc.SetNormals(mesh, normals)
//	mat := doc.NewMaterial("Skin")
//	doc.AddMaterial(mesh, mat)
//	err := doc.Save(file)
type Document struct {
	// Version is the FBX container version written to the file header.
	// The default (DefaultWriteVersion, 7400) is accepted by every importer.
	Version uint32
	// Creator is recorded in FBXHeaderExtension and the top level Creator node.
	Creator string
	// Settings are written to GlobalSettings. The zero value is replaced by
	// FBX standard defaults (Y up, Z front, X right, centimeters) at Save
	// time unless WithRawSettings is used.
	Settings Settings
	// CompressArrays zlib-compresses large array properties when building
	// new elements.
	CompressArrays bool

	root    *Element
	header  *Element // FBXHeaderExtension
	globals *Element // GlobalSettings
	defs    *Element // Definitions
	objs    *Element // Objects
	conns   *Element // Connections

	defCounts map[string]int
	nextID    uint64
	// rawSettings suppresses applying FBX standard defaults at Save time.
	rawSettings bool
	// onlySupported filters SaveScene output to modeled object types.
	onlySupported bool

	// scene holds the typed graph of created objects so that link resolution
	// keeps working without a re-parse.
	scene *Scene

	materials []*Material
	// animTakes records created animation stacks for the Takes section.
	animTakes []takeEntry
}

// DocOption configures a NewDocument call.
type DocOption func(*Document)

// WithVersion overrides the written FBX version.
func WithVersion(v uint32) DocOption {
	return func(d *Document) { d.Version = v }
}

// WithCreator overrides the creator string.
func WithCreator(c string) DocOption {
	return func(d *Document) { d.Creator = c }
}

// WithCompressedArrays enables zlib array compression for newly built arrays.
func WithCompressedArrays() DocOption {
	return func(d *Document) { d.CompressArrays = true }
}

// WithRawSettings keeps the Settings zero value instead of applying FBX
// standard defaults.
func WithRawSettings() DocOption {
	return func(d *Document) { d.rawSettings = true }
}

// OnlySupportedObjects affects SaveScene: it drops objects whose element
// types this library does not model (e.g. Pose, Video, Camera). By default
// all objects are kept.
func OnlySupportedObjects() DocOption {
	return func(d *Document) { d.onlySupported = true }
}

// NewDocument creates a document with an empty FBX skeleton
// (FBXHeaderExtension, GlobalSettings, Definitions, Objects, Connections).
func NewDocument(opts ...DocOption) *Document {
	d := &Document{
		Version:   DefaultWriteVersion,
		Creator:   "ofbx",
		nextID:    100,
		defCounts: map[string]int{},
		scene:     &Scene{ObjectMap: map[uint64]Obj{}},
	}
	for _, o := range opts {
		o(d)
	}

	now := time.Now()
	d.root = NewElement("")

	d.header = d.root.AddChildElement("FBXHeaderExtension")
	d.header.AddChildElement("FBXHeaderVersion", NewInt32Property(1003))
	d.header.AddChildElement("FBXVersion", NewInt32Property(int32(d.Version)))
	d.header.AddChildElement("EncryptionType", NewInt32Property(0))
	ts := d.header.AddChildElement("CreationTimeStamp")
	ts.AddChildElement("Version", NewInt32Property(1000))
	ts.AddChildElement("Year", NewInt32Property(int32(now.Year())))
	ts.AddChildElement("Month", NewInt32Property(int32(int(now.Month()))))
	ts.AddChildElement("Day", NewInt32Property(int32(now.Day())))
	ts.AddChildElement("Hour", NewInt32Property(int32(now.Hour())))
	ts.AddChildElement("Minute", NewInt32Property(int32(now.Minute())))
	ts.AddChildElement("Second", NewInt32Property(int32(now.Second())))
	ts.AddChildElement("Millisecond", NewInt32Property(0))
	d.header.AddChildElement("Creator", NewStringProperty(d.Creator))

	d.root.AddChildElement("Creator", NewStringProperty(d.Creator))

	d.globals = d.root.AddChildElement("GlobalSettings")
	d.globals.AddChildElement("Version", NewInt32Property(1000))
	d.globals.AddChildElement("Properties70")

	d.defs = d.root.AddChildElement("Definitions")
	d.defs.AddChildElement("Version", NewInt32Property(100))
	d.defs.AddChildElement("Count", NewInt32Property(0))

	d.objs = d.root.AddChildElement("Objects")
	d.conns = d.root.AddChildElement("Connections")

	return d
}

// Scene returns the typed object graph of the objects created so far.
func (d *Document) Scene() *Scene {
	return d.scene
}

// Root returns the document's element tree. Call Prepare first when the
// document was not saved yet but the finished tree is needed (e.g. for
// EncodeText).
func (d *Document) Root() *Element {
	return d.root
}

// allocID hands out the next unique object id. 0 is reserved for the scene
// root and never returned.
func (d *Document) allocID() uint64 {
	d.nextID++
	return d.nextID
}

func (d *Document) countType(name string) {
	d.defCounts[name]++
}

func (d *Document) connect(from, to Obj, property string) {
	c := Connection{from: from.ID(), to: to.ID()}
	if property == "" {
		c.typ = ObjectConn
		d.conns.AddChildElement("C",
			NewStringProperty("OO"), NewInt64Property(int64(c.from)), NewInt64Property(int64(c.to)))
	} else {
		c.typ = PropConn
		c.property = property
		d.conns.AddChildElement("C",
			NewStringProperty("OP"), NewInt64Property(int64(c.from)), NewInt64Property(int64(c.to)),
			NewStringProperty(property))
	}
	d.scene.Connections = append(d.scene.Connections, c)
}

// connectToRoot links an object to the scene root (object id 0).
func (d *Document) connectToRoot(o Obj) {
	c := Connection{from: o.ID(), typ: ObjectConn}
	d.conns.AddChildElement("C",
		NewStringProperty("OO"), NewInt64Property(int64(o.ID())), NewInt64Property(0))
	d.scene.Connections = append(d.scene.Connections, c)
}

// register stores a created object in the document scene.
func (d *Document) register(id uint64, o Obj) {
	o.SetID(id)
	d.scene.ObjectMap[id] = o
}

// ---------- meshes ----------

// NewMesh creates a Model+Geometry pair. vertices holds one entry per control
// point; faces holds vertex-index polygons (triangles or n-gons).
func (d *Document) NewMesh(name string, vertices []Point3, faces [][]int) (*Mesh, error) {
	if len(vertices) == 0 {
		return nil, errors.New("ofbx: mesh needs at least one vertex")
	}
	for _, f := range faces {
		if len(f) < 3 {
			return nil, fmt.Errorf("ofbx: face with %d vertices in mesh %q", len(f), name)
		}
		for _, idx := range f {
			if idx < 0 || idx >= len(vertices) {
				return nil, fmt.Errorf("ofbx: face index %d out of range in mesh %q", idx, name)
			}
		}
	}

	geomEl, err := d.buildGeometryElement(name, vertices, faces)
	if err != nil {
		return nil, err
	}
	geom, err := parseGeometry(d.scene, geomEl)
	if err != nil {
		return nil, err
	}
	geomID := d.allocID()
	geomEl.Properties[0] = NewInt64Property(int64(geomID))
	geomEl.Properties[1] = NewStringProperty(name + "\x00\x01Geometry")
	d.register(geomID, geom)
	d.objs.AddChild(geomEl)

	modelEl := NewElement("Model")
	modelEl.AddProperty(NewInt64Property(0)) // placeholder id
	modelEl.AddProperty(NewStringProperty(name + "\x00\x01Model"))
	modelEl.AddProperty(NewStringProperty("Mesh"))
	modelEl.AddChildElement("Version", NewInt32Property(232))
	p70 := modelEl.AddChildElement("Properties70")
	p70.AddChild(pLclVec3("Lcl Translation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Rotation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Scaling", 1, 1, 1))
	p70.AddChild(pInt("DefaultAttributeIndex", 0))
	modelEl.AddChildElement("Shading", NewBoolProperty(true))
	modelEl.AddChildElement("Culling", NewStringProperty("CullingOff"))

	mesh := NewMesh(d.scene, modelEl)
	meshID := d.allocID()
	modelEl.Properties[0] = NewInt64Property(int64(meshID))
	d.register(meshID, mesh)
	d.objs.AddChild(modelEl)
	d.scene.Meshes = append(d.scene.Meshes, mesh)
	mesh.Geometry = geom

	d.connect(geom, mesh, "")
	d.connectToRoot(mesh)
	d.countType("Geometry")
	d.countType("Model")
	return mesh, nil
}

// buildGeometryElement assembles the Geometry element for a mesh. The first
// property (the object id) is a placeholder fixed up by the caller.
func (d *Document) buildGeometryElement(name string, vertices []Point3, faces [][]int) (*Element, error) {
	flat := make([]float64, 0, 3*len(vertices))
	for _, v := range vertices {
		flat = append(flat, v.X(), v.Y(), v.Z())
	}
	indices := make([]int32, 0, countFaceVerts(faces))
	for _, f := range faces {
		for i, idx := range f {
			if i == len(f)-1 {
				indices = append(indices, int32(-idx-1)) // negative-encoded polygon end
			} else {
				indices = append(indices, int32(idx))
			}
		}
	}

	vertsProp, err := NewDoubleArrayProperty(flat, d.CompressArrays)
	if err != nil {
		return nil, err
	}
	idxProp, err := NewInt32ArrayProperty(indices, d.CompressArrays)
	if err != nil {
		return nil, err
	}

	el := NewElement("Geometry")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Geometry"))
	el.AddProperty(NewStringProperty("Mesh"))
	el.AddChildElement("Vertices", vertsProp)
	el.AddChildElement("PolygonVertexIndex", idxProp)
	el.AddChildElement("GeometryVersion", NewInt32Property(124))
	return el, nil
}

func countFaceVerts(faces [][]int) int {
	n := 0
	for _, f := range faces {
		n += len(f)
	}
	return n
}

// geometryElement returns the (already registered) Geometry element of a
// document-created mesh.
func (d *Document) geometryElement(m *Mesh) (*Element, error) {
	if m == nil || m.Geometry == nil || m.Geometry.Element() == nil {
		return nil, errors.New("ofbx: mesh has no geometry")
	}
	return m.Geometry.Element(), nil
}

// SetNormals attaches a LayerElementNormal with ByPolygonVertex/Direct
// mapping. normals must hold one entry per polygon vertex (corner).
func (d *Document) SetNormals(m *Mesh, normals []Point3) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	if expected := polygonVertexCount(geomEl); len(normals) != expected {
		return fmt.Errorf("ofbx: got %d normals, mesh has %d polygon vertices", len(normals), expected)
	}
	flat := flattenPoint3s(normals)
	data, err := NewDoubleArrayProperty(flat, d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementNormal", NewInt64Property(0))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty(""))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	layer.AddChildElement("Normals", data)
	d.addLayerEntry(geomEl, "LayerElementNormal")
	return nil
}

// SetUVs attaches a LayerElementUV with ByPolygonVertex/Direct mapping.
func (d *Document) SetUVs(m *Mesh, set int, uvs []Point2) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	if set < 0 || set >= MaxUvs {
		return fmt.Errorf("ofbx: uv set %d out of range", set)
	}
	if expected := polygonVertexCount(geomEl); len(uvs) != expected {
		return fmt.Errorf("ofbx: got %d uvs, mesh has %d polygon vertices", len(uvs), expected)
	}
	flat := make([]float64, 0, 2*len(uvs))
	for _, uv := range uvs {
		flat = append(flat, uv.X(), uv.Y())
	}
	data, err := NewDoubleArrayProperty(flat, d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementUV", NewInt64Property(int64(set)))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty("UVMap"))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	layer.AddChildElement("UV", data)
	d.addLayerEntry(geomEl, "LayerElementUV")
	return nil
}

// SetUVsWithIndex attaches a LayerElementUV using IndexToDirect referencing:
// uvs holds unique coordinates and uvIndex one entry per polygon vertex.
func (d *Document) SetUVsWithIndex(m *Mesh, set int, uvs []Point2, uvIndex []int32) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	if set < 0 || set >= MaxUvs {
		return fmt.Errorf("ofbx: uv set %d out of range", set)
	}
	if expected := polygonVertexCount(geomEl); len(uvIndex) != expected {
		return fmt.Errorf("ofbx: got %d uv indices, mesh has %d polygon vertices", len(uvIndex), expected)
	}
	for _, idx := range uvIndex {
		if idx < 0 || int(idx) >= len(uvs) {
			return fmt.Errorf("ofbx: uv index %d out of range (%d uvs)", idx, len(uvs))
		}
	}
	flat := make([]float64, 0, 2*len(uvs))
	for _, uv := range uvs {
		flat = append(flat, uv.X(), uv.Y())
	}
	data, err := NewDoubleArrayProperty(flat, d.CompressArrays)
	if err != nil {
		return err
	}
	idxProp, err := NewInt32ArrayProperty(uvIndex, d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementUV", NewInt64Property(int64(set)))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty("UVMap"))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("IndexToDirect"))
	layer.AddChildElement("UV", data)
	layer.AddChildElement("UVIndex", idxProp)
	d.addLayerEntry(geomEl, "LayerElementUV")
	return nil
}

// SetNormalsByVertex attaches per-control-point normals (ByVertice/Direct).
func (d *Document) SetNormalsByVertex(m *Mesh, normals []Point3) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	if m.Geometry == nil || len(normals) != len(m.Geometry.Vertices) {
		return fmt.Errorf("ofbx: got %d normals, mesh has %d control points", len(normals), len(m.Geometry.Vertices))
	}
	data, err := NewDoubleArrayProperty(flattenPoint3s(normals), d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementNormal", NewInt64Property(0))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty(""))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByVertice"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	layer.AddChildElement("Normals", data)
	d.addLayerEntry(geomEl, "LayerElementNormal")
	return nil
}

// SetVertexColors attaches a LayerElementColor with ByPolygonVertex/Direct
// mapping. Each color entry is (r, g, b, a).
func (d *Document) SetVertexColors(m *Mesh, colors []Point4) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	if expected := polygonVertexCount(geomEl); len(colors) != expected {
		return fmt.Errorf("ofbx: got %d colors, mesh has %d polygon vertices", len(colors), expected)
	}
	flat := make([]float64, 0, 4*len(colors))
	for _, c := range colors {
		flat = append(flat, c[0], c[1], c[2], c[3])
	}
	data, err := NewDoubleArrayProperty(flat, d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementColor", NewInt64Property(0))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty(""))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	layer.AddChildElement("Colors", data)
	d.addLayerEntry(geomEl, "LayerElementColor")
	return nil
}

// SetMaterialIndices attaches a LayerElementMaterial. mapping must be
// "AllSame" (indices holds a single entry) or "ByPolygon" (one entry per
// polygon).
func (d *Document) SetMaterialIndices(m *Mesh, mapping string, indices []int32) error {
	geomEl, err := d.geometryElement(m)
	if err != nil {
		return err
	}
	switch mapping {
	case "AllSame":
		if len(indices) != 1 {
			return errors.New(`ofbx: AllSame mapping needs exactly one index`)
		}
	case "ByPolygon":
		want := len(m.Geometry.Faces)
		if len(indices) != want {
			return fmt.Errorf("ofbx: got %d material indices, mesh has %d polygons", len(indices), want)
		}
	default:
		return fmt.Errorf("ofbx: unsupported material mapping %q", mapping)
	}
	data, err := NewInt32ArrayProperty(indices, d.CompressArrays)
	if err != nil {
		return err
	}
	layer := geomEl.AddChildElement("LayerElementMaterial", NewInt64Property(0))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty(""))
	layer.AddChildElement("MappingInformationType", NewStringProperty(mapping))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("IndexToDirect"))
	layer.AddChildElement("Materials", data)
	d.addLayerEntry(geomEl, "LayerElementMaterial")
	return nil
}

// addLayerEntry records a typed layer in the mesh's Layer: 0 node, creating
// that node on first use.
func (d *Document) addLayerEntry(geomEl *Element, typ string) {
	layer0 := geomEl.Child("Layer")
	if layer0 == nil {
		layer0 = geomEl.AddChildElement("Layer", NewInt64Property(0))
		layer0.AddChildElement("Version", NewInt32Property(100))
	}
	entry := layer0.AddChildElement("LayerElement")
	entry.AddChildElement("Type", NewStringProperty(typ))
	typed := len(layer0.ChildrenOf("LayerElement")) - 1
	entry.AddChildElement("TypedIndex", NewInt32Property(int32(typed)))
}

// polygonVertexCount counts the polygon vertices of a built geometry element.
func polygonVertexCount(geomEl *Element) int {
	props := findChildProperty(geomEl, "PolygonVertexIndex")
	if len(props) == 0 {
		return 0
	}
	return props[0].Count
}

// ---------- materials ----------

// NewMaterial creates a Phong material. Colors and factors are read from the
// returned object's exported fields at Save time, so mutate them freely
// before saving.
func (d *Document) NewMaterial(name string) *Material {
	el := NewElement("Material")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Material"))
	el.AddProperty(NewStringProperty(""))
	el.AddChildElement("Version", NewInt32Property(102))
	el.AddChildElement("ShadingModel", NewStringProperty("phong"))
	el.AddChildElement("MultiLayer", NewInt32Property(0))
	el.AddChildElement("Properties70")

	mat := NewMaterial(d.scene, el)
	mat.DiffuseColor = Color{1, 1, 1}
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, mat)
	d.objs.AddChild(el)
	d.materials = append(d.materials, mat)
	d.countType("Material")
	return mat
}

// AddMaterial attaches a material to a mesh and records the connection.
func (d *Document) AddMaterial(m *Mesh, mat *Material) {
	m.Materials = append(m.Materials, mat)
	d.connect(mat, m, "")
}

// syncMaterial rewrites the material's Properties70 from its typed fields.
func (d *Document) syncMaterial(mat *Material) {
	el := mat.Element()
	el.RemoveChild("Properties70")
	el.AddChild(materialProperties70(mat))
}

// materialProperties70 builds the standard Phong P entry list from a
// material's typed fields.
func materialProperties70(mat *Material) *Element {
	p70 := NewElement("Properties70")
	p70.AddChild(pColor("EmissiveColor", mat.EmissiveColor))
	p70.AddChild(pNumber("EmissiveFactor", mat.EmissiveFactor))
	p70.AddChild(pColor("AmbientColor", mat.AmbientColor))
	p70.AddChild(pColor("DiffuseColor", mat.DiffuseColor))
	p70.AddChild(pNumber("DiffuseFactor", mat.DiffuseFactor))
	p70.AddChild(pColor("TransparentColor", mat.TransparentColor))
	p70.AddChild(pColor("SpecularColor", mat.SpecularColor))
	p70.AddChild(pNumber("SpecularFactor", mat.SpecularFactor))
	p70.AddChild(pNumber("Shininess", mat.Shininess))
	p70.AddChild(pNumber("ShininessExponent", mat.ShininessExponent))
	p70.AddChild(pColor("ReflectionColor", mat.ReflectionColor))
	p70.AddChild(pNumber("ReflectionFactor", mat.ReflectionFactor))
	return p70
}

// ---------- textures ----------

// NewTexture creates a file texture object.
func (d *Document) NewTexture(name, filename, relativeFilename string) *Texture {
	el := NewElement("Texture")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Texture"))
	el.AddProperty(NewStringProperty(""))
	el.AddChildElement("Type", NewStringProperty("TextureVideoClip"))
	el.AddChildElement("Version", NewInt32Property(202))
	el.AddChildElement("TextureName", NewStringProperty("Texture::"+name))
	el.AddChildElement("FileName", NewStringProperty(filename))
	el.AddChildElement("RelativeFilename", NewStringProperty(relativeFilename))
	el.AddChildElement("ModelUVTranslation", NewFloat64Property(0), NewFloat64Property(0))
	el.AddChildElement("ModelUVScaling", NewFloat64Property(1), NewFloat64Property(1))
	el.AddChildElement("Texture_Alpha_Source", NewStringProperty("None"))
	el.AddChildElement("Cropping", NewInt32Property(0), NewInt32Property(0), NewInt32Property(0), NewInt32Property(0))

	tex := parseTexture(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, tex)
	d.objs.AddChild(el)
	d.countType("Texture")
	return tex
}

// SetTexture connects a texture to a material slot ("DiffuseColor" for
// DIFFUSE, "NormalMap" for NORMAL).
func (d *Document) SetTexture(mat *Material, tt TextureType, tex *Texture) error {
	if tt != DIFFUSE && tt != NORMAL {
		return fmt.Errorf("ofbx: unsupported texture slot %d", tt)
	}
	property := "DiffuseColor"
	if tt == NORMAL {
		property = "NormalMap"
	}
	mat.Textures[tt] = tex
	d.connect(tex, mat, property)
	return nil
}

// ---------- nodes ----------

// NewNode creates a Model of the given subclass ("LimbNode", "Null" or
// "Root"). LimbNodes additionally get a Skeleton NodeAttribute so that
// importers recognize them as bones.
func (d *Document) NewNode(name, subclass string) (*Node, error) {
	switch subclass {
	case "LimbNode", "Null", "Root":
	default:
		return nil, fmt.Errorf("ofbx: unsupported node subclass %q", subclass)
	}

	el := NewElement("Model")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Model"))
	el.AddProperty(NewStringProperty(subclass))
	el.AddChildElement("Version", NewInt32Property(232))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pLclVec3("Lcl Translation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Rotation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Scaling", 1, 1, 1))
	el.AddChildElement("Culling", NewStringProperty("CullingOff"))

	nodeType := NULL_NODE
	if subclass == "LimbNode" {
		nodeType = LIMB_NODE
	}
	node := NewNode(d.scene, el, nodeType)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, node)
	d.objs.AddChild(el)
	d.countType("Model")

	if subclass == "LimbNode" {
		attrEl := NewElement("NodeAttribute")
		attrEl.AddProperty(NewInt64Property(0)) // placeholder id
		attrEl.AddProperty(NewStringProperty(name + "\x00\x01NodeAttribute"))
		attrEl.AddProperty(NewStringProperty("LimbNode"))
		attrEl.AddChildElement("TypeFlags", NewStringProperty("Skeleton"))
		attr, err := parseNodeAttribute(d.scene, attrEl)
		if err != nil {
			return nil, err
		}
		attrID := d.allocID()
		attrEl.Properties[0] = NewInt64Property(int64(attrID))
		d.register(attrID, attr)
		d.objs.AddChild(attrEl)
		node.SetNodeAttribute(attr)
		d.connect(attr, node, "")
		d.countType("NodeAttribute")
	}
	return node, nil
}

// SetParent connects a node to a parent node (nil parent attaches to the
// scene root).
func (d *Document) SetParent(child Obj, parent Obj) {
	if parent == nil {
		d.connectToRoot(child)
		return
	}
	d.connect(child, parent, "")
}

// ---------- Properties70 P entries ----------

func pEntry(name, typ, subtype, flag string, values ...*Property) *Element {
	e := NewElement("P")
	e.AddProperty(NewStringProperty(name))
	e.AddProperty(NewStringProperty(typ))
	e.AddProperty(NewStringProperty(subtype))
	e.AddProperty(NewStringProperty(flag))
	for _, v := range values {
		e.AddProperty(v)
	}
	return e
}

func pNumber(name string, v float64) *Element {
	return pEntry(name, "Number", "", "A", NewFloat64Property(v))
}

func pInt(name string, v int32) *Element {
	return pEntry(name, "int", "Integer", "", NewInt32Property(v))
}

func pDouble(name string, v float64) *Element {
	return pEntry(name, "double", "Number", "", NewFloat64Property(v))
}

func pEnum(name string, v int32) *Element {
	return pEntry(name, "enum", "", "", NewInt32Property(v))
}

func pKTime(name string, v int64) *Element {
	return pEntry(name, "KTime", "Time", "", NewInt64Property(v))
}

func pStringEntry(name, v string) *Element {
	return pEntry(name, "KString", "", "", NewStringProperty(v))
}

func pColor(name string, c Color) *Element {
	return pEntry(name, "Color", "", "A",
		NewFloat64Property(float64(c.R)), NewFloat64Property(float64(c.G)), NewFloat64Property(float64(c.B)))
}

func pLclVec3(name string, x, y, z float64) *Element {
	return pEntry(name, name, "", "A",
		NewFloat64Property(x), NewFloat64Property(y), NewFloat64Property(z))
}

// ---------- save ----------

// Prepare finalizes the document in memory (global settings, definitions,
// takes, material properties). Save calls it automatically; calling it by
// hand is useful before custom serialization such as EncodeText.
func (d *Document) Prepare() {
	d.syncGlobalSettings()
	d.syncDefinitions()
	d.syncTakes()
	for _, mat := range d.materials {
		d.syncMaterial(mat)
	}
}

// Save finalizes the document (global settings, definitions, material
// properties) and writes it out.
func (d *Document) Save(w io.Writer) error {
	d.Prepare()
	return Encode(w, d.root, d.Version)
}

// Bytes finalizes the document and returns the encoded file.
func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.Save(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// syncGlobalSettings (re)writes the GlobalSettings Properties70.
func (d *Document) syncGlobalSettings() {
	s := d.Settings
	if !d.rawSettings && s.UnitScaleFactor == 0 {
		// FBX standard defaults: Y up, Z front, X right, centimeter units.
		// Axis values use the file encoding 0=X, 1=Y, 2=Z.
		s.UpAxis = UpVector(1)
		s.UpAxisSign = 1
		s.FrontAxis = FrontVector(2)
		s.FrontAxisSign = 1
		s.CoordAxis = CoordSystem(0)
		s.CoordAxisSign = 1
		s.OriginalUpAxis = -1
		s.OriginalUpAxisSign = 1
		s.UnitScaleFactor = 1
		s.OriginalUnitScaleFactor = 1
		s.TimeMode = FrameRate30
		s.CustomFrameRate = -1
	}

	d.globals.RemoveChild("Properties70")
	p70 := d.globals.AddChildElement("Properties70")
	p70.AddChild(pInt("UpAxis", int32(s.UpAxis)))
	p70.AddChild(pInt("UpAxisSign", int32(s.UpAxisSign)))
	p70.AddChild(pInt("FrontAxis", int32(s.FrontAxis)))
	p70.AddChild(pInt("FrontAxisSign", int32(s.FrontAxisSign)))
	p70.AddChild(pInt("CoordAxis", int32(s.CoordAxis)))
	p70.AddChild(pInt("CoordAxisSign", int32(s.CoordAxisSign)))
	p70.AddChild(pInt("OriginalUpAxis", int32(s.OriginalUpAxis)))
	p70.AddChild(pInt("OriginalUpAxisSign", int32(s.OriginalUpAxisSign)))
	p70.AddChild(pDouble("UnitScaleFactor", float64(s.UnitScaleFactor)))
	p70.AddChild(pDouble("OriginalUnitScaleFactor", float64(s.OriginalUnitScaleFactor)))
	p70.AddChild(pStringEntry("DefaultCamera", "Producer Perspective"))
	p70.AddChild(pKTime("TimeSpanStart", int64(s.TimeSpanStart)))
	p70.AddChild(pKTime("TimeSpanStop", int64(s.TimeSpanStop)))
	p70.AddChild(pEnum("TimeMode", int32(s.TimeMode)))
	p70.AddChild(pDouble("CustomFrameRate", float64(s.CustomFrameRate)))
}

// syncDefinitions rewrites the Definitions section from the recorded object
// counts.
func (d *Document) syncDefinitions() {
	d.defs.RemoveChild("Count")
	total := 1 // GlobalSettings
	for _, n := range d.defCounts {
		total += n
	}
	d.defs.AddChildElement("Count", NewInt32Property(int32(total)))

	// order object types the way exporters usually emit them
	order := []string{"Model", "Geometry", "Material", "Texture", "NodeAttribute",
		"Deformer", "Pose", "AnimationStack", "AnimationLayer", "AnimationCurveNode", "AnimationCurve"}
	counted := map[string]bool{}
	emit := func(name string, count int) {
		counted[name] = true
		ot := d.defs.AddChildElement("ObjectType", NewStringProperty(name))
		ot.AddChildElement("Count", NewInt32Property(int32(count)))
		if tmpl := objectTypeTemplate(name); tmpl != nil {
			ot.AddChild(tmpl)
		}
	}
	emit("GlobalSettings", 1)
	for _, name := range order {
		if n, ok := d.defCounts[name]; ok {
			emit(name, n)
		}
	}
	for name := range d.defCounts { // any type outside the canonical order
		if !counted[name] {
			emit(name, d.defCounts[name])
		}
	}
}

// objectTypeTemplate returns a small PropertyTemplate for well known object
// types. Importers tolerate missing templates; these carry the entries most
// importers expect to find.
func objectTypeTemplate(name string) *Element {
	p70 := NewElement("Properties70")
	var tmplName string
	switch name {
	case "Model":
		tmplName = "FbxNode"
		p70.AddChild(pLclVec3("Lcl Translation", 0, 0, 0))
		p70.AddChild(pLclVec3("Lcl Rotation", 0, 0, 0))
		p70.AddChild(pLclVec3("Lcl Scaling", 1, 1, 1))
	case "Geometry":
		tmplName = "FbxMesh"
		p70.AddChild(pColor("Color", Color{}))
		p70.AddChild(pEntry("BBoxMin", "Vector3D", "Vector", "", NewFloat64Property(0), NewFloat64Property(0), NewFloat64Property(0)))
		p70.AddChild(pEntry("BBoxMax", "Vector3D", "Vector", "", NewFloat64Property(0), NewFloat64Property(0), NewFloat64Property(0)))
		p70.AddChild(pEntry("Primary Visibility", "Visibility", "", "", NewBoolProperty(false), NewInt32Property(0)))
	case "Material":
		tmplName = "FbxSurfacePhong"
		p70.AddChild(pColor("EmissiveColor", Color{}))
		p70.AddChild(pColor("DiffuseColor", Color{1, 1, 1}))
		p70.AddChild(pColor("SpecularColor", Color{}))
		p70.AddChild(pNumber("ShininessExponent", 2))
	case "Texture":
		tmplName = "FbxFileTexture"
		p70.AddChild(pStringEntry("TextureTypeUse", "Standard"))
		p70.AddChild(pEntry("AlphaSource", "enum", "", "", NewInt32Property(1)))
		p70.AddChild(pEntry("CurrentTextureBlendMode", "enum", "", "", NewInt32Property(1)))
	case "NodeAttribute":
		tmplName = "FbxSkeleton"
		p70.AddChild(pEntry("Size", "double", "Number", "", NewFloat64Property(1)))
		p70.AddChild(pEntry("SkeletonType", "enum", "", "", NewInt32Property(0)))
	case "AnimationStack":
		tmplName = "FbxAnimStack"
		p70.AddChild(pKTime("LocalStart", 0))
		p70.AddChild(pKTime("LocalStop", 0))
		p70.AddChild(pKTime("ReferenceStart", 0))
		p70.AddChild(pKTime("ReferenceStop", 0))
		p70.AddChild(pStringEntry("Description", ""))
	case "AnimationLayer":
		tmplName = "FbxAnimLayer"
		p70.AddChild(pNumber("Weight", 100))
		p70.AddChild(pEntry("Mute", "bool", "", "", NewInt32Property(0)))
		p70.AddChild(pEntry("BlendMode", "enum", "", "", NewInt32Property(0)))
	case "AnimationCurveNode":
		tmplName = "FbxAnimCurveNode"
		p70.AddChild(pNumber("d|X", 1))
		p70.AddChild(pNumber("d|Y", 1))
		p70.AddChild(pNumber("d|Z", 1))
	default:
		return nil
	}
	tmpl := NewElement("PropertyTemplate")
	tmpl.AddProperty(NewStringProperty(tmplName))
	tmpl.AddChild(p70)
	return tmpl
}
