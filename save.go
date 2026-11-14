package ofbx

import (
	"errors"
	"io"
)

// supportedObjectTypes lists the element names under Objects that the parser
// turns into typed objects.
var supportedObjectTypes = map[string]bool{
	"Geometry":           true,
	"Material":           true,
	"Model":              true,
	"Texture":            true,
	"NodeAttribute":      true,
	"Deformer":           true,
	"AnimationStack":     true,
	"AnimationLayer":     true,
	"AnimationCurve":     true,
	"AnimationCurveNode": true,
}

// SaveScene serializes a parsed Scene into a fresh, clean FBX document: a new
// header, GlobalSettings from the scene, recomputed Definitions and the
// scene's object elements and connections. Object elements are reused
// verbatim, so everything the reader preserved (including object types this
// library does not model) survives; geometry layers are re-emitted from the
// retained raw arrays.
//
// Options behave as in NewDocument; the written version defaults to the
// source file's version (from FBXHeaderExtension) instead of
// DefaultWriteVersion. OnlySupportedObjects drops unmodeled object types.
func SaveScene(scene *Scene, w io.Writer, opts ...DocOption) error {
	if scene == nil || scene.RootElement == nil {
		return errors.New("ofbx: cannot save nil scene")
	}
	cfg := &Document{}
	for _, o := range opts {
		o(cfg)
	}
	version := cfg.Version
	if version == 0 {
		version = sourceVersion(scene)
	}
	if version == 0 {
		version = DefaultWriteVersion
	}

	docOpts := []DocOption{WithVersion(version)}
	if cfg.Creator != "" {
		docOpts = append(docOpts, WithCreator(cfg.Creator))
	}
	if cfg.CompressArrays {
		docOpts = append(docOpts, WithCompressedArrays())
	}
	doc := NewDocument(docOpts...)
	doc.onlySupported = cfg.onlySupported
	doc.Settings = scene.Settings

	objsEl := scene.RootElement.Child("Objects")
	if objsEl == nil {
		return errors.New("ofbx: scene has no Objects section")
	}

	kept := map[uint64]bool{}
	for _, objEl := range objsEl.Children {
		if objEl.ID == nil {
			continue
		}
		typeName := objEl.IDString()
		if doc.onlySupported && !supportedObjectTypes[typeName] {
			continue
		}
		if p := objEl.getProperty(0); p != nil && isLong(p) {
			kept[uint64(propInt(p))] = true
		}
		doc.objs.AddChild(objEl)
		doc.countType(typeName)
	}

	connsEl := scene.RootElement.Child("Connections")
	if connsEl != nil {
		for _, c := range connsEl.Children {
			props := c.Properties
			if len(props) < 3 || !isString(props[0]) || !isLong(props[1]) || !isLong(props[2]) {
				continue
			}
			from := uint64(propInt(props[1]))
			to := uint64(propInt(props[2]))
			// keep connections whose both endpoints are kept; "to" may be the
			// scene root (0)
			if !kept[from] {
				continue
			}
			if to != 0 && !kept[to] {
				continue
			}
			doc.conns.AddChild(c)
		}
	}

	if takesEl := scene.RootElement.Child("Takes"); takesEl != nil {
		doc.root.AddChild(takesEl)
	}

	return doc.Save(w)
}

// sourceVersion extracts the FBX version of the file a scene was loaded from.
func sourceVersion(scene *Scene) uint32 {
	header := scene.RootElement.Child("FBXHeaderExtension")
	if header == nil {
		return 0
	}
	versionEl := header.Child("FBXVersion")
	if versionEl == nil || len(versionEl.Properties) == 0 {
		return 0
	}
	return uint32(versionEl.Properties[0].value.toInt32())
}
