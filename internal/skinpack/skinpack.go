package skinpack

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SkinInput struct {
	Name     string `json:"name"`
	Geometry string `json:"geometry"`
	Height   string `json:"height"`
	Data     string `json:"data"`
}

type Entry struct {
	Name     string `json:"name"`
	Geometry string `json:"geometry"`
	Height   string `json:"height"`
	Texture  string `json:"texture"`
	Data     string `json:"data"`
}

type manifest struct {
	FormatVersion int              `json:"format_version"`
	Header        manifestHeader   `json:"header"`
	Modules       []manifestModule `json:"modules"`
}

type manifestHeader struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	UUID             string `json:"uuid"`
	Version          [3]int `json:"version"`
	MinEngineVersion [3]int `json:"min_engine_version"`
}

type manifestModule struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid"`
	Version [3]int `json:"version"`
}

type skinManifest struct {
	Skins            []skinManifestEntry `json:"skins"`
	SerializeName    string              `json:"serialize_name"`
	LocalizationName string              `json:"localization_name"`
}

type skinManifestEntry struct {
	LocalizationName string `json:"localization_name"`
	Geometry         string `json:"geometry"`
	Texture          string `json:"texture"`
	Type             string `json:"type"`
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

const (
	defaultGeometry = "geometry.humanoid.custom"
	customShort     = "geometry.bedrock_nexus.short"
	customTall      = "geometry.bedrock_nexus.tall"
	customSlimShort = "geometry.bedrock_nexus.slim_short"
	customSlimTall  = "geometry.bedrock_nexus.slim_tall"
)

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeName.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		name = "Custom Skin Pack"
	}
	return name
}

func decodeInputs(inputs []SkinInput) ([]Entry, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("ERR_SKINPACK_EMPTY")
	}
	entries := make([]Entry, 0, len(inputs))
	for i, input := range inputs {
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input.Data))
		if err != nil || len(data) == 0 {
			return nil, fmt.Errorf("ERR_SKINPACK_INVALID_IMAGE_%d", i+1)
		}
		cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
		if err != nil || (cfg.Width != 64 && cfg.Width != 128) || (cfg.Height != 32 && cfg.Height != 64 && cfg.Height != 128) {
			return nil, fmt.Errorf("ERR_SKINPACK_INVALID_DIMENSIONS_%d", i+1)
		}
		geometry := strings.TrimSpace(input.Geometry)
		if geometry == "" {
			geometry = defaultGeometry
		}
		height := strings.ToLower(strings.TrimSpace(input.Height))
		if height != "short" && height != "tall" {
			height = "normal"
		}
		name := strings.TrimSpace(input.Name)
		if name == "" {
			name = fmt.Sprintf("Skin %d", i+1)
		}
		entries = append(entries, Entry{Name: name, Geometry: geometry, Height: height, Texture: fmt.Sprintf("skin_%d.png", i), Data: input.Data})
	}
	return entries, nil
}

func resolveGeometry(geometry, height string) string {
	if height == "normal" {
		return geometry
	}
	slim := strings.Contains(strings.ToLower(geometry), "slim")
	if height == "short" {
		if slim {
			return customSlimShort
		}
		return customShort
	}
	if slim {
		return customSlimTall
	}
	return customTall
}

func buildManifest(packName string) (manifest, string) {
	serializeName := cleanName(packName)
	return manifest{
		FormatVersion: 2,
		Header: manifestHeader{
			Name: packName, Description: "Custom Minecraft Bedrock skin pack", UUID: newUUID(), Version: [3]int{1, 0, 0}, MinEngineVersion: [3]int{1, 20, 0},
		},
		Modules: []manifestModule{{Type: "skin_pack", UUID: newUUID(), Version: [3]int{1, 0, 0}}},
	}, serializeName
}

func renderFiles(root, packName string, entries []Entry, preserveIcon []byte) error {
	m, serializeName := buildManifest(packName)
	manifestData, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestData, 0o644); err != nil {
		return err
	}
	items := make([]skinManifestEntry, 0, len(entries))
	for _, entry := range entries {
		data, err := base64.StdEncoding.DecodeString(entry.Data)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(entry.Texture)), data, 0o644); err != nil {
			return err
		}
		items = append(items, skinManifestEntry{
			LocalizationName: fmt.Sprintf("skin_%d", len(items)),
			Geometry:         entry.Geometry,
			Texture:          entry.Texture,
			Type:             "free",
		})
	}
	skinsData, _ := json.MarshalIndent(skinManifest{Skins: items, SerializeName: serializeName, LocalizationName: serializeName}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "skins.json"), skinsData, 0o644); err != nil {
		return err
	}
	texts := "skinpack." + serializeName + "=" + packName + "\n"
	for i, entry := range entries {
		texts += fmt.Sprintf("skinpack.%s.skin_%d=%s\n", serializeName, i, entry.Name)
	}
	if err := os.MkdirAll(filepath.Join(root, "texts"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "texts", "en_US.lang"), []byte(texts), 0o644); err != nil {
		return err
	}
	if len(preserveIcon) > 0 {
		return os.WriteFile(filepath.Join(root, "pack_icon.png"), preserveIcon, 0o644)
	}
	if len(entries) > 0 {
		icon, err := base64.StdEncoding.DecodeString(entries[0].Data)
		if err == nil {
			return os.WriteFile(filepath.Join(root, "pack_icon.png"), icon, 0o644)
		}
	}
	return nil
}

func Create(packName string, inputs []SkinInput) (string, error) {
	entries, err := decodeInputs(inputs)
	if err != nil {
		return "", err
	}
	root, err := os.MkdirTemp("", "BedrockNexus_SkinPack_")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(root)
	if err := renderFiles(root, packName, entries, nil); err != nil {
		return "", err
	}
	outDir := filepath.Join(os.TempDir(), "BedrockNexus", "SkinPacks")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	skinPack, err := zipDirectory(root)
	if err != nil {
		return "", err
	}
	archivePath := filepath.Join(outDir, cleanName(packName)+".mcpack")
	return archivePath, os.WriteFile(archivePath, skinPack, 0o644)
}

func hasCustomHeight(entries []Entry) bool {
	for _, entry := range entries {
		if entry.Height == "short" || entry.Height == "tall" {
			return true
		}
	}
	return false
}

func zipDirectory(root string) ([]byte, error) {
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(h, in)
		_ = in.Close()
		return copyErr
	})
	if err == nil {
		err = zw.Close()
	} else {
		_ = zw.Close()
	}
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeAddonArchive(path string, skinPack, resourcePack []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(file)
	for name, data := range map[string][]byte{
		"skin_pack.mcpack":     skinPack,
		"height_models.mcpack": resourcePack,
	} {
		entry, createErr := zw.Create(name)
		if createErr != nil {
			_ = file.Close()
			return createErr
		}
		if _, writeErr := entry.Write(data); writeErr != nil {
			_ = file.Close()
			return writeErr
		}
	}
	if err = zw.Close(); err == nil {
		err = file.Close()
	} else {
		_ = file.Close()
	}
	return err
}

func renderHeightResourcePack(root string) error {
	manifest := map[string]any{
		"format_version": 2,
		"header": map[string]any{
			"name":               "Bedrock Nexus Height Models",
			"description":        "Beta visual height models generated by Bedrock Nexus",
			"uuid":               newUUID(),
			"version":            []int{1, 0, 0},
			"min_engine_version": []int{1, 20, 0},
		},
		"modules": []map[string]any{{"type": "resources", "uuid": newUUID(), "version": []int{1, 0, 0}}},
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, "manifest.json"), manifestData, 0o644); err != nil {
		return err
	}
	models := map[string]any{
		"format_version":                    "1.8.0",
		"geometry.bedrock_nexus.short":      makeHeightGeometry(10, 10, 4),
		"geometry.bedrock_nexus.tall":       makeHeightGeometry(14, 14, 4),
		"geometry.bedrock_nexus.slim_short": makeHeightGeometry(10, 10, 3),
		"geometry.bedrock_nexus.slim_tall":  makeHeightGeometry(14, 14, 3),
	}
	modelsData, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(root, "models"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "models", "mobs.json"), modelsData, 0o644)
}

func makeHeightGeometry(legHeight, bodyHeight, armWidth int) map[string]any {
	bodyTop := legHeight + bodyHeight
	armOrigin := -8
	leftArmOrigin := 4
	if armWidth == 3 {
		armOrigin = -7
	}
	geometry := map[string]any{
		"visible_bounds_width":  1,
		"visible_bounds_height": float64(bodyTop+8) / 16,
		"visible_bounds_offset": []float64{0, float64(bodyTop+8) / 32, 0},
		"texturewidth":          64,
		"textureheight":         64,
		"bones":                 []map[string]any{},
	}
	bones := geometry["bones"].([]map[string]any)
	addBone := func(name, parent string, pivot any, origin any, size any, uv any) {
		bone := map[string]any{"name": name, "pivot": pivot}
		if parent != "" {
			bone["parent"] = parent
		}
		if origin != nil {
			bone["cubes"] = []map[string]any{{"origin": origin, "size": size, "uv": uv}}
		}
		bones = append(bones, bone)
	}
	addBone("root", "", []int{0, 0, 0}, nil, nil, nil)
	addBone("waist", "root", []int{0, legHeight, 0}, nil, nil, nil)
	addBone("body", "waist", []int{0, bodyTop, 0}, []int{-4, legHeight, -2}, []int{8, bodyHeight, 4}, []int{16, 16})
	addBone("head", "body", []int{0, bodyTop, 0}, []int{-4, bodyTop, -4}, []int{8, 8, 8}, []int{0, 0})
	addBone("hat", "head", []int{0, bodyTop, 0}, []int{-4, bodyTop, -4}, []int{8, 8, 8}, []int{32, 0})
	addBone("rightArm", "body", []int{-5, bodyTop - 2, 0}, []int{armOrigin, legHeight, -2}, []int{armWidth, bodyHeight, 4}, []int{40, 16})
	addBone("leftArm", "body", []int{5, bodyTop - 2, 0}, []int{leftArmOrigin, legHeight, -2}, []int{armWidth, bodyHeight, 4}, []int{32, 48})
	addBone("rightLeg", "root", []float64{-1.9, float64(legHeight), 0}, []float64{-3.9, 0, -2}, []int{4, legHeight, 4}, []int{0, 16})
	addBone("leftLeg", "root", []float64{1.9, float64(legHeight), 0}, []float64{-0.1, 0, -2}, []int{4, legHeight, 4}, []int{16, 48})
	addBone("rightSleeve", "rightArm", []int{-5, bodyTop - 2, 0}, []int{armOrigin, legHeight, -2}, []int{armWidth, bodyHeight, 4}, []int{40, 32})
	addBone("leftSleeve", "leftArm", []int{5, bodyTop - 2, 0}, []int{leftArmOrigin, legHeight, -2}, []int{armWidth, bodyHeight, 4}, []int{48, 48})
	addBone("rightPants", "rightLeg", []float64{-1.9, float64(legHeight), 0}, []float64{-3.9, 0, -2}, []int{4, legHeight, 4}, []int{0, 32})
	addBone("leftPants", "leftLeg", []float64{1.9, float64(legHeight), 0}, []float64{-0.1, 0, -2}, []int{4, legHeight, 4}, []int{0, 48})
	addBone("jacket", "body", []int{0, bodyTop, 0}, []int{-4, legHeight, -2}, []int{8, bodyHeight, 4}, []int{16, 32})
	geometry["bones"] = bones
	return geometry
}

func Entries(packDir string) ([]Entry, error) {
	data, err := os.ReadFile(filepath.Join(packDir, "skins.json"))
	if err != nil {
		return nil, err
	}
	var raw skinManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(raw.Skins))
	for _, item := range raw.Skins {
		imageData, err := os.ReadFile(filepath.Join(packDir, filepath.FromSlash(item.Texture)))
		if err != nil {
			return nil, err
		}
		entries = append(entries, Entry{Name: item.LocalizationName, Geometry: item.Geometry, Height: heightForGeometry(item.Geometry), Texture: item.Texture, Data: base64.StdEncoding.EncodeToString(imageData)})
	}
	texts := map[string]string{}
	if textData, textErr := os.ReadFile(filepath.Join(packDir, "texts", "en_US.lang")); textErr == nil {
		for _, line := range strings.Split(string(textData), "\n") {
			parts := strings.SplitN(strings.TrimSpace(strings.TrimSuffix(line, "\r")), "=", 2)
			if len(parts) == 2 {
				texts[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	for i := range entries {
		key := fmt.Sprintf("skinpack.%s.%s", raw.SerializeName, raw.Skins[i].LocalizationName)
		if display := texts[key]; display != "" {
			entries[i].Name = display
		}
	}
	return entries, nil
}

func heightForGeometry(geometry string) string {
	switch geometry {
	case customShort, customSlimShort:
		return "short"
	case customTall, customSlimTall:
		return "tall"
	default:
		return "normal"
	}
}

func Update(packDir, packName string, inputs []SkinInput) error {
	entries, err := decodeInputs(inputs)
	if err != nil {
		return err
	}
	old, _ := Entries(packDir)
	for _, entry := range old {
		_ = os.Remove(filepath.Join(packDir, filepath.FromSlash(entry.Texture)))
	}
	return renderFiles(packDir, packName, entries, nil)
}
