package ofbx

func resolveEnumProperty(object Obj, name string, defaultVal int) int {
	element := resolveProperty(object, name)
	if element == nil {
		return defaultVal
	}
	x := element.getProperty(4)
	if x == nil {
		return defaultVal
	}

	return propInt(x)
}

func resolveVec3Property(object Obj, name string, defaultVal Point3) Point3 {
	element := resolveProperty(object, name)
	if element == nil {
		return defaultVal
	}
	if len(element.Properties) < 6 {
		return defaultVal
	}

	return Point3{
		propFloat(element.getProperty(4)),
		propFloat(element.getProperty(5)),
		propFloat(element.getProperty(6)),
	}
}

func splatVec2(mapping VertexDataMapping, data []Point2, indices []int, origIndices []int) (out []Point2) {
	switch mapping {
	case ByPolygonVertex, ByPolygon:
		if len(indices) == 0 {
			out = make([]Point2, len(data))
			copy(out, data)
		} else {
			out = make([]Point2, len(indices))
			for i := 0; i < len(indices); i++ {
				if indices[i] < len(data) {
					out[i] = data[indices[i]]
				} else {
					out[i] = Point2{}
				}
			}
		}
	case ByVertex:
		//  v0  v1 ...
		// uv0 uv1 ...

		out = make([]Point2, len(origIndices))

		for i := 0; i < len(origIndices); i++ {
			idx := origIndices[i]
			if idx < 0 {
				idx = -idx - 1
			}
			if idx < len(data) {
				out[i] = data[idx]
			} else {
				out[i] = Point2{}
			}
		}
	}
	return out
}

func splatVec3(mapping VertexDataMapping, data []Point3, indices []int, origIndices []int) (out []Point3) {
	switch mapping {
	case ByPolygonVertex, ByPolygon:
		if len(indices) == 0 {
			out = make([]Point3, len(data))
			copy(out, data)
		} else {
			out = make([]Point3, len(indices))
			for i := 0; i < len(indices); i++ {
				if indices[i] < len(data) {
					out[i] = data[indices[i]]
				} else {
					out[i] = Point3{}
				}
			}
		}
	case ByVertex:
		//  v0  v1 ...
		// uv0 uv1 ...

		out = make([]Point3, len(origIndices))

		for i := 0; i < len(origIndices); i++ {
			idx := origIndices[i]
			if idx < 0 {
				idx = -idx - 1
			}
			if idx < len(data) {
				out[i] = data[idx]
			} else {
				out[i] = Point3{}
			}
		}
	default:
		out = make([]Point3, len(origIndices))
	}
	return out
}

func splatVec4(mapping VertexDataMapping, data []Point4, indices []int, origIndices []int) (out []Point4) {
	switch mapping {
	case ByPolygonVertex, ByPolygon:
		if len(indices) == 0 {
			out = make([]Point4, len(data))
			copy(out, data)
		} else {
			out = make([]Point4, len(indices))
			for i := 0; i < len(indices); i++ {
				if indices[i] < len(data) {
					out[i] = data[indices[i]]
				} else {
					out[i] = Point4{}
				}
			}
		}
	case ByVertex:
		//  v0  v1 ...
		// uv0 uv1 ...

		out = make([]Point4, len(origIndices))

		for i := 0; i < len(origIndices); i++ {
			idx := origIndices[i]
			if idx < 0 {
				idx = -idx - 1
			}
			if idx < len(data) {
				out[i] = data[idx]
			} else {
				out[i] = Point4{}
			}
		}
	default:
		out = make([]Point4, len(origIndices))
	}
	return out
}

func RemapVec2(out *[]Point2, m []int) {
	if len(*out) == 0 {
		return
	}

	old := make([]Point2, len(*out))
	copy(old, *out)
	for i := 0; i < len(m); i++ {
		if m[i] < len(old) {
			*out = append(*out, old[m[i]])
		} else {
			*out = append(*out, Point2{})
		}
	}
}

func RemapVec3(out *[]Point3, m []int) {
	if len(*out) == 0 {
		return
	}

	old := make([]Point3, len(*out))
	copy(old, *out)
	for i := 0; i < len(m); i++ {
		if m[i] < len(old) {
			*out = append(*out, old[m[i]])
		} else {
			*out = append(*out, Point3{})
		}
	}
}

func RemapVec4(out *[]Point4, m []int) {
	if len(*out) == 0 {
		return
	}

	old := make([]Point4, len(*out))
	copy(old, *out)
	for i := 0; i < len(m); i++ {
		if m[i] < len(old) {
			*out = append(*out, old[m[i]])
		} else {
			*out = append(*out, Point4{})
		}
	}
}
