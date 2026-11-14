package ofbx

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedMedia creates an embedded texture and reads it back.
func TestEmbeddedMedia(t *testing.T) {
	content := make([]byte, 4096)
	_, err := rand.Read(content)
	require.NoError(t, err)

	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	mat := doc.NewMaterial("Mat")

	tex, err := doc.NewEmbeddedTexture("SkinTex", "png", content, "/abs/skin.png", "skin.png")
	require.NoError(t, err)
	require.NoError(t, doc.SetTexture(mat, DIFFUSE, tex))
	doc.AddMaterial(mesh, mat)

	scene := reloadDoc(t, doc)
	require.Len(t, scene.Videos, 1)
	v := scene.Videos[0]
	assert.Equal(t, "/abs/skin.png", v.Filename)
	assert.Equal(t, "skin.png", v.RelativeFilename)
	require.NotNil(t, v.Content, "embedded content must round trip")
	assert.Equal(t, content, v.Content)

	// the texture references the video through Media
	require.Len(t, scene.Meshes[0].Materials, 1)
	rt := scene.Meshes[0].Materials[0].Textures[DIFFUSE]
	require.NotNil(t, rt)
	require.NotNil(t, rt.Media, "texture must link its video")
	assert.Equal(t, v.ID(), rt.Media.ID())
	assert.Equal(t, content, rt.Media.Content)
}

// TestExternalVideoLinks checks that file-referenced videos (as in
// FBXcs.fbx) parse and attach to their textures.
func TestExternalVideoLinks(t *testing.T) {
	scene := loadScene(t, "testdata/FBXcs.fbx")
	require.NotEmpty(t, scene.Videos)
	v := scene.Videos[0]
	assert.NotEmpty(t, v.Filename)
	assert.Nil(t, v.Content, "FBXcs videos are file references")

	// at least one mesh material texture is video-backed
	linked := 0
	for _, m := range scene.Meshes {
		for _, mat := range m.Materials {
			for i := DIFFUSE; i < TextureCOUNT; i++ {
				if mat.Textures[i] != nil && mat.Textures[i].Media != nil {
					linked++
				}
			}
		}
	}
	assert.NotZero(t, linked, "expected video-backed textures")
}

// TestBlendShapeRoundTrip builds a morph target and reads it back.
func TestBlendShapeRoundTrip(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)

	bs, err := doc.NewBlendShape(mesh, "Morph")
	require.NoError(t, err)
	ch := doc.NewBlendShapeChannel(bs, "Smile", 75.0)
	_, err = doc.AddShape(ch, "SmileShape",
		[]int{0, 1, 2}, []Point3{{1, 0, 0}, {0, 2, 0}, {0, 0, 3}})
	require.NoError(t, err)

	// mismatched inputs must be rejected
	_, err = doc.AddShape(ch, "Bad", []int{0}, []Point3{{1, 0, 0}, {0, 1, 0}})
	assert.Error(t, err)

	scene := reloadDoc(t, doc)
	require.Len(t, scene.Meshes, 1)
	g := scene.Meshes[0].Geometry
	require.Len(t, g.BlendShapes, 1)
	rbs := g.BlendShapes[0]
	require.Len(t, rbs.Channels, 1)
	rch := rbs.Channels[0]
	assert.InDelta(t, 75.0, rch.Weight, 1e-9)
	require.Len(t, rch.Shapes, 1)
	sh := rch.Shapes[0]
	assert.Equal(t, []int{0, 1, 2}, sh.Indices)
	assert.Equal(t, []Point3{{1, 0, 0}, {0, 2, 0}, {0, 0, 3}}, sh.Vertices)
}

// TestCameraLightRoundTrip builds a camera and a light and reads them back.
func TestCameraLightRoundTrip(t *testing.T) {
	doc := NewDocument()
	cam, err := doc.NewCamera("Shot", 45.0, 50.0)
	require.NoError(t, err)
	doc.SetParent(cam, nil)

	light, err := doc.NewLight("Sun", "Directional", Color{1, 0.95, 0.9}, 2.5)
	require.NoError(t, err)
	doc.SetParent(light, nil)

	_, err = doc.NewLight("Bad", "Disco", Color{}, 1)
	assert.Error(t, err)

	scene := reloadDoc(t, doc)
	require.Len(t, scene.Cameras, 1)
	rc := scene.Cameras[0]
	assert.Contains(t, rc.Name(), "Shot")
	require.NotNil(t, rc.NodeAttribute())
	attrs, ok := rc.NodeAttribute().(*CameraAttributes)
	require.True(t, ok)
	assert.InDelta(t, 45.0, attrs.FieldOfView, 1e-9)
	assert.InDelta(t, 50.0, attrs.FocalLength, 1e-9)

	require.Len(t, scene.Lights, 1)
	rl := scene.Lights[0]
	assert.Contains(t, rl.Name(), "Sun")
	require.NotNil(t, rl.NodeAttribute())
	lattrs, ok := rl.NodeAttribute().(*LightAttributes)
	require.True(t, ok)
	assert.Equal(t, "Directional", lattrs.LightType)
	assert.Equal(t, Color{1, 0.95, 0.9}, lattrs.Color)
	assert.InDelta(t, 2.5, lattrs.Intensity, 1e-9)
}

// TestRealFileCameraLights parses the camera/light nodes in the animated
// skeleton sample.
func TestRealFileCameraLights(t *testing.T) {
	scene := loadScene(t, "testdata/anim_skeleton.fbx")
	require.Len(t, scene.Cameras, 1)
	require.Len(t, scene.Lights, 1)

	cam := scene.Cameras[0]
	require.NotNil(t, cam.NodeAttribute())
	_, ok := cam.NodeAttribute().(*CameraAttributes)
	require.True(t, ok, "camera attribute must be typed")

	light := scene.Lights[0]
	require.NotNil(t, light.NodeAttribute())
	_, ok = light.NodeAttribute().(*LightAttributes)
	require.True(t, ok, "light attribute must be typed")

	// and they still round trip
	reloaded := saveAndReload(t, scene)
	require.Len(t, reloaded.Cameras, 1)
	require.Len(t, reloaded.Lights, 1)
}

// TestPublicPropertyAccessors covers the value readers added for external
// element inspection.
func TestPublicPropertyAccessors(t *testing.T) {
	root := NewElement("")
	el := root.AddChildElement("E",
		NewStringProperty("s"), NewInt64Property(-42), NewFloat64Property(2.5))
	arr, err := NewInt32ArrayProperty([]int32{1, 2, 3}, false)
	require.NoError(t, err)
	el.AddChildElement("Arr", arr)

	props := el.Properties
	assert.Equal(t, STRING, props[0].GetType())
	assert.Equal(t, "s", props[0].ValueString())
	assert.Equal(t, int64(-42), props[1].ValueI64())
	assert.Equal(t, uint64(0xFFFFFFFFFFFFFFD6), props[1].ValueU64())
	assert.Equal(t, 2.5, props[2].ValueF64())

	ap := el.Child("Arr").Properties[0]
	assert.Equal(t, 3, ap.GetCount())
	vals, err := ap.ArrayI64()
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3}, vals)

	// binary content access via ValueBytes
	raw := NewRawProperty([]byte{1, 2, 3})
	assert.Equal(t, []byte{1, 2, 3}, raw.ValueBytes())
	_ = bytes.MinRead
}
