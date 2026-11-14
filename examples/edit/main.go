// Command edit loads an FBX file, modifies a transform, a material and one
// vertex, and re-exports it.
package main

import (
	"log"
	"os"

	ofbx "github.com/flywave/ofbx"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: edit <input.fbx> <output.fbx>")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	scene, err := ofbx.Load(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	for _, mesh := range scene.Meshes {
		// move the mesh up
		if err := ofbx.SetTranslation(mesh, ofbx.Point3{0, 10, 0}); err != nil {
			log.Fatal(err)
		}

		// tint its first material red and sync the change into the file data
		if len(mesh.Materials) > 0 {
			mesh.Materials[0].DiffuseColor = ofbx.Color{R: 1}
			ofbx.SyncMaterial(mesh.Materials[0])
		}

		// nudge one vertex and rebuild the geometry element
		if g := mesh.Geometry; g != nil && len(g.Vertices) > 0 {
			g.Vertices[0] = g.Vertices[0].Add(ofbx.Point3{0.5, 0.5, 0.5})
			if err := ofbx.RebuildGeometryElement(mesh); err != nil {
				log.Fatal(err)
			}
		}
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if err := ofbx.SaveScene(scene, out); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", os.Args[2])
}
