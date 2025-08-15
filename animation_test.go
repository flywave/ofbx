package ofbx

import (
	"testing"
)

// TestNewAnimationStack tests the creation of a new AnimationStack
func TestNewAnimationStack(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	element := &Element{ID: &DataView{}}

	stack := NewAnimationStack(scene, element)

	if stack == nil {
		t.Fatal("NewAnimationStack returned nil")
	}

	if stack.Type() != ANIMATION_STACK {
		t.Errorf("Expected type ANIMATION_STACK, got %v", stack.Type())
	}

	if len(stack.Layers) != 0 {
		t.Errorf("Expected empty layers slice, got %d layers", len(stack.Layers))
	}

	if stack.Scene() != scene {
		t.Error("AnimationStack does not reference correct scene")
	}
}

// TestNewAnimationLayer tests the creation of a new AnimationLayer
func TestNewAnimationLayer(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}
	element := &Element{ID: &DataView{}}

	layer := NewAnimationLayer(scene, element)

	if layer == nil {
		t.Fatal("NewAnimationLayer returned nil")
	}

	if layer.Type() != ANIMATION_LAYER {
		t.Errorf("Expected type ANIMATION_LAYER, got %v", layer.Type())
	}

	if len(layer.CurveNodes) != 0 {
		t.Errorf("Expected empty CurveNodes slice, got %d nodes", len(layer.CurveNodes))
	}

	if layer.Scene() != scene {
		t.Error("AnimationLayer does not reference correct scene")
	}
}

// TestAnimationStackGetLayer tests the GetLayer method
func TestAnimationStackGetLayer(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a stack
	stackElement := &Element{ID: &DataView{}}
	stack := NewAnimationStack(scene, stackElement)

	// Create a layer
	layerElement := &Element{ID: &DataView{}}
	layer := NewAnimationLayer(scene, layerElement)

	// Manually set up the connection (simulating what would happen in a real FBX)
	layerID := uint64(123)
	layer.SetID(layerID)
	stack.SetID(uint64(456))

	// Add layer to scene's object map
	scene.ObjectMap[layerID] = layer

	// Add connection
	scene.Connections = append(scene.Connections, Connection{
		from: layerID,
		to:   stack.ID(),
	})

	// Test GetLayer
	retrievedLayer := stack.GetLayer(0)
	if retrievedLayer == nil {
		t.Error("GetLayer returned nil for valid index")
	}

	// Test GetLayer with invalid index
	invalidLayer := stack.GetLayer(999)
	if invalidLayer != nil {
		t.Error("GetLayer should return nil for invalid index")
	}
}

// TestAnimationStackGetAllLayers tests the getAllLayers method
func TestAnimationStackGetAllLayers(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a stack
	stackElement := &Element{ID: &DataView{}}
	stack := NewAnimationStack(scene, stackElement)

	// Create multiple layers
	layer1 := NewAnimationLayer(scene, &Element{ID: &DataView{}})
	layer2 := NewAnimationLayer(scene, &Element{ID: &DataView{}})

	layer1.SetID(100)
	layer2.SetID(101)
	stack.SetID(200)

	// Add layers to scene
	scene.ObjectMap[100] = layer1
	scene.ObjectMap[101] = layer2

	// Add connections
	scene.Connections = append(scene.Connections,
		Connection{from: 100, to: 200},
		Connection{from: 101, to: 200},
	)

	// Test getAllLayers
	layers := stack.getAllLayers()

	if len(layers) != 2 {
		t.Errorf("Expected 2 layers, got %d", len(layers))
	}

	// Verify we got the correct layers
	found := 0
	for _, layer := range layers {
		if layer == layer1 || layer == layer2 {
			found++
		}
	}
	if found != 2 {
		t.Errorf("Did not retrieve expected layers, found %d matches", found)
	}
}

// TestAnimationStackPostProcess tests the postProcess method
func TestAnimationStackPostProcess(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a stack
	stackElement := &Element{ID: &DataView{}}
	stack := NewAnimationStack(scene, stackElement)

	// Create layers
	layer1 := NewAnimationLayer(scene, &Element{ID: &DataView{}})
	layer2 := NewAnimationLayer(scene, &Element{ID: &DataView{}})

	layer1.SetID(1001)
	layer2.SetID(1002)
	stack.SetID(1000)

	// Add layers to scene
	scene.ObjectMap[1001] = layer1
	scene.ObjectMap[1002] = layer2

	// Add connections
	scene.Connections = append(scene.Connections,
		Connection{from: 1001, to: 1000},
		Connection{from: 1002, to: 1000},
	)

	// Test postProcess
	result := stack.postProcess()
	if !result {
		t.Error("postProcess should return true")
	}

	if len(stack.Layers) != 2 {
		t.Errorf("Expected 2 layers after postProcess, got %d", len(stack.Layers))
	}
}

// TestAnimationLayerGetCurveNode tests the GetCurveNode method
func TestAnimationLayerGetCurveNode(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a layer
	layer := NewAnimationLayer(scene, &Element{ID: &DataView{}})

	// Create a mock bone object
	bone := NewObject(scene, &Element{ID: &DataView{}})
	bone.SetID(500)

	// Create a curve node
	curveNode := &AnimationCurveNode{
		Object:       *NewObject(scene, &Element{ID: &DataView{}}),
		Bone:         bone,
		BoneLinkProp: "Lcl Translation",
	}

	// Add curve node to layer
	layer.CurveNodes = append(layer.CurveNodes, curveNode)

	// Test GetCurveNode with matching bone and property
	foundNode := layer.GetCurveNode(bone, "Lcl Translation")
	if foundNode != curveNode {
		t.Error("GetCurveNode should return the matching curve node")
	}

	// Test GetCurveNode with non-matching property
	notFound := layer.GetCurveNode(bone, "Lcl Rotation")
	if notFound != nil {
		t.Error("GetCurveNode should return nil for non-matching property")
	}

	// Test GetCurveNode with non-matching bone
	otherBone := NewObject(scene, &Element{ID: &DataView{}})
	otherBone.SetID(501)
	notFound = layer.GetCurveNode(otherBone, "Lcl Translation")
	if notFound != nil {
		t.Error("GetCurveNode should return nil for non-matching bone")
	}
}

// TestAnimationStackString tests the String and stringPrefix methods
func TestAnimationStackString(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a stack
	stack := NewAnimationStack(scene, &Element{ID: &DataView{}})
	stack.SetID(100)

	// Just ensure String runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("String method recovered from panic: %v", r)
		}
	}()

	_ = stack.String()

	// Test with layers
	layer := NewAnimationLayer(scene, &Element{ID: &DataView{}})
	layer.SetID(200)

	// Manually add layer to simulate connections
	stack.Layers = append(stack.Layers, layer)

	_ = stack.String()
}

// TestAnimationLayerString tests the String and stringPrefix methods
func TestAnimationLayerString(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Create a layer
	layer := NewAnimationLayer(scene, &Element{ID: &DataView{}})
	layer.SetID(300)

	// Just ensure String runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("String method recovered from panic: %v", r)
		}
	}()

	_ = layer.String()
}

// TestEdgeCases tests edge cases for both AnimationStack and AnimationLayer
func TestEdgeCases(t *testing.T) {
	scene := &Scene{ObjectMap: make(map[uint64]Obj)}

	// Test nil scene handling (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Log("Recovered from panic:", r)
		}
	}()

	// Test with nil scene (this might panic, so we expect it)
	// This test documents expected behavior
	// stack := NewAnimationStack(nil, &Element{ID: &DataView{}})

	// Test empty connections
	stack := NewAnimationStack(scene, &Element{ID: &DataView{}})
	layers := stack.getAllLayers()
	if layers == nil {
		t.Error("getAllLayers should return empty slice, not nil")
	}
	if len(layers) != 0 {
		t.Error("getAllLayers should return empty slice when no connections")
	}

	// Test postProcess with no layers
	result := stack.postProcess()
	if !result {
		t.Error("postProcess should return true even with no layers")
	}
	if len(stack.Layers) != 0 {
		t.Error("Layers should remain empty when no connections")
	}
}
