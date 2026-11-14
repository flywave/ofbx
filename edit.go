package ofbx

import (
	"errors"
	"fmt"
	"strings"
)

// Edit-mode helpers: mutate objects of a loaded Scene in place (their element
// trees are modified) so that a subsequent SaveScene writes the changes while
// every untouched part of the file survives verbatim.

// SyncMaterial rewrites a material's Properties70 from its typed fields so
// that modifications made to a loaded material survive SaveScene. Document
// materials are synced automatically at Save time; parsed materials need this
// call after their fields are changed.
func SyncMaterial(mat *Material) {
	if mat == nil || mat.Element() == nil {
		return
	}
	el := mat.Element()
	el.RemoveChild("Properties70")
	el.AddChild(materialProperties70(mat))
}

// Rename changes an object's name, preserving the "\x00\x01<Type>" suffix
// FBX stores in the name property.
func Rename(o Obj, name string) error {
	if o == nil || o.Element() == nil {
		return errors.New("ofbx: object has no element")
	}
	p := o.Element().getProperty(1)
	if p == nil {
		return errors.New("ofbx: object element has no name property")
	}
	if i := strings.Index(p.value.String(), "\x00\x01"); i >= 0 {
		name += p.value.String()[i:]
	}
	o.Element().Properties[1] = NewStringProperty(name)
	return nil
}

// SetTextureFilename rewrites a texture's absolute and relative file names.
func SetTextureFilename(tex *Texture, filename, relative string) {
	if tex == nil || tex.Element() == nil {
		return
	}
	el := tex.Element()
	if filename != "" {
		fn := NewElement("FileName")
		fn.AddProperty(NewStringProperty(filename))
		el.ReplaceChild("FileName", fn)
	}
	if relative != "" {
		rf := NewElement("RelativeFilename")
		rf.AddProperty(NewStringProperty(relative))
		el.ReplaceChild("RelativeFilename", rf)
	}
}

// SetLclVec3 writes a three component Properties70 entry (adding or
// replacing) on the object's element, e.g. "Lcl Translation".
func SetLclVec3(o Obj, name string, v Point3) error {
	return setPValues(o, name, 3,
		NewFloat64Property(v.X()), NewFloat64Property(v.Y()), NewFloat64Property(v.Z()))
}

// SetLclDouble writes a single component Properties70 entry (adding or
// replacing) on the object's element.
func SetLclDouble(o Obj, name string, v float64) error {
	return setPValues(o, name, 1, NewFloat64Property(v))
}

// SetTranslation sets the object's "Lcl Translation".
func SetTranslation(o Obj, v Point3) error {
	return SetLclVec3(o, "Lcl Translation", v)
}

// SetRotation sets the object's "Lcl Rotation" (degrees).
func SetRotation(o Obj, deg Point3) error {
	return SetLclVec3(o, "Lcl Rotation", deg)
}

// SetScaling sets the object's "Lcl Scaling".
func SetScaling(o Obj, s Point3) error {
	return SetLclVec3(o, "Lcl Scaling", s)
}

// setPValues adds or replaces the "P" entry with the given name inside the
// object element's Properties70. When replacing, the entry's type descriptor
// properties are kept and only the value properties are swapped.
func setPValues(o Obj, name string, wantValues int, values ...*Property) error {
	if o == nil || o.Element() == nil {
		return errors.New("ofbx: object has no element")
	}
	if len(values) != wantValues {
		return fmt.Errorf("ofbx: %d values expected, got %d", wantValues, len(values))
	}

	el := o.Element()
	p70 := el.Child("Properties70")
	if p70 == nil {
		p70 = el.AddChildElement("Properties70")
	}

	for _, p := range p70.Children {
		if len(p.Properties) == 0 || p.Properties[0].value.String() != name {
			continue
		}
		// found the entry: keep descriptor props (name/type/flags), replace values
		if len(p.Properties) > 4 {
			p.Properties = p.Properties[:4]
		}
		p.Properties = append(p.Properties, values...)
		return nil
	}

	// not present yet: append a fresh entry
	entry := pEntry(name, name, "", "A", values...)
	p70.AddChild(entry)
	return nil
}

// RebuildGeometryElement rebuilds a mesh's Geometry element in place from its
// current processed data, so a following SaveScene (or Document.Save) writes
// the changes. Vertices (control points) and Faces are written as-is; layers
// are re-emitted from the retained raw arrays when present, preserving the
// original mapping/reference modes, or kept from the existing element when
// they were added through the Document setters.
//
// The mesh's Geometric* properties are stripped in the same pass: on the next
// load the geometric transform would otherwise bake into the already-baked
// vertices a second time.
func RebuildGeometryElement(m *Mesh) error {
	if m == nil || m.Geometry == nil {
		return errors.New("ofbx: mesh has no geometry")
	}
	g := m.Geometry
	el := g.Element()
	if el == nil {
		return errors.New("ofbx: geometry has no element")
	}
	if len(g.Vertices) == 0 || len(g.Faces) == 0 {
		return errors.New("ofbx: cannot rebuild empty geometry")
	}

	// remember setter-built layers before clearing (used when there is no
	// raw retention, e.g. for Document-created meshes)
	var keepLayers []*Element
	if len(g.raw.layers) == 0 {
		for _, c := range el.Children {
			if isLayerChild(c) {
				keepLayers = append(keepLayers, c)
			}
		}
	}

	flat := flattenPoint3s(g.Vertices)
	vertsProp, err := NewDoubleArrayProperty(flat, false)
	if err != nil {
		return err
	}
	indices := make([]int32, 0)
	for _, f := range g.Faces {
		for i, idx := range f {
			if i == len(f)-1 {
				indices = append(indices, int32(-idx-1))
			} else {
				indices = append(indices, int32(idx))
			}
		}
	}
	idxProp, err := NewInt32ArrayProperty(indices, false)
	if err != nil {
		return err
	}

	// rebuild the element in place: its identity stays the same everywhere
	// (parsed scene trees and Documents both reference this node)
	el.Children = el.Children[:0]
	el.AddChildElement("Vertices", vertsProp)
	el.AddChildElement("PolygonVertexIndex", idxProp)
	el.AddChildElement("GeometryVersion", NewInt32Property(124))

	switch {
	case len(g.raw.layers) > 0:
		if err := appendRawLayers(el, g.raw.layers); err != nil {
			return err
		}
	case len(keepLayers) > 0:
		// geometries built through Document setters carry their layers in
		// the element already; keep them unchanged
		el.Children = append(el.Children, keepLayers...)
	default:
		if err := appendProcessedLayers(el, g); err != nil {
			return err
		}
	}

	// strip Geometric* props from the model: the transform is already baked
	// refresh the raw retention from the rebuilt element so later reads and
	// comparisons see the new source of truth
	g.raw.vertices = flat
	g.raw.indices = make([]int, 0, len(indices))
	for _, idx := range indices {
		g.raw.indices = append(g.raw.indices, int(idx))
	}

	stripProperties(m, "GeometricTranslation", "GeometricRotation", "GeometricScaling")
	return nil
}

// isLayerChild reports whether an element is a geometry layer node.
func isLayerChild(c *Element) bool {
	if c == nil || c.ID == nil {
		return false
	}
	id := c.ID.String()
	return strings.HasPrefix(id, "LayerElement") || id == "Layer"
}

// appendRawLayers re-emits retained layers with their original mapping and
// reference modes.
func appendRawLayers(el *Element, layers []rawLayer) error {
	layerIdx := map[string]int{}
	for _, l := range layers {
		var layerEl *Element
		var dataName, idxName, entryType string
		switch l.kind {
		case layerNormal:
			entryType, dataName, idxName = "LayerElementNormal", "Normals", "NormalsIndex"
			layerEl = el.AddChildElement(entryType, NewInt64Property(0))
		case layerTangent:
			entryType, dataName, idxName = "LayerElementTangents", "Tangents", "TangentsIndex"
			layerEl = el.AddChildElement(entryType, NewInt64Property(0))
		case layerUV:
			entryType, dataName, idxName = "LayerElementUV", "UV", "UVIndex"
			layerEl = el.AddChildElement(entryType, NewInt64Property(int64(l.set)))
		case layerColor:
			entryType, dataName, idxName = "LayerElementColor", "Colors", "ColorIndex"
			layerEl = el.AddChildElement(entryType, NewInt64Property(0))
		case layerMaterial:
			entryType = "LayerElementMaterial"
			layerEl = el.AddChildElement(entryType, NewInt64Property(0))
		default:
			continue
		}

		layerEl.AddChildElement("Version", NewInt32Property(101))
		layerEl.AddChildElement("Name", NewStringProperty(""))
		layerEl.AddChildElement("MappingInformationType", NewStringProperty(orDefault(l.mapping, "ByPolygonVertex")))
		layerEl.AddChildElement("ReferenceInformationType", NewStringProperty(orDefault(l.reference, "Direct")))

		if l.kind == layerMaterial {
			ints := make([]int32, len(l.dataI))
			for i, v := range l.dataI {
				ints[i] = int32(v)
			}
			matProp, err := NewInt32ArrayProperty(ints, false)
			if err != nil {
				return err
			}
			layerEl.AddChildElement("Materials", matProp)
		} else {
			dataProp, err := NewDoubleArrayProperty(l.dataF, false)
			if err != nil {
				return err
			}
			layerEl.AddChildElement(dataName, dataProp)
			if len(l.indices) > 0 {
				idxs := make([]int32, len(l.indices))
				for i, v := range l.indices {
					idxs[i] = int32(v)
				}
				idxProp, err := NewInt32ArrayProperty(idxs, false)
				if err != nil {
					return err
				}
				layerEl.AddChildElement(idxName, idxProp)
			}
		}

		layerIdx[entryType]++
		addLayerEntryStatic(el, entryType)
	}
	return nil
}

// addLayerEntryStatic records a typed layer in the geometry's Layer: 0 node,
// creating that node on first use (the rebuild-path twin of
// Document.addLayerEntry).
func addLayerEntryStatic(geomEl *Element, typ string) {
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

// appendProcessedLayers writes the processed per-corner layer data as
// ByPolygonVertex/Direct layers.
func appendProcessedLayers(el *Element, g *Geometry) error {
	cornerCount := 0
	for _, f := range g.Faces {
		cornerCount += len(f)
	}
	if len(g.Normals) == cornerCount && cornerCount > 0 {
		data, err := NewDoubleArrayProperty(flattenPoint3s(g.Normals), false)
		if err != nil {
			return err
		}
		layerEl := el.AddChildElement("LayerElementNormal", NewInt64Property(0))
		layerEl.AddChildElement("Version", NewInt32Property(101))
		layerEl.AddChildElement("Name", NewStringProperty(""))
		layerEl.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
		layerEl.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
		layerEl.AddChildElement("Normals", data)
		addLayerEntryStatic(el, "LayerElementNormal")
	}
	if len(g.UVs[0]) == cornerCount && cornerCount > 0 {
		flat := make([]float64, 0, 2*len(g.UVs[0]))
		for _, uv := range g.UVs[0] {
			flat = append(flat, uv.X(), uv.Y())
		}
		data, err := NewDoubleArrayProperty(flat, false)
		if err != nil {
			return err
		}
		layerEl := el.AddChildElement("LayerElementUV", NewInt64Property(0))
		layerEl.AddChildElement("Version", NewInt32Property(101))
		layerEl.AddChildElement("Name", NewStringProperty("UVMap"))
		layerEl.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
		layerEl.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
		layerEl.AddChildElement("UV", data)
		addLayerEntryStatic(el, "LayerElementUV")
	}
	return nil
}

// stripProperties removes the named Properties70 entries from an object.
func stripProperties(o Obj, names ...string) {
	el := o.Element()
	if el == nil {
		return
	}
	p70 := el.Child("Properties70")
	if p70 == nil {
		return
	}
	remove := map[string]bool{}
	for _, n := range names {
		remove[n] = true
	}
	kept := p70.Children[:0]
	for _, p := range p70.Children {
		if len(p.Properties) > 0 && remove[p.Properties[0].value.String()] {
			continue
		}
		kept = append(kept, p)
	}
	p70.Children = kept
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
