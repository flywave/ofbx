package ofbx

// Raw retention: while parsing, a Geometry keeps the source arrays and
// mapping modes of its layers. The processed Vertices/Faces/UVs/... fields
// are convenient for consumers but lossy (layer data is splatted per polygon
// vertex, control points are separated from triangles); the retained raw
// data allows the writer to re-emit layers exactly as they were read.

// layerKind identifies what a geometry layer stores.
type layerKind int

const (
	layerNormal layerKind = iota
	layerTangent
	layerUV
	layerColor
	layerMaterial
)

// rawLayer retains one geometry layer exactly as it appeared in the source
// file: its mapping/reference modes plus the original direct and index
// arrays.
type rawLayer struct {
	kind      layerKind
	set       int // layer set index (UV set number, 0 otherwise)
	mapping   string
	reference string
	// dataF holds the layer's direct float array (flattened), dataI the
	// direct int array (material indices). Exactly one is set.
	dataF []float64
	dataI []int64
	// indices holds the IndexToDirect index array, nil for Direct references.
	indices []int64
}

// rawGeometry retains a geometry's source arrays for lossless write-back.
type rawGeometry struct {
	vertices []float64 // control point positions, flattened xyz
	indices  []int     // PolygonVertexIndex exactly as read (negative-encoded)
	layers   []rawLayer
}

// RawVertices returns the retained control point positions, flattened as
// xyz triples. The slice aliases retained data; callers must not modify it.
func (g *Geometry) RawVertices() []float64 {
	return g.raw.vertices
}

// RawIndices returns the retained PolygonVertexIndex array (negative-encoded
// polygon ends) exactly as read from the file. The slice aliases retained
// data; callers must not modify it.
func (g *Geometry) RawIndices() []int {
	return g.raw.indices
}

// RawLayers returns the retained layer descriptors in file order. The slice
// aliases retained data; callers must not modify it.
func (g *Geometry) RawLayers() []rawLayer {
	return g.raw.layers
}

func flattenPoint3s(pts []Point3) []float64 {
	out := make([]float64, 0, 3*len(pts))
	for _, p := range pts {
		out = append(out, p.X(), p.Y(), p.Z())
	}
	return out
}
