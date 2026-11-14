package ofbx

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realWorldSamples are files produced by other exporters (Autodesk FBX SDK
// variants, assimp's test corpus). Each must load, round trip through
// SaveScene, and reload to an equivalent scene.
var realWorldSamples = []struct {
	path     string
	meshes   int
	anims    int
	embedded int // minimal embedded content size, 0 for none
}{
	{"testdata/cube.fbx", 1, 0, 0},
	{"testdata/FBXcs.fbx", 2, 1, 0},
	{"testdata/FBXcs2.fbx", 2, 1, 0},
	{"testdata/jyj.FBX", 51, 0, 0},
	{"testdata/anim_skeleton.fbx", 1, 1, 0},
	{"testdata/global_settings.fbx", 2, 0, 0},
	{"testdata/huesitos.fbx", 1, 1, 0},
	{"testdata/spider.fbx", 19, 0, 0},
	{"testdata/cubes_with_mirroring_and_pivot.fbx", 4, 1, 0},
	{"testdata/box_ascii_real.fbx", 1, 1, 400000},
	{"testdata/box_orphant_embedded_texture.fbx", 1, 1, 9000},
}

// TestRealWorldSamplesLoad parses every sample and checks headline counts.
func TestRealWorldSamplesLoad(t *testing.T) {
	for _, s := range realWorldSamples {
		t.Run(s.path, func(t *testing.T) {
			scene := loadScene(t, s.path)
			require.Len(t, scene.Meshes, s.meshes)
			assert.Len(t, scene.AnimationStacks, s.anims)
			if s.embedded > 0 {
				found := 0
				for _, v := range scene.Videos {
					if len(v.Content) > found {
						found = len(v.Content)
					}
				}
				assert.GreaterOrEqual(t, found, s.embedded, "embedded media size")
			}
		})
	}
}

// TestRealWorldSamplesRoundTrip exports every sample through SaveScene and
// reloads it.
func TestRealWorldSamplesRoundTrip(t *testing.T) {
	for _, s := range realWorldSamples {
		t.Run(s.path, func(t *testing.T) {
			scene := loadScene(t, s.path)
			reloaded := saveAndReload(t, scene)
			compareScenes(t, scene, reloaded)
		})
	}
}

// TestAutodeskAsciiDialect pins the text dialect quirks found in real
// Autodesk exports.
func TestAutodeskAsciiDialect(t *testing.T) {
	// empty placeholder property + base64 on the next line + unescaped
	// backslash paths + trailing blank lines
	src := `; FBX 7.4.0 project file
; ----------

Objects:  {
	Video: 123, "v\x00\x01Video", "Clip" {
	Type: "Clip"
	Filename: "C:\Users\x\Desktop\uv.png"
	Content: ,
"iVBORw0KGgoAAAANSUhEUg=="
	}
}

`
	scene, err := Load(bytes.NewReader([]byte(src)))
	require.NoError(t, err)
	require.Len(t, scene.Videos, 1)
	v := scene.Videos[0]
	assert.Equal(t, `C:\Users\x\Desktop\uv.png`, v.Filename)
	require.NotNil(t, v.Content)
	// base64 of the PNG header bytes
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0, 0, 0, 0x0d, 0x49, 0x48, 0x44, 0x52}, v.Content)
}

// TestAssimpAcceptsOurOutput is an external validation gate: when the
// assimp CLI is installed, every file this library writes must import
// cleanly in a third-party FBX importer.
func TestAssimpAcceptsOurOutput(t *testing.T) {
	assimp, err := exec.LookPath("assimp")
	if err != nil {
		t.Skip("assimp CLI not installed; skipping external validation")
	}

	var outputs []struct {
		name string
		data []byte
	}

	// from-scratch document with every feature
	doc := NewDocument(WithVersion(FBXVersion7400))
	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	require.NoError(t, err)
	corner := cornerCount(cubeFaces)
	normals := make([]Point3, 0, corner)
	uvs := make([]Point2, 0, corner)
	for _, f := range cubeFaces {
		for i := range f {
			normals = append(normals, Point3{0, 1, 0})
			uvs = append(uvs, Point2{float64(i % 2), 0})
		}
	}
	require.NoError(t, doc.SetNormals(mesh, normals))
	require.NoError(t, doc.SetUVs(mesh, 0, uvs))
	mat := doc.NewMaterial("M")
	doc.AddMaterial(mesh, mat)
	bone, err := doc.NewNode("Bone", "LimbNode")
	require.NoError(t, err)
	doc.SetParent(bone, nil)
	skin, err := doc.NewSkin(mesh)
	require.NoError(t, err)
	idx := make([]int, len(cubeVertices))
	weights := make([]float64, len(cubeVertices))
	for i := range idx {
		idx[i] = i
		weights[i] = 1
	}
	_, err = doc.NewCluster(skin, bone, idx, weights, IdentityMatrix())
	require.NoError(t, err)
	stack := doc.NewAnimationStack("Take001", 0, stdTimeToFbxTime(1e9))
	layer := doc.NewAnimationLayer("L", stack)
	curve, err := doc.NewAnimationCurve("c", []time.Duration{0, 1e9}, []float32{0, 2})
	require.NoError(t, err)
	_, err = doc.NewAnimationCurveNode("T", layer, mesh, "Lcl Translation", [3]*AnimationCurve{curve, nil, nil})
	require.NoError(t, err)
	cam, err := doc.NewCamera("Shot", 45, 50)
	require.NoError(t, err)
	doc.SetParent(cam, nil)
	light, err := doc.NewLight("Sun", "Directional", Color{1, 1, 1}, 1)
	require.NoError(t, err)
	doc.SetParent(light, nil)

	bin, err := doc.Bytes()
	require.NoError(t, err)
	outputs = append(outputs, struct {
		name string
		data []byte
	}{"scratch-7400.fbx", bin})

	// real files: re-encode and export, binary and ASCII
	for _, path := range []string{
		"testdata/cube.fbx",
		"testdata/anim_skeleton.fbx",
		"testdata/huesitos.fbx",
		"testdata/jyj.FBX",
		"testdata/box_ascii_real.fbx",
	} {
		scene := loadScene(t, path)
		var buf bytes.Buffer
		require.NoError(t, Encode(&buf, scene.RootElement, 7400))
		outputs = append(outputs, struct {
			name string
			data []byte
		}{path + " [reencode]", buf.Bytes()})

		var exp bytes.Buffer
		require.NoError(t, SaveScene(scene, &exp))
		outputs = append(outputs, struct {
			name string
			data []byte
		}{path + " [export]", exp.Bytes()})

		var txt bytes.Buffer
		require.NoError(t, EncodeText(&txt, scene.RootElement, 7400))
		outputs = append(outputs, struct {
			name string
			data []byte
		}{path + " [ascii]", txt.Bytes()})
	}

	for _, out := range outputs {
		out := out
		t.Run(out.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := tmp + "/out.fbx"
			require.NoError(t, os.WriteFile(path, out.data, 0o644))
			cmd := exec.Command(assimp, "info", path)
			cmd.Stdout = nil
			cmd.Stderr = nil
			err := cmd.Run()
			require.NoError(t, err, "assimp rejected our output")
		})
	}
}
