package ofbx

import (
	"fmt"
)

// Cameras and lights: Model elements with the "Camera"/"Light" subclass
// carry the node transform; the matching NodeAttribute holds the parameters.

// Camera is a camera node.
type Camera struct {
	Object
}

// NewCamera makes a stub Camera object.
func NewCamera(scene *Scene, element *Element) *Camera {
	c := &Camera{}
	c.Object = *NewObject(scene, element)
	c.Object.isNode = true
	return c
}

// Type returns CAMERA.
func (c *Camera) Type() Type { return CAMERA }

func (c *Camera) String() string { return c.stringPrefix("") }

func (c *Camera) stringPrefix(prefix string) string {
	return prefix + "Camera: " + c.Name() + "\n" + c.Object.stringPrefix(prefix+"\t")
}

// CameraAttributes are read from a camera's NodeAttribute.
type CameraAttributes struct {
	Object
	FieldOfView          float64
	FocalLength          float64
	NearPlane            float64
	FarPlane             float64
	ApertureMode         string
	Position, Up, LookAt Point3
}

// NewCameraAttributes makes a stub camera attribute object.
func NewCameraAttributes(scene *Scene, element *Element) *CameraAttributes {
	a := &CameraAttributes{}
	a.Object = *NewObject(scene, element)
	return a
}

// Type returns NODE_ATTRIBUTE.
func (a *CameraAttributes) Type() Type { return NODE_ATTRIBUTE }

func (a *CameraAttributes) String() string { return a.stringPrefix("") }

func (a *CameraAttributes) stringPrefix(prefix string) string {
	return fmt.Sprintf("%sCameraAttributes: %s fov=%v focal=%v\n",
		prefix, a.Name(), a.FieldOfView, a.FocalLength)
}

func parseCameraAttributes(scene *Scene, element *Element) *CameraAttributes {
	a := NewCameraAttributes(scene, element)
	a.FieldOfView = propFloat(resolvePropertyP(element, "FieldOfView"))
	a.FocalLength = propFloat(resolvePropertyP(element, "FocalLength"))
	a.NearPlane = propFloat(resolvePropertyP(element, "NearPlane"))
	a.FarPlane = propFloat(resolvePropertyP(element, "FarPlane"))
	if p := resolvePropertyP(element, "ApertureMode"); p != nil {
		a.ApertureMode = p.value.String()
	}
	a.Position = vec3FromProps(element, "Position")
	a.Up = vec3FromProps(element, "Up")
	a.LookAt = vec3FromProps(element, "LookAt")
	return a
}

// vec3FromProps reads a child element holding three double properties.
func vec3FromProps(element *Element, name string) Point3 {
	p := findChildProperty(element, name)
	if len(p) < 3 {
		return Point3{}
	}
	return Point3{propFloat(p[0]), propFloat(p[1]), propFloat(p[2])}
}

// Light is a light node.
type Light struct {
	Object
}

// NewLight makes a stub Light object.
func NewLight(scene *Scene, element *Element) *Light {
	l := &Light{}
	l.Object = *NewObject(scene, element)
	l.Object.isNode = true
	return l
}

// Type returns LIGHT.
func (l *Light) Type() Type { return LIGHT }

func (l *Light) String() string { return l.stringPrefix("") }

func (l *Light) stringPrefix(prefix string) string {
	return prefix + "Light: " + l.Name() + "\n" + l.Object.stringPrefix(prefix+"\t")
}

// LightAttributes are read from a light's NodeAttribute.
type LightAttributes struct {
	Object
	LightType string
	Color     Color
	Intensity float64
}

// NewLightAttributes makes a stub light attribute object.
func NewLightAttributes(scene *Scene, element *Element) *LightAttributes {
	a := &LightAttributes{}
	a.Object = *NewObject(scene, element)
	return a
}

// Type returns NODE_ATTRIBUTE.
func (a *LightAttributes) Type() Type { return NODE_ATTRIBUTE }

func (a *LightAttributes) String() string { return a.stringPrefix("") }

func (a *LightAttributes) stringPrefix(prefix string) string {
	return fmt.Sprintf("%sLightAttributes: %s type=%s intensity=%v\n",
		prefix, a.Name(), a.LightType, a.Intensity)
}

func parseLightAttributes(scene *Scene, element *Element) *LightAttributes {
	a := NewLightAttributes(scene, element)
	// the light type lives in the TypeFlags child ("Point", "Spot", ...),
	// with a Properties70 "LightType" enum as fallback
	if tf := findChildProperty(element, "TypeFlags"); len(tf) > 0 {
		a.LightType = tf[0].value.String()
	} else if p := resolvePropertyP(element, "LightType"); p != nil {
		a.LightType = p.value.String()
	}
	if e := findColorP(element, "Color"); e != nil {
		a.Color = *e
	}
	a.Intensity = propFloat(resolvePropertyP(element, "Intensity"))
	return a
}

// findColorP reads a three component color P entry.
func findColorP(element *Element, name string) *Color {
	for _, p70 := range element.ChildrenOf("Properties70") {
		for _, p := range p70.Children {
			if prop := p.getProperty(0); prop != nil && prop.value.String() == name && len(p.Properties) >= 7 {
				return &Color{
					float32(propFloat(p.getProperty(4))),
					float32(propFloat(p.getProperty(5))),
					float32(propFloat(p.getProperty(6))),
				}
			}
		}
	}
	return nil
}

// ---------- document builders ----------

// NewCamera creates a camera node with a CameraNodeAttribute carrying the
// given parameters (field of view in degrees).
func (d *Document) NewCamera(name string, fieldOfView, focalLength float64) (*Camera, error) {
	el := NewElement("Model")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Model"))
	el.AddProperty(NewStringProperty("Camera"))
	el.AddChildElement("Version", NewInt32Property(232))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pLclVec3("Lcl Translation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Rotation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Scaling", 1, 1, 1))
	el.AddChildElement("Shading", NewBoolProperty(true))
	el.AddChildElement("Culling", NewStringProperty("CullingOff"))

	cam := NewCamera(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, cam)
	d.objs.AddChild(el)
	d.countType("Model")

	attrEl := NewElement("NodeAttribute")
	attrEl.AddProperty(NewInt64Property(0)) // placeholder id
	attrEl.AddProperty(NewStringProperty(name + "\x00\x01NodeAttribute"))
	attrEl.AddProperty(NewStringProperty("Camera"))
	attrEl.AddChildElement("TypeFlags", NewStringProperty("Camera"))
	attrP70 := attrEl.AddChildElement("Properties70")
	attrP70.AddChild(pEntry("FieldOfView", "FieldOfView", "", "A", NewFloat64Property(fieldOfView)))
	attrP70.AddChild(pEntry("FocalLength", "FocalLength", "", "A", NewFloat64Property(focalLength)))

	attr := parseCameraAttributes(d.scene, attrEl)
	attrID := d.allocID()
	attrEl.Properties[0] = NewInt64Property(int64(attrID))
	d.register(attrID, attr)
	d.objs.AddChild(attrEl)
	cam.SetNodeAttribute(attr)
	d.connect(attr, cam, "")
	d.countType("NodeAttribute")
	return cam, nil
}

// NewLight creates a light node; lightType is one of "Point", "Spot",
// "Directional", "Area" or "Volume".
func (d *Document) NewLight(name, lightType string, color Color, intensity float64) (*Light, error) {
	switch lightType {
	case "Point", "Spot", "Directional", "Area", "Volume":
	default:
		return nil, fmt.Errorf("ofbx: unsupported light type %q", lightType)
	}
	el := NewElement("Model")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Model"))
	el.AddProperty(NewStringProperty("Light"))
	el.AddChildElement("Version", NewInt32Property(232))
	p70 := el.AddChildElement("Properties70")
	p70.AddChild(pLclVec3("Lcl Translation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Rotation", 0, 0, 0))
	p70.AddChild(pLclVec3("Lcl Scaling", 1, 1, 1))

	light := NewLight(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, light)
	d.objs.AddChild(el)
	d.countType("Model")

	attrEl := NewElement("NodeAttribute")
	attrEl.AddProperty(NewInt64Property(0)) // placeholder id
	attrEl.AddProperty(NewStringProperty(name + "\x00\x01NodeAttribute"))
	attrEl.AddProperty(NewStringProperty("Light"))
	attrEl.AddChildElement("TypeFlags", NewStringProperty(lightType))
	attrP70 := attrEl.AddChildElement("Properties70")
	attrP70.AddChild(pColor("Color", color))
	attrP70.AddChild(pEntry("Intensity", "Number", "", "A", NewFloat64Property(intensity)))

	attr := parseLightAttributes(d.scene, attrEl)
	attrID := d.allocID()
	attrEl.Properties[0] = NewInt64Property(int64(attrID))
	d.register(attrID, attr)
	d.objs.AddChild(attrEl)
	light.SetNodeAttribute(attr)
	d.connect(attr, light, "")
	d.countType("NodeAttribute")
	return light, nil
}
