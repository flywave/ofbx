package ofbx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"testing"
)

func TestPrintScene(t *testing.T) {
	f, err := os.Open("./testdata/FBXcs2.fbx")
	if err != nil {
		log.Fatal(err)
	}
	scene, err := Load(f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(scene)
	fmt.Println(scene.Meshes[0].Materials[0].Textures[0].relativeFilename)
}

type Model struct {
	Vertices []float32
	Normals  []float32
	UVs      []float32
	Indices  [][]int
}

func LoadFBXModel(path string) ([]*Model, map[uint64][16]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse FBX file: %w", err)
	}
	defer file.Close()

	scene, err := Load(file)
	if err != nil {
		return nil, nil, err
	}

	mmap := make(map[uint64][16]float64)

	var models []*Model
	for _, mesh := range scene.Meshes {
		model := &Model{}
		geo := mesh.Geometry
		if geo == nil {
			continue
		}

		// 在矩阵计算中应用
		matrix := mesh.GetGlobalMatrix()
		mmap[mesh.id] = matrix.ToArray()
		// 转换顶点并应用变换矩阵
		for _, v := range geo.Vertices {
			transformed := matrix.MulPosition(v)
			// 添加场景单位缩放处理
			model.Vertices = append(model.Vertices,
				float32(transformed.X()),
				float32(transformed.Y()),
				float32(transformed.Z()))
		}

		// 转换法线向量（需要去除缩放影响）
		normalMatrix := matrix.RemoveScale()
		for _, n := range geo.Normals {
			transformed := normalMatrix.MulDirection(n)
			model.Normals = append(model.Normals,
				float32(transformed.X()),
				float32(transformed.Y()),
				float32(transformed.Z()))
		}

		// UV坐标保持不变
		if len(geo.UVs) > 0 {
			for _, uv := range geo.UVs[0] {
				model.UVs = append(model.UVs,
					float32(uv.X()),
					float32(uv.Y()))
			}
		}

		// 面索引
		model.Indices = append(model.Indices, geo.Faces...)

		models = append(models, model)
	}

	return models, mmap, nil
}

// 添加批量导出函数
func ExportModelsToOBJ(models []*Model, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var vertexOffset, uvOffset, normalOffset int

	for modelIndex, model := range models {
		// 写入当前模型的顶点数据
		for i := 0; i < len(model.Vertices); i += 3 {
			fmt.Fprintf(file, "v %.6f %.6f %.6f\n",
				model.Vertices[i],
				model.Vertices[i+1],
				model.Vertices[i+2])
		}

		// 写入当前模型的UV坐标
		for i := 0; i < len(model.UVs); i += 2 {
			fmt.Fprintf(file, "vt %.6f %.6f\n",
				model.UVs[i],
				model.UVs[i+1])
		}

		// 写入当前模型的法线
		for i := 0; i < len(model.Normals); i += 3 {
			fmt.Fprintf(file, "vn %.6f %.6f %.6f\n",
				model.Normals[i],
				model.Normals[i+1],
				model.Normals[i+2])
		}

		// 写入当前模型的面数据（调整索引偏移）
		for _, face := range model.Indices {
			fmt.Fprint(file, "f ")
			for i, idx := range face {
				if i > 0 {
					fmt.Fprint(file, " ")
				}
				// 计算全局索引：当前模型偏移 + 原始索引 + 1
				vIdx := vertexOffset/3 + idx + 1
				vtIdx := uvOffset/2 + idx + 1
				vnIdx := normalOffset/3 + idx + 1
				fmt.Fprintf(file, "%d/%d/%d", vIdx, vtIdx, vnIdx)
			}
			fmt.Fprintln(file)
		}

		// 更新偏移量供下一个模型使用
		vertexOffset += len(model.Vertices)
		uvOffset += len(model.UVs)
		normalOffset += len(model.Normals)

		// 添加模型分隔注释
		fmt.Fprintf(file, "\n# Model %d\n", modelIndex+1)
	}

	return nil
}
func TestMatrix(t *testing.T) {
	f, mmap, err := LoadFBXModel("./testdata/jyj.FBX")
	if err != nil {
		log.Fatal(err)
	}

	ExportModelsToOBJ(f, "./testdata/jyj.obj")

	data, _ := json.Marshal(mmap)
	ioutil.WriteFile("testdata/data.json", data, os.ModePerm)
}
func TestMatrix2(t *testing.T) {
	onlyInA, onlyInB, diffIDs, err := CompareMatrices("./testdata/data2.json", "./testdata/data.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("仅存在于data2.json的ID:", onlyInA)
	fmt.Println("仅存在于data.json的ID:", onlyInB)
	fmt.Println("矩阵不同的ID:", diffIDs)

}

func CompareMatrices(file1Path, file2Path string) (onlyInA, onlyInB, diffIDs []uint64, err error) {
	// 读取并解析两个JSON文件
	dataA, err := os.ReadFile(file1Path)
	if err != nil {
		return nil, nil, nil, err
	}

	dataB, err := os.ReadFile(file2Path)
	if err != nil {
		return nil, nil, nil, err
	}

	var mapA, mapB map[uint64][]float64
	if err := json.Unmarshal(dataA, &mapA); err != nil {
		return nil, nil, nil, err
	}
	if err := json.Unmarshal(dataB, &mapB); err != nil {
		return nil, nil, nil, err
	}

	// 比较ID存在性和矩阵内容
	for id, matrixA := range mapA {
		if matrixB, exists := mapB[id]; !exists {
			onlyInA = append(onlyInA, id)
		} else if !compareMatrix(matrixA, matrixB) {
			diffIDs = append(diffIDs, id)
		}
		delete(mapB, id)
	}

	// 剩余在mapB中的ID
	for id := range mapB {
		onlyInB = append(onlyInB, id)
	}

	return onlyInA, onlyInB, diffIDs, nil
}
func compareMatrix(a, b []float64) bool {
	const epsilon = 1e-9
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > epsilon {
			return false
		}
	}
	return true
}
