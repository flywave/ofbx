package ofbx

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncodeTextRoundTripCube converts a binary scene to ASCII and back.
func TestEncodeTextRoundTripCube(t *testing.T) {
	f, err := os.Open("testdata/cube.fbx")
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var text bytes.Buffer
	require.NoError(t, EncodeText(&text, scene1.RootElement, 7400))
	// sanity: the output really is text
	assert.Contains(t, text.String(), "; FBX 7.4.0 project file")

	scene2, err := Load(&text)
	require.NoError(t, err)
	compareScenes(t, scene1, scene2)
}

// TestEncodeTextRoundTripDocument builds from scratch, exports text and
// verifies the scene reloaded from text.
func TestEncodeTextRoundTripDocument(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	require.NoError(t, err)
	require.NoError(t, doc.SetNormals(mesh, cubeCornerNormals()))
	require.NoError(t, doc.SetUVs(mesh, 0, cubeCornerUVs()))
	mat := doc.NewMaterial("M")
	mat.DiffuseColor = Color{0.2, 0.4, 0.6}
	doc.AddMaterial(mesh, mat)

	doc.Prepare()
	var text bytes.Buffer
	require.NoError(t, EncodeText(&text, doc.root, 7400))
	scene, err := Load(&text)
	require.NoError(t, err)

	require.Len(t, scene.Meshes, 1)
	g := scene.Meshes[0].Geometry
	assert.Equal(t, cubeVertices, g.Vertices)
	assert.Equal(t, normalized(cubeCornerNormals()), g.Normals)
	assert.Equal(t, cubeCornerUVs(), g.UVs[0])
	require.Len(t, scene.Meshes[0].Materials, 1)
	assert.Equal(t, Color{0.2, 0.4, 0.6}, scene.Meshes[0].Materials[0].DiffuseColor)
}

// TestTextParsesLiterals checks the text tokenizer's number, string, boolean
// and array literals.
func TestTextParsesLiterals(t *testing.T) {
	src := `; comment
Root:  {
	Num: 42, -7, 3.5, -2.25e-2
	Str: "hello \"world\"", ""
	Flag: T, F
	Arr: *4 { a: 1,2,3,4 }
	DArr: *3 {
		a: 0.5,1.5,-2.0
	}
	NoProps: {
	}
}`
	root, err := tokenize(bytes.NewReader([]byte(src)))
	require.NoError(t, err)
	require.Len(t, root.Children, 1)
	el := root.Children[0]

	num := el.Child("Num")
	require.NotNil(t, num)
	require.Len(t, num.Properties, 4)
	assert.Equal(t, 42, propInt(num.Properties[0]))
	assert.Equal(t, -7, propInt(num.Properties[1]))
	assert.Equal(t, 3.5, num.Properties[2].value.toDouble())
	assert.InDelta(t, -0.0225, num.Properties[3].value.toDouble(), 1e-12)

	str := el.Child("Str")
	require.NotNil(t, str)
	assert.Equal(t, `hello "world"`, str.Properties[0].value.String())
	assert.Equal(t, "", str.Properties[1].value.String())

	flag := el.Child("Flag")
	require.NotNil(t, flag)
	assert.True(t, flag.Properties[0].value.toBool())
	assert.False(t, flag.Properties[1].value.toBool())

	arr := el.Child("Arr")
	require.NotNil(t, arr)
	ints, err := parseArrayRawInt64(arr.Properties[0])
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3, 4}, ints)

	darr := el.Child("DArr")
	require.NotNil(t, darr)
	fl, err := parseArrayRawFloat64(darr.Properties[0])
	require.NoError(t, err)
	assert.Equal(t, []float64{0.5, 1.5, -2.0}, fl)

	require.NotNil(t, el.Child("NoProps"))
}

// TestTextRejectsGarbage guards the text parser's error paths.
func TestTextRejectsGarbage(t *testing.T) {
	for _, src := range []string{
		"garbage without colon",
		"Bad: *3 { a: 1,2 }", // count mismatch
		"Bad: *x { a: 1 }",   // bad count
		"Bad: *2 { b: 1,2 }", // missing a:
	} {
		_, err := tokenize(bytes.NewReader([]byte(src)))
		assert.Error(t, err, "input %q", src)
	}
}
