package ofbx

import (
	"fmt"
	"log"
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

func TestMatrix(t *testing.T) {
	f, err := os.Open("/home/hj/snap/dukto/16/md/货车2.fbx")
	if err != nil {
		log.Fatal(err)
	}
	scene, err := Load(f)
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range scene.Meshes {
		for _, m := range m.Materials {
			for _, t := range m.Textures {
				if t == nil {
					continue
				}
				fmt.Println(t.getFileName())
			}
		}
	}

}
func TestMatrix2(t *testing.T) {
	str := []int{1, 2, 3}
	fmt.Println(str[:2])
}
