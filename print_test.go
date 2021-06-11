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
