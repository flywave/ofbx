package ofbx

import (
	"fmt"
	"strconv"
)

// Video objects: external or embedded texture media. A Video either
// references a file (Filename/RelativeFilename) or carries the image bytes
// inline (Content); textures reference videos through their Media property
// and an OO connection.

// Video is a media clip (texture image file or embedded content).
type Video struct {
	Object
	UseMipMap        bool
	Filename         string
	RelativeFilename string
	// Content holds the embedded image bytes when present, nil for
	// file-referenced clips.
	Content []byte
}

// NewVideo makes a stub Video object.
func NewVideo(scene *Scene, element *Element) *Video {
	v := &Video{}
	v.Object = *NewObject(scene, element)
	return v
}

// Type returns VIDEO.
func (v *Video) Type() Type {
	return VIDEO
}

func (v *Video) String() string {
	return v.stringPrefix("")
}

func (v *Video) stringPrefix(prefix string) string {
	s := prefix + "Video: " + v.Name() + "\n"
	s += prefix + "\tfilename: " + v.Filename + "\n"
	if v.Content != nil {
		s += prefix + "\tcontent: " + strconv.Itoa(len(v.Content)) + " bytes\n"
	}
	return s
}

func parseVideo(scene *Scene, element *Element) *Video {
	v := NewVideo(scene, element)
	if p := findSingleChildProperty(element, "UseMipMap"); p != nil {
		v.UseMipMap = p.value.toBool()
	}
	if p := findSingleChildProperty(element, "Filename"); p != nil {
		v.Filename = p.value.String()
	}
	if p := findSingleChildProperty(element, "RelativeFilename"); p != nil {
		v.RelativeFilename = p.value.String()
	}
	if p := findSingleChildProperty(element, "Content"); p != nil {
		switch p.Type {
		case RAWSTRING:
			v.Content = p.rawBytes()
		case ArrayBYTE:
			if b, err := parseArrayRawBytes(p); err == nil {
				v.Content = b
			}
		}
	}
	return v
}

// NewVideo creates a video clip. With content the clip is embedded
// (Content + ImageExtension written); otherwise it references filename.
func (d *Document) NewVideo(name, filename, relativeFilename string, content []byte, imageExtension string) (*Video, error) {
	el := NewElement("Video")
	el.AddProperty(NewInt64Property(0)) // placeholder id
	el.AddProperty(NewStringProperty(name + "\x00\x01Video"))
	el.AddProperty(NewStringProperty("Clip"))
	el.AddChildElement("Type", NewStringProperty("Clip"))
	el.AddChildElement("Properties70")
	if len(content) > 0 {
		if int64(len(content)) > maxArrayBytes {
			return nil, fmt.Errorf("ofbx: embedded media of %d bytes exceeds limit", len(content))
		}
		contentProp, err := NewByteArrayProperty(content, d.CompressArrays)
		if err != nil {
			return nil, err
		}
		el.AddChildElement("Content", contentProp)
		if imageExtension == "" {
			imageExtension = "jpg"
		}
		el.AddChildElement("ImageExtension", NewStringProperty(imageExtension))
	}
	el.AddChildElement("UseMipMap", NewInt32Property(0))
	el.AddChildElement("Filename", NewStringProperty(filename))
	el.AddChildElement("RelativeFilename", NewStringProperty(relativeFilename))

	video := parseVideo(d.scene, el)
	id := d.allocID()
	el.Properties[0] = NewInt64Property(int64(id))
	d.register(id, video)
	d.objs.AddChild(el)
	d.countType("Video")
	return video, nil
}

// NewEmbeddedTexture creates a texture backed by embedded image bytes: a
// Video clip with inline content plus a file texture whose Media property
// and OO connection reference it.
func (d *Document) NewEmbeddedTexture(name, imageExtension string, content []byte, filename, relativeFilename string) (*Texture, error) {
	video, err := d.NewVideo(name, filename, relativeFilename, content, imageExtension)
	if err != nil {
		return nil, err
	}
	tex := d.NewTexture(name, filename, relativeFilename)
	// Media property + connection mirror the FBX SDK wiring
	texEl := tex.Element()
	if media := texEl.Child("Media"); media == nil {
		// insert before FileName like exporters do; order is not normative,
		// appending is fine
		texEl.AddChildElement("Media", NewStringProperty(name+"\x00\x01Video"))
	}
	d.connect(video, tex, "")
	return tex, nil
}
