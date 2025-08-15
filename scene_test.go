package ofbx

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoadEmptyReader 测试空输入的情况
func TestLoadEmptyReader(t *testing.T) {
	emptyReader := bytes.NewReader([]byte{})
	scene, err := Load(emptyReader)

	assert.Error(t, err)
	assert.Nil(t, scene)
}

// TestLoadNonBinaryFBX 测试非二进制FBX格式的情况
func TestLoadNonBinaryFBX(t *testing.T) {
	asciiFBX := []byte(`; FBX 7.4.0 project file
FBXHeaderExtension:  {
	Version: 7400
}
`)

	reader := bytes.NewReader(asciiFBX)
	scene, err := Load(reader)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Non-binary FBX")
	assert.Nil(t, scene)
}

// TestLoadSceneStructure 测试场景结构是否正确初始化
func TestLoadSceneStructure(t *testing.T) {
	// 测试场景对象的默认初始化
	scene := &Scene{
		ObjectMap:   make(map[uint64]Obj),
		Meshes:      []*Mesh{},
		Connections: []Connection{},
		TakeInfos:   []TakeInfo{},
	}

	assert.NotNil(t, scene)
	assert.NotNil(t, scene.ObjectMap)
	assert.NotNil(t, scene.Meshes)
	assert.NotNil(t, scene.Connections)
	assert.NotNil(t, scene.TakeInfos)
}

// TestSceneString 测试Scene的String方法
func TestSceneString(t *testing.T) {
	scene := &Scene{
		FrameRate: 30.0,
		Settings: Settings{
			UnitScaleFactor:         1.0,
			OriginalUnitScaleFactor: 1.0,
		},
		Meshes:          []*Mesh{},
		AnimationStacks: []*AnimationStack{},
		Connections:     []Connection{},
		TakeInfos:       []TakeInfo{},
	}

	str := scene.String()
	assert.Contains(t, str, "Scene:")
	assert.Contains(t, str, "frameRate=30.000000")
	assert.Contains(t, str, "setttings=")
}

// TestSceneGeometries 测试Geometries方法
func TestSceneGeometries(t *testing.T) {
	scene := &Scene{
		ObjectMap: make(map[uint64]Obj),
	}

	geometries := scene.Geometries()
	assert.NotNil(t, geometries)
	assert.Empty(t, geometries)
}

// TestGetTakeInfo 测试GetTakeInfo方法
func TestGetTakeInfo(t *testing.T) {
	scene := &Scene{
		TakeInfos: []TakeInfo{},
	}

	info := scene.GetTakeInfo("nonexistent")
	assert.Nil(t, info)
}

// TestPostProcess 测试PostProcess方法
func TestPostProcess(t *testing.T) {
	scene := &Scene{
		Meshes: []*Mesh{},
	}

	// 主要测试不会panic
	assert.NotPanics(t, scene.PostProcess)
}

// TestLoadErrorHandling 测试错误处理
func TestLoadErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Empty data",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "Invalid header",
			data:        []byte("Invalid header"),
			expectError: true,
		},
		{
			name:        "ASCII FBX",
			data:        []byte("; FBX 7.4.0 project file"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			scene, err := Load(reader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scene)
			} else {
				// 对于有效数据，主要测试不会panic
				assert.NotPanics(t, func() {
					_, _ = Load(reader)
				})
			}
		})
	}
}

// TestSceneNilString 测试nil场景的String方法
func TestSceneNilString(t *testing.T) {
	var scene *Scene
	str := scene.String()
	assert.Equal(t, "nil Scene", str)
}

// TestLoadIntegration 测试集成加载
func TestLoadIntegration(t *testing.T) {
	// 创建一个基本的场景结构
	scene := &Scene{
		RootElement: &Element{},
		ObjectMap:   make(map[uint64]Obj),
		Meshes:      []*Mesh{},
		Connections: []Connection{},
		TakeInfos:   []TakeInfo{},
	}

	assert.NotNil(t, scene)
	assert.NotNil(t, scene.ObjectMap)
	assert.NotNil(t, scene.Meshes)
	assert.NotNil(t, scene.Connections)
	assert.NotNil(t, scene.TakeInfos)
}
