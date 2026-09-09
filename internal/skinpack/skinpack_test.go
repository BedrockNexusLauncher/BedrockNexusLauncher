package skinpack

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func testPNG() string {
	var buffer bytes.Buffer
	imageData := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			imageData.Set(x, y, color.RGBA{R: 80, G: 140, B: 220, A: 255})
		}
	}
	_ = png.Encode(&buffer, imageData)
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func TestCreateRejectsInvalidSkinData(t *testing.T) {
	invalid := base64.StdEncoding.EncodeToString([]byte("not-a-real-png"))
	if _, err := Create("test", []SkinInput{{Name: "Skin", Data: invalid}}); err == nil {
		t.Fatal("expected invalid PNG data to be rejected")
	}
}

func TestCreateProducesMcpackArchive(t *testing.T) {
	png := testPNG()
	archivePath, err := Create("Test Pack", []SkinInput{{Name: "Alex", Data: png}})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(archivePath)

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	wanted := map[string]bool{"manifest.json": false, "skins.json": false, "texts/en_US.lang": false, "skin_0.png": false}
	for _, file := range archive.File {
		if _, ok := wanted[file.Name]; ok {
			wanted[file.Name] = true
		}
	}
	for name, found := range wanted {
		if !found {
			t.Fatalf("archive missing %s", name)
		}
	}
}

func TestCreateHeightFallsBackToStandardMcpack(t *testing.T) {
	png := testPNG()
	archivePath, err := Create("Height Pack", []SkinInput{{Name: "Short", Geometry: "geometry.humanoid.custom", Height: "short", Data: png}})
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(archivePath)
	if filepath.Ext(archivePath) != ".mcpack" {
		t.Fatalf("expected mcpack, got %s", archivePath)
	}

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	wanted := map[string]bool{"manifest.json": false, "skins.json": false, "skin_0.png": false}
	for _, file := range archive.File {
		if _, ok := wanted[file.Name]; ok {
			wanted[file.Name] = true
		}
	}
	for name, found := range wanted {
		if !found {
			t.Fatalf("standard skin pack missing %s", name)
		}
	}
}

func TestEntriesAndUpdate(t *testing.T) {
	root := t.TempDir()
	png := testPNG()
	if err := renderFiles(root, "Test Pack", []Entry{{Name: "Old", Geometry: "geometry.humanoid.customSlim", Texture: "skin_0.png", Data: png}}, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := Entries(root)
	if err != nil || len(entries) != 1 || entries[0].Name != "Old" || entries[0].Geometry != "geometry.humanoid.customSlim" {
		t.Fatalf("unexpected entries: %#v, %v", entries, err)
	}
	if err := Update(root, "Updated Pack", []SkinInput{{Name: "New", Data: png}}); err != nil {
		t.Fatal(err)
	}
	updated, err := Entries(root)
	if err != nil || len(updated) != 1 || updated[0].Name != "New" {
		t.Fatalf("unexpected updated entries: %#v, %v", updated, err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}
