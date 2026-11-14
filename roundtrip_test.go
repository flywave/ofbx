package ofbx

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadAndEncode loads a test file, re-encodes its element tree at the given
// version and parses the result back, returning both scenes.
func loadAndEncode(t *testing.T, path string, version uint32) (*Scene, *Scene) {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var buf bytes.Buffer
	require.NoError(t, Encode(&buf, scene1.RootElement, version))
	require.Greater(t, buf.Len(), 0)

	scene2, err := Load(&buf)
	require.NoError(t, err)
	return scene1, scene2
}

func compareGeometries(t *testing.T, a, b *Geometry) {
	t.Helper()
	assert.Equal(t, a.Vertices, b.Vertices, "vertices")
	assert.Equal(t, a.Faces, b.Faces, "faces")
	assert.Equal(t, a.Normals, b.Normals, "normals")
	assert.Equal(t, a.Tangents, b.Tangents, "tangents")
	assert.Equal(t, a.Colors, b.Colors, "colors")
	assert.Equal(t, a.Materials, b.Materials, "materials")
	for i := range a.UVs {
		assert.Equal(t, a.UVs[i], b.UVs[i], "uv set %d", i)
	}
	// raw retention of the reloaded file must match the first parse too
	assert.Equal(t, a.RawVertices(), b.RawVertices(), "raw vertices")
	assert.Equal(t, a.RawIndices(), b.RawIndices(), "raw indices")
	assert.Equal(t, len(a.RawLayers()), len(b.RawLayers()), "raw layer count")
}

func compareScenes(t *testing.T, a, b *Scene) {
	t.Helper()

	assert.Equal(t, a.Settings, b.Settings, "settings")
	assert.Equal(t, a.FrameRate, b.FrameRate, "frame rate")
	assert.Equal(t, len(a.TakeInfos), len(b.TakeInfos), "take info count")
	assert.Equal(t, len(a.Connections), len(b.Connections), "connection count")
	for i := range a.Connections {
		assert.Equal(t, a.Connections[i].typ, b.Connections[i].typ, "connection %d type", i)
		assert.Equal(t, a.Connections[i].from, b.Connections[i].from, "connection %d from", i)
		assert.Equal(t, a.Connections[i].to, b.Connections[i].to, "connection %d to", i)
		assert.Equal(t, a.Connections[i].property, b.Connections[i].property, "connection %d property", i)
	}

	assert.Equal(t, len(a.Meshes), len(b.Meshes), "mesh count")
	for i := range a.Meshes {
		ma, mb := a.Meshes[i], b.Meshes[i]
		assert.Equal(t, ma.ID(), mb.ID(), "mesh %d id", i)
		assert.Equal(t, ma.Name(), mb.Name(), "mesh %d name", i)
		require.NotNil(t, ma.Geometry, "mesh %d geometry", i)
		require.NotNil(t, mb.Geometry, "mesh %d geometry", i)
		compareGeometries(t, ma.Geometry, mb.Geometry)

		require.Equal(t, len(ma.Materials), len(mb.Materials), "mesh %d material count", i)
		for j := range ma.Materials {
			assert.Equal(t, ma.Materials[j].DiffuseColor, mb.Materials[j].DiffuseColor, "mesh %d material %d diffuse", i, j)
			assert.Equal(t, ma.Materials[j].SpecularColor, mb.Materials[j].SpecularColor, "mesh %d material %d specular", i, j)
			for k := DIFFUSE; k < TextureCOUNT; k++ {
				if ma.Materials[j].Textures[k] == nil {
					assert.Nil(t, mb.Materials[j].Textures[k])
					continue
				}
				require.NotNil(t, mb.Materials[j].Textures[k])
				assert.Equal(t,
					ma.Materials[j].Textures[k].GetFileName(),
					mb.Materials[j].Textures[k].GetFileName(),
					"mesh %d material %d texture filename", i, j)
			}
		}
	}

	assert.Equal(t, len(a.AnimationStacks), len(b.AnimationStacks), "animation stack count")
	if len(a.AnimationStacks) > 0 {
		compareAnimations(t, a, b)
	}

	// newer object kinds
	assert.Equal(t, len(a.Videos), len(b.Videos), "video count")
	for i := range a.Videos {
		assert.Equal(t, a.Videos[i].Filename, b.Videos[i].Filename, "video %d filename", i)
		assert.Equal(t, a.Videos[i].Content, b.Videos[i].Content, "video %d content", i)
	}
	assert.Equal(t, len(a.Cameras), len(b.Cameras), "camera count")
	assert.Equal(t, len(a.Lights), len(b.Lights), "light count")

	totalShapes := func(s *Scene) int {
		n := 0
		for _, m := range s.Meshes {
			if m.Geometry == nil {
				continue
			}
			for _, bs := range m.Geometry.BlendShapes {
				for _, ch := range bs.Channels {
					n += len(ch.Shapes)
				}
			}
		}
		return n
	}
	assert.Equal(t, totalShapes(a), totalShapes(b), "blend shape count")

	skinWeightSum := func(s *Scene) float64 {
		sum := 0.0
		for _, m := range s.Meshes {
			if m.Geometry == nil || m.Geometry.Skin == nil {
				continue
			}
			for _, c := range m.Geometry.Skin.Clusters {
				for _, w := range c.Weights {
					sum += w
				}
			}
		}
		return sum
	}
	assert.InDelta(t, skinWeightSum(a), skinWeightSum(b), 1e-6, "skin weight sum")
}

func compareAnimations(t *testing.T, a, b *Scene) {
	t.Helper()
	for i := range a.AnimationStacks {
		sa, sb := a.AnimationStacks[i], b.AnimationStacks[i]
		assert.Equal(t, sa.ID(), sb.ID(), "stack %d id", i)
		assert.Equal(t, len(sa.Layers), len(sb.Layers), "stack %d layer count", i)
		for j := range sa.Layers {
			la, lb := sa.Layers[j], sb.Layers[j]
			assert.Equal(t, la.ID(), lb.ID(), "layer %d id", j)
			assert.Equal(t, len(la.CurveNodes), len(lb.CurveNodes), "layer %d curve node count", j)
			for k := range la.CurveNodes {
				na, nb := la.CurveNodes[k], lb.CurveNodes[k]
				assert.Equal(t, na.ID(), nb.ID(), "curve node %d id", k)
				assert.Equal(t, na.BoneLinkProp, nb.BoneLinkProp, "curve node %d prop", k)
				if na.Bone != nil {
					require.NotNil(t, nb.Bone, "curve node %d bone", k)
					assert.Equal(t, na.Bone.ID(), nb.Bone.ID(), "curve node %d bone id", k)
				}
				for c := 0; c < 3; c++ {
					ca, cb := na.Curves[c].Curve, nb.Curves[c].Curve
					if ca == nil {
						assert.Nil(t, cb, "curve node %d curve %d", k, c)
						continue
					}
					require.NotNil(t, cb, "curve node %d curve %d", k, c)
					assert.Equal(t, ca.ID(), cb.ID(), "curve %d id", c)
					assert.Equal(t, ca.Times, cb.Times, "curve %d times", c)
					assert.Equal(t, ca.Values, cb.Values, "curve %d values", c)
				}
			}
		}
	}
}

func TestRoundTripCube(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/cube.fbx", 7400)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripCubeAs7700(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/cube.fbx", 7700)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripCubeAs7100(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/cube.fbx", 7100)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripFBXcs(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/FBXcs.fbx", 7700)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripFBXcsAs7400(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/FBXcs.fbx", 7400)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripFBXcs2(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/FBXcs2.fbx", 7700)
	compareScenes(t, scene1, scene2)
}

func TestRoundTripJyj(t *testing.T) {
	scene1, scene2 := loadAndEncode(t, "testdata/jyj.FBX", 7400)
	compareScenes(t, scene1, scene2)
}

// TestEncodeDecodesBuiltTree builds a small tree from scratch and verifies it
// survives an encode/tokenize cycle.
func TestEncodeDecodesBuiltTree(t *testing.T) {
	root := NewElement("")
	child := root.AddChildElement("Numbers",
		NewStringProperty("S value"), NewInt64Property(42), NewFloat64Property(1.5))
	child.AddChildElement("Inner", NewBoolProperty(true), NewInt16Property(-2))
	darr, err := NewDoubleArrayProperty([]float64{1, 2.5, -3}, true)
	require.NoError(t, err)
	iarr, err := NewInt32ArrayProperty([]int32{7, 8, 9, 10}, false)
	require.NoError(t, err)
	root.AddChildElement("Arrays", darr, iarr)

	for _, version := range []uint32{7100, 7400, 7500, 7700} {
		var buf bytes.Buffer
		require.NoError(t, Encode(&buf, root, version))

		elem, err := tokenize(&buf)
		require.NoError(t, err, "version %d", version)

		numbers := elem.Child("Numbers")
		require.NotNil(t, numbers, "version %d", version)
		require.Len(t, numbers.Properties, 3)
		assert.Equal(t, "S value", numbers.Properties[0].value.String())
		assert.Equal(t, int64(42), numbers.Properties[1].value.toint64())
		assert.Equal(t, 1.5, numbers.Properties[2].value.toDouble())
		inner := numbers.Child("Inner")
		require.NotNil(t, inner, "version %d", version)
		assert.True(t, inner.Properties[0].value.toBool())
		// 'Y' payloads are 2 bytes; compare the raw little endian value
		assert.Equal(t, []byte{0xFE, 0xFF}, inner.Properties[1].rawBytes())

		arrays := elem.Child("Arrays")
		require.NotNil(t, arrays, "version %d", version)
		f64s, err := parseArrayRawFloat64(arrays.Properties[0])
		require.NoError(t, err, "version %d", version)
		assert.Equal(t, []float64{1, 2.5, -3}, f64s)
		ints, err := parseArrayRawInt(arrays.Properties[1])
		require.NoError(t, err, "version %d", version)
		assert.Equal(t, []int{7, 8, 9, 10}, ints)
	}
}

// TestEncodeRejectsBadVersion guards the version switch.
func TestEncodeRejectsBadVersion(t *testing.T) {
	err := Encode(io.Discard, NewElement(""), 6000)
	assert.Error(t, err)
}

// TestEncodedFileHasValidHeader checks the container header bytes.
func TestEncodedFileHasValidHeader(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Encode(&buf, NewElement(""), 7400))
	data := buf.Bytes()
	assert.True(t, isBinary(bytes.NewReader(data)))
	assert.Equal(t, uint8(0x1a), data[21])
	assert.Equal(t, uint8(0x00), data[22])
	assert.Equal(t, uint32(7400), binary.LittleEndian.Uint32(data[23:27]))
	// file ends with the trailing footer magic
	assert.Equal(t, footerTrailingMagic, data[len(data)-len(footerTrailingMagic):])
}
