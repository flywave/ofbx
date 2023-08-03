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
	f, err := os.Open("/home/hj/snap/dukto/16/md/AOI.fbx")
	if err != nil {
		log.Fatal(err)
	}
	scene, err := Load(f)
	if err != nil {
		log.Fatal(err)
	}

	mt := getGlobalTransform(scene.ObjectMap[2281135157232])
	fmt.Println(mt)
	mt2 := getGlobalTransform(scene.ObjectMap[2281135157232])
	fmt.Println(mt2)

	for _, m := range scene.Meshes[12:] {
		// mt := getLocalTransform(m)
		// fmt.Println(mt)

		mt2 := getGlobalTransform(m)
		fmt.Println(mt2)

	}

}
