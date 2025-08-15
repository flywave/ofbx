package ofbx

import (
	"fmt"
	"io"
)

// A Scene is an overarching FBX costruct containing objects and animations
type Scene struct {
	RootElement *Element
	RootNode    *Node
	FrameRate   float32 // = -1
	Settings
	ObjectMap       map[uint64]Obj
	Meshes          []*Mesh
	AnimationStacks []*AnimationStack
	Connections     []Connection
	TakeInfos       []TakeInfo
}

func (s *Scene) String() string {
	if s == nil {
		return "nil Scene"
	}
	st := "Scene: " + "\n"
	st += "frameRate=" + fmt.Sprintf("%f", s.FrameRate) + "\n"
	st += "setttings=" + fmt.Sprintf("%+v", s.Settings) + "\n"
	if s.Meshes != nil {
		st += "meshes="
		for _, mesh := range s.Meshes {
			st += "\n"
			st += mesh.stringPrefix("\t")
		}
		st += "\n"
	}
	if s.AnimationStacks != nil {
		st += "animations="
		for _, anim := range s.AnimationStacks {
			st += "\n"
			st += anim.stringPrefix("\t")
		}
		st += "\n"
	}
	if len(s.Connections) > 0 {
		st += "connections=" + "\n"
		for _, c := range s.Connections {
			st += "\t" + c.String() + "\n"
		}
	}
	if len(s.TakeInfos) > 0 {
		st += "takeInfos=" + "\n"
		for _, tk := range s.TakeInfos {
			st += "\t" + tk.String()
		}
	}
	return st
}

// Geometries returns a scene's geometries
func (s *Scene) Geometries() []*Geometry {
	out := make([]*Geometry, 0)
	for _, o := range s.ObjectMap {
		elem := o.Element()
		if elem == nil {
			continue
		}
		if elem.ID.String() == "Geometry" {
			out = append(out, o.(*Geometry))
		}
	}
	return out
}

func (s *Scene) GetTakeInfo(name string) *TakeInfo {
	for _, info := range s.TakeInfos {
		if info.name.String() == name {
			return &info
		}
	}
	return nil
}

// Load tries to load a scene
func Load(r io.Reader) (*Scene, error) {
	s := &Scene{}
	s.ObjectMap = make(map[uint64]Obj)
	root, err := tokenize(r)
	// Todo: reimplement text
	if err != nil {
		return nil, err
	}

	s.RootElement = root

	if ok, err := parseConnection(root, s); !ok {
		return nil, err
	}
	if ok, err := parseTakes(s); !ok {
		return nil, err
	}
	if ok, err := parseObjects(root, s); !ok {
		return nil, err
	}
	parseGlobalSettings(root, s)
	s.PostProcess() // 添加后处理调用

	return s, nil
}

// 在Scene结构体中添加
func (s *Scene) PostProcess() {
	for _, m := range s.Meshes {
		m.applyLocalTransform()
	}
}
