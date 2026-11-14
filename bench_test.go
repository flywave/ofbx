package ofbx

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func benchFile(b *testing.B, path string) []byte {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("missing %s: %v", path, err)
	}
	return data
}

// BenchmarkLoadJyj loads the largest sample (750KB, 51 meshes).
func BenchmarkLoadJyj(b *testing.B) {
	data := benchFile(b, "testdata/jyj.FBX")
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadAnimSkeleton loads the animated/skinned sample.
func BenchmarkLoadAnimSkeleton(b *testing.B) {
	data := benchFile(b, "testdata/anim_skeleton.fbx")
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncodeJyj re-encodes the largest sample's element tree.
func BenchmarkEncodeJyj(b *testing.B) {
	data := benchFile(b, "testdata/jyj.FBX")
	scene, err := Load(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Encode(io.Discard, scene.RootElement, 7400); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSaveSceneAnimSkeleton exports the animated sample.
func BenchmarkSaveSceneAnimSkeleton(b *testing.B) {
	data := benchFile(b, "testdata/anim_skeleton.fbx")
	scene, err := Load(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := SaveScene(scene, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}
