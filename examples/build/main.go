// Command build creates a textured, animated cube from scratch and writes it
// as binary and ASCII FBX.
package main

import (
	"log"
	"os"
	"time"

	ofbx "github.com/flywave/ofbx"
)

var cubeVertices = []ofbx.Point3{
	{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
	{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
}

var cubeFaces = [][]int{
	{0, 1, 2, 3}, {4, 5, 6, 7}, {0, 1, 5, 4},
	{2, 3, 7, 6}, {0, 3, 7, 4}, {1, 2, 6, 5},
}

func main() {
	doc := ofbx.NewDocument(ofbx.WithVersion(ofbx.FBXVersion7400))

	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	if err != nil {
		log.Fatal(err)
	}

	// one flat normal per face, repeated per corner
	normals := make([]ofbx.Point3, 0, 24)
	for _, f := range cubeFaces {
		for range f {
			normals = append(normals, ofbx.Point3{0, 1, 0})
		}
	}
	if err := doc.SetNormals(mesh, normals); err != nil {
		log.Fatal(err)
	}

	uvs := make([]ofbx.Point2, 0, 24)
	for _, f := range cubeFaces {
		for i := range f {
			uvs = append(uvs, ofbx.Point2{float64(i % 2), float64(i / 2)})
		}
	}
	if err := doc.SetUVs(mesh, 0, uvs); err != nil {
		log.Fatal(err)
	}

	mat := doc.NewMaterial("CubeMaterial")
	mat.DiffuseColor = ofbx.Color{R: 0.9, G: 0.3, B: 0.2}
	doc.AddMaterial(mesh, mat)

	tex := doc.NewTexture("CubeTex", "/textures/cube.png", "textures/cube.png")
	if err := doc.SetTexture(mat, ofbx.DIFFUSE, tex); err != nil {
		log.Fatal(err)
	}

	// a bobbing animation on the mesh
	stack := doc.NewAnimationStack("Bob", 0, 46186158000)
	layer := doc.NewAnimationLayer("BaseLayer", stack)
	up, err := doc.NewAnimationCurve("c",
		[]time.Duration{0, 500 * time.Millisecond, time.Second},
		[]float32{0, 2, 0})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := doc.NewAnimationCurveNode("T", layer, mesh, "Lcl Translation",
		[3]*ofbx.AnimationCurve{up, nil, nil}); err != nil {
		log.Fatal(err)
	}

	write := func(name string, encode func(*os.File) error) {
		f, err := os.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		if err := encode(f); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", name)
	}

	write("cube.fbx", func(f *os.File) error { return doc.Save(f) })
	write("cube_ascii.fbx", func(f *os.File) error {
		doc.Prepare()
		return ofbx.EncodeText(f, doc.Root(), 7400)
	})
}
