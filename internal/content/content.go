package content

import (
	"archive/zip"
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/nbt"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/packages"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/types"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/utils"
)

// Zip-bomb limits for the pack/world importers below, mirroring the
// limits internal/uwp already enforces for package extraction: a cap
// on the number of archive entries, a cumulative budget for the
// decompressed bytes an import may write, and a small cap for entries
// read fully into memory (manifests, nested packs). Declared entry
// sizes are checked up front, and every copy/read is additionally
// limited while it runs, so an entry that decompresses past its
// declared size still trips the budget instead of filling the disk
// or memory.
const (
	maxImportZipEntries    = 100000
	maxImportZipBytes      = int64(20) << 30
	maxImportManifestBytes = int64(16) << 20
)

var errImportArchiveTooLarge = errors.New("archive exceeds the decompressed size limit")

// checkImportZipLimits fails fast on the archive's declared shape:
// too many entries, or a total declared decompressed size over the
// import budget. It returns the import error code to propagate, or
// "" when the archive is within limits.
func checkImportZipLimits(zr *zip.Reader) string {
	if len(zr.File) > maxImportZipEntries {
		return "ERR_ARCHIVE_TOO_MANY_ENTRIES"
	}
	var declared int64
	for _, f := range zr.File {
		if f.UncompressedSize64 > uint64(maxImportZipBytes-declared) {
			return "ERR_ARCHIVE_TOO_LARGE"
		}
		declared += int64(f.UncompressedSize64)
	}
	return ""
}

// readImportZipEntry reads one archive entry into memory under a hard
// limit. It returns errImportArchiveTooLarge when the decompressed
// entry is larger than limit.
func readImportZipEntry(rc io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errImportArchiveTooLarge
	}
	return b, nil
}

// copyImportZipEntry copies one decompressed entry to out, charging
// the bytes written to the shared budget *remaining (which the caller
// keeps across the whole import). It returns "" on success, or the
// import error code to propagate: "ERR_ARCHIVE_TOO_LARGE" when the
// budget is exhausted, "ERR_WRITE_FILE" for any other copy failure.
func copyImportZipEntry(out io.Writer, rc io.Reader, remaining *int64) string {
	limit := *remaining
	if limit < 0 {
		limit = 0
	}
	n, err := io.Copy(out, io.LimitReader(rc, limit+1))
	*remaining -= n
	if err != nil {
		return "ERR_WRITE_FILE"
	}
	if *remaining < 0 {
		return "ERR_ARCHIVE_TOO_LARGE"
	}
	return ""
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int8:
		return int64(t)
	case uint8:
		return int64(t)
	case int16:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case float32:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int8:
		return float64(t)
	case uint8:
		return float64(t)
	case int16:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}

func coerceNumberTo(old any, v any) any {
	switch old.(type) {
	case int8:
		return int8(toInt64(v))
	case uint8:
		return uint8(toInt64(v))
	case int16:
		return int16(toInt64(v))
	case int32:
		return int32(toInt64(v))
	case int64:
		return toInt64(v)
	case float32:
		return float32(toFloat64(v))
	case float64:
		return toFloat64(v)
	default:
		return v
	}
}

func coerceSliceToElemType(arr []any, oldElemKind reflect.Kind) any {
	switch oldElemKind {
	case reflect.Int8:
		tmp := make([]int8, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, int8(toInt64(e)))
		}
		return tmp
	case reflect.Uint8:
		tmp := make([]uint8, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, uint8(toInt64(e)))
		}
		return tmp
	case reflect.Int16:
		tmp := make([]int16, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, int16(toInt64(e)))
		}
		return tmp
	case reflect.Int32:
		tmp := make([]int32, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, int32(toInt64(e)))
		}
		return tmp
	case reflect.Int64:
		tmp := make([]int64, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, toInt64(e))
		}
		return tmp
	case reflect.Float32:
		tmp := make([]float32, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, float32(toFloat64(e)))
		}
		return tmp
	case reflect.Float64:
		tmp := make([]float64, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, toFloat64(e))
		}
		return tmp
	case reflect.String:
		tmp := make([]string, 0, len(arr))
		for _, e := range arr {
			tmp = append(tmp, fmt.Sprintf("%v", e))
		}
		return tmp
	default:
		return arr
	}
}

func coerceArrayFromSlice(arr []any, elemKind reflect.Kind) any {
	n := len(arr)
	switch elemKind {
	case reflect.Int32:
		a := reflect.New(reflect.ArrayOf(n, reflect.TypeOf(int32(0)))).Elem()
		for i := 0; i < n; i++ {
			a.Index(i).SetInt(int64(int32(toInt64(arr[i]))))
		}
		return a.Interface()
	case reflect.Int64:
		a := reflect.New(reflect.ArrayOf(n, reflect.TypeOf(int64(0)))).Elem()
		for i := 0; i < n; i++ {
			a.Index(i).SetInt(toInt64(arr[i]))
		}
		return a.Interface()
	case reflect.Uint8:
		a := reflect.New(reflect.ArrayOf(n, reflect.TypeOf(uint8(0)))).Elem()
		for i := 0; i < n; i++ {
			a.Index(i).SetUint(uint64(uint8(toInt64(arr[i]))))
		}
		return a.Interface()
	default:
		return arr
	}
}

func coerceMapTo(old map[string]any, m map[string]any) map[string]any {
	if old == nil || m == nil {
		return m
	}
	for k, v := range m {
		if ov, ok := old[k]; ok && ov != nil {
			switch v.(type) {
			case float64, float32, int, int8, uint8, int16, int32, int64, string:
				m[k] = coerceNumberTo(ov, v)
			case []any:
				ovv := reflect.ValueOf(ov)
				if ovv.Kind() == reflect.Array {
					m[k] = coerceArrayFromSlice(v.([]any), ovv.Type().Elem().Kind())
				} else if ovv.Kind() == reflect.Slice {
					m[k] = coerceSliceToElemType(v.([]any), ovv.Type().Elem().Kind())
				}
			case map[string]any:
				if om, ok2 := ov.(map[string]any); ok2 {
					m[k] = coerceMapTo(om, v.(map[string]any))
				}
			}
		}
	}
	return m
}

// importStaging is one staged import target: archive entries are
// extracted into stage (a sibling temp dir of final) and only moved
// into place by commit after extraction has fully succeeded, so a
// corrupt or hostile archive can no longer destroy — or half-write
// over — the pack that is already installed (M-12).
type importStaging struct {
	stage string
	final string
}

func stageImportTarget(final string) (importStaging, string) {
	parent := filepath.Dir(final)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return importStaging{}, "ERR_CREATE_TARGET_DIR"
	}
	stage, err := os.MkdirTemp(parent, ".bnx-import-*")
	if err != nil {
		return importStaging{}, "ERR_CREATE_TARGET_DIR"
	}
	return importStaging{stage: stage, final: final}, ""
}

func discardImportStaging(list []importStaging) {
	for _, s := range list {
		_ = utils.RemoveDir(s.stage)
	}
}

// commit replaces final with the staged directory.
func (s importStaging) commit() string {
	if utils.DirExists(s.final) {
		if err := utils.RemoveDir(s.final); err != nil {
			return "ERR_WRITE_FILE"
		}
	}
	if err := os.Rename(s.stage, s.final); err != nil {
		return "ERR_WRITE_FILE"
	}
	return ""
}

// extractZipIntoDir extracts the entries of zr that live under
// prefixDir ("" = archive root) into root, charging the shared
// extraction budget. It is the single extraction loop the importers
// share; its containment checks and error codes are the ones the
// per-importer loops previously duplicated.
func extractZipIntoDir(zr *zip.Reader, prefixDir string, root string, extractRemaining *int64) string {
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		var relInDir string
		if prefixDir != "" {
			if nameInZip != prefixDir && !strings.HasPrefix(nameInZip, prefixDir+"/") {
				continue
			}
			relInDir = strings.TrimPrefix(strings.TrimPrefix(nameInZip, prefixDir), "/")
		} else {
			relInDir = nameInZip
		}
		target := root
		if relInDir != "" && relInDir != "/" {
			target = filepath.Join(root, relInDir)
		}
		safeTarget, _ := filepath.Abs(target)
		safeRoot, _ := filepath.Abs(root)
		if !strings.HasPrefix(strings.ToLower(safeTarget), strings.ToLower(safeRoot+string(os.PathSeparator))) && strings.ToLower(safeTarget) != strings.ToLower(safeRoot) {
			continue
		}
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(target, 0755); err != nil {
				return "ERR_CREATE_TARGET_DIR"
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "ERR_CREATE_TARGET_DIR"
		}
		rc, err := f.Open()
		if err != nil {
			return "ERR_READ_ZIP_ENTRY"
		}
		out, er := os.Create(target)
		if er != nil {
			rc.Close()
			return "ERR_WRITE_FILE"
		}
		if code := copyImportZipEntry(out, rc, extractRemaining); code != "" {
			out.Close()
			rc.Close()
			return code
		}
		out.Close()
		rc.Close()
	}
	return ""
}

// importPackTargets imports the entries of zr under prefixDir into
// every directory in finals, atomically per the staging pattern:
// duplicate-folder checks run before anything is written, extraction
// happens entirely in staging dirs, and only then are the packs in
// deleteFirst removed (strictDelete=false tolerates their removal
// errors, matching the mcaddon importers) and the staging dirs renamed
// into place. On any error the staging dirs are removed and
// pre-existing packs are left untouched.
func importPackTargets(zr *zip.Reader, prefixDir string, finals []string, overwrite bool, deleteFirst []string, strictDelete bool, extractRemaining *int64) string {
	for _, final := range finals {
		if utils.DirExists(final) && !overwrite {
			return "ERR_DUPLICATE_FOLDER"
		}
	}
	staged := make([]importStaging, 0, len(finals))
	for _, final := range finals {
		st, code := stageImportTarget(final)
		if code != "" {
			discardImportStaging(staged)
			return code
		}
		staged = append(staged, st)
	}
	for _, st := range staged {
		if code := extractZipIntoDir(zr, prefixDir, st.stage, extractRemaining); code != "" {
			discardImportStaging(staged)
			return code
		}
	}
	for _, p := range deleteFirst {
		if err := utils.RemoveDir(p); err != nil && strictDelete {
			discardImportStaging(staged)
			return "ERR_WRITE_FILE"
		}
	}
	for _, st := range staged {
		if code := st.commit(); code != "" {
			discardImportStaging(staged)
			return code
		}
	}
	return ""
}

func findManifestDir(dir string) string {
	d := strings.TrimSpace(dir)
	if d == "" {
		return ""
	}
	if utils.FileExists(filepath.Join(d, "manifest.json")) {
		return d
	}
	fi, err := os.Stat(d)
	if err != nil || !fi.IsDir() {
		return ""
	}
	queue := []string{d}
	seen := map[string]struct{}{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, ok := seen[cur]; ok {
			continue
		}
		seen[cur] = struct{}{}
		ents, er := os.ReadDir(cur)
		if er != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(cur, e.Name())
			if utils.FileExists(filepath.Join(sub, "manifest.json")) {
				return sub
			}
			queue = append(queue, sub)
		}
	}
	return ""
}

func findPackPathsByUuid(uuid string, dirs ...string) []string {
	if uuid == "" {
		return nil
	}
	var paths []string
	for _, root := range dirs {
		if strings.TrimSpace(root) == "" || !utils.DirExists(root) {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(root, e.Name())
			mfPath := filepath.Join(p, "manifest.json")
			if !utils.FileExists(mfPath) {
				continue
			}
			b, err := os.ReadFile(mfPath)
			if err != nil {
				continue
			}
			var mf packages.RawManifest
			if err := json.Unmarshal(utils.JsonCompatBytes(b), &mf); err != nil {
				continue
			}
			if mf.Header.UUID == uuid {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func generateRandomPackName() string {
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func ImportMcpackToDirs(data []byte, archiveName string, resDir string, bpDir string, overwrite bool) string {
	if len(data) == 0 || (strings.TrimSpace(resDir) == "" && strings.TrimSpace(bpDir) == "") {
		return "ERR_OPEN_ZIP"
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "ERR_OPEN_ZIP"
	}
	if code := checkImportZipLimits(zr); code != "" {
		return code
	}
	extractRemaining := maxImportZipBytes
	manifestDir := ""
	var manifest packages.RawManifest
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.HasSuffix(nameInZip, "/") {
			continue
		}
		if strings.EqualFold(path.Base(nameInZip), "manifest.json") {
			dir := path.Dir(nameInZip)
			rc, er := f.Open()
			if er == nil {
				b, _ := readImportZipEntry(rc, maxImportManifestBytes)
				_ = rc.Close()
				_ = json.Unmarshal(utils.JsonCompatBytes(b), &manifest)
			}
			if dir != "." && strings.TrimSpace(dir) != "" {
				manifestDir = dir
			}
			break
		}
	}
	if manifestDir == "" && len(manifest.Modules) == 0 {
		return "ERR_MANIFEST_NOT_FOUND"
	}

	existing := findPackPathsByUuid(manifest.Header.UUID, resDir, bpDir)
	if len(existing) > 0 && !overwrite {
		return "ERR_DUPLICATE_UUID"
	}
	// The existing packs are removed only at the commit step of the
	// staged import below, once the replacement is fully extracted.

	baseName := generateRandomPackName()
	baseName = utils.SanitizeFilename(baseName)
	if strings.TrimSpace(baseName) == "" {
		baseName = "pack"
	}
	targets := make([]string, 0, 2)
	hasRes := false
	hasData := false
	for _, m := range manifest.Modules {
		tp := strings.ToLower(strings.TrimSpace(m.Type))
		if tp == "resources" && strings.TrimSpace(resDir) != "" {
			targets = append(targets, filepath.Join(resDir, baseName))
			hasRes = true
		} else if (tp == "data" || tp == "script") && strings.TrimSpace(bpDir) != "" {
			targets = append(targets, filepath.Join(bpDir, baseName))
			hasData = true
		}
	}
	if !hasRes && !hasData {
		return "ERR_INVALID_PACKAGE"
	}
	return importPackTargets(zr, manifestDir, targets, overwrite, existing, true, &extractRemaining)
}

func stripKnownArchiveExt(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	base := filepath.Base(s)
	lower := strings.ToLower(base)
	known := []string{".mcpack", ".mcworld", ".mcaddon", ".zip"}
	for _, ext := range known {
		if strings.HasSuffix(lower, ext) && len(base) > len(ext) {
			return base[:len(base)-len(ext)]
		}
	}
	return base
}

func normalizeZipEntryName(name string) string {
	n := strings.TrimSpace(name)
	n = strings.TrimPrefix(n, "./")
	n = strings.ReplaceAll(n, "\\", "/")
	n = strings.TrimPrefix(n, "/")
	for strings.Contains(n, "//") {
		n = strings.ReplaceAll(n, "//", "/")
	}
	return n
}

func ImportMcpackToDirs2(data []byte, archiveName string, resDir string, bpDir string, skinDir string, overwrite bool) string {
	if len(data) == 0 || (strings.TrimSpace(resDir) == "" && strings.TrimSpace(bpDir) == "" && strings.TrimSpace(skinDir) == "") {
		return "ERR_OPEN_ZIP"
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "ERR_OPEN_ZIP"
	}
	if code := checkImportZipLimits(zr); code != "" {
		return code
	}
	extractRemaining := maxImportZipBytes
	manifestDir := ""
	var manifest packages.RawManifest
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.HasSuffix(nameInZip, "/") {
			continue
		}
		if strings.EqualFold(path.Base(nameInZip), "manifest.json") {
			dir := path.Dir(nameInZip)
			rc, er := f.Open()
			if er == nil {
				b, _ := readImportZipEntry(rc, maxImportManifestBytes)
				_ = rc.Close()
				_ = json.Unmarshal(utils.JsonCompatBytes(b), &manifest)
			}
			if dir != "." && strings.TrimSpace(dir) != "" {
				manifestDir = dir
			}
			break
		}
	}
	if manifestDir == "" && len(manifest.Modules) == 0 {
		return "ERR_MANIFEST_NOT_FOUND"
	}
	existing := findPackPathsByUuid(manifest.Header.UUID, resDir, bpDir, skinDir)
	if len(existing) > 0 && !overwrite {
		return "ERR_DUPLICATE_UUID"
	}
	// The existing packs are removed only at the commit step of the
	// staged import below, once the replacement is fully extracted.

	baseName := generateRandomPackName()
	baseName = utils.SanitizeFilename(baseName)
	if strings.TrimSpace(baseName) == "" {
		baseName = "pack"
	}
	targetsMap := make(map[string]bool)
	hasAny := false
	hadSkin := false
	for _, m := range manifest.Modules {
		tp := strings.ToLower(strings.TrimSpace(m.Type))
		if tp == "resources" && strings.TrimSpace(resDir) != "" {
			targetsMap[filepath.Join(resDir, baseName)] = true
			hasAny = true
		} else if (tp == "data" || tp == "script") && strings.TrimSpace(bpDir) != "" {
			targetsMap[filepath.Join(bpDir, baseName)] = true
			hasAny = true
		} else if tp == "skin_pack" {
			hadSkin = true
			if strings.TrimSpace(skinDir) != "" {
				targetsMap[filepath.Join(skinDir, baseName)] = true
				hasAny = true
			}
		}
	}
	if !hasAny {
		if hadSkin && strings.TrimSpace(skinDir) == "" {
			return "ERR_NO_PLAYER"
		}
		return "ERR_INVALID_PACKAGE"
	}
	targets := make([]string, 0, len(targetsMap))
	for t := range targetsMap {
		targets = append(targets, t)
	}
	return importPackTargets(zr, manifestDir, targets, overwrite, existing, true, &extractRemaining)
}

func ImportMcaddonToDirs2(data []byte, resDir string, bpDir string, skinDir string, overwrite bool) string {
	if len(data) == 0 {
		return "ERR_OPEN_ZIP"
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "ERR_OPEN_ZIP"
	}
	if code := checkImportZipLimits(zr); code != "" {
		return code
	}
	extractRemaining := maxImportZipBytes
	imported := false
	hadSkin := false
	for _, f := range zr.File {
		name := strings.TrimSpace(f.Name)
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".mcpack") {
			rc, er := f.Open()
			if er != nil {
				return "ERR_OPEN_ZIP"
			}
			b, readErr := readImportZipEntry(rc, extractRemaining)
			_ = rc.Close()
			if errors.Is(readErr, errImportArchiveTooLarge) {
				return "ERR_ARCHIVE_TOO_LARGE"
			}
			if readErr != nil {
				return "ERR_OPEN_ZIP"
			}
			extractRemaining -= int64(len(b))
			base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			err2 := ImportMcpackToDirs2(b, base, resDir, bpDir, skinDir, overwrite)
			if err2 != "" {
				return err2
			}
			imported = true
		}
	}

	type packInfo struct {
		dir      string
		manifest packages.RawManifest
	}
	packs := []packInfo{}
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.HasSuffix(nameInZip, "/") {
			continue
		}
		if strings.EqualFold(path.Base(nameInZip), "manifest.json") {
			dir := path.Dir(nameInZip)
			if dir == "." || strings.TrimSpace(dir) == "" {
				continue
			}
			rc, er := f.Open()
			if er != nil {
				continue
			}
			b, _ := readImportZipEntry(rc, maxImportManifestBytes)
			_ = rc.Close()
			var mf packages.RawManifest
			_ = json.Unmarshal(utils.JsonCompatBytes(b), &mf)
			packs = append(packs, packInfo{dir: dir, manifest: mf})
		}
	}
	for _, p := range packs {
		existing := findPackPathsByUuid(p.manifest.Header.UUID, resDir, bpDir, skinDir)
		if len(existing) > 0 && !overwrite {
			return "ERR_DUPLICATE_UUID"
		}
		// Existing packs are removed at the staged import's commit
		// step below, not before the replacement is validated.

		baseName := generateRandomPackName()
		baseName = utils.SanitizeFilename(baseName)
		if strings.TrimSpace(baseName) == "" {
			baseName = "pack"
		}
		targetsMap := make(map[string]bool)
		hasAny := false
		packHadSkin := false
		for _, m := range p.manifest.Modules {
			tp := strings.ToLower(strings.TrimSpace(m.Type))
			if tp == "resources" && strings.TrimSpace(resDir) != "" {
				targetsMap[filepath.Join(resDir, baseName)] = true
				hasAny = true
			} else if (tp == "data" || tp == "script") && strings.TrimSpace(bpDir) != "" {
				targetsMap[filepath.Join(bpDir, baseName)] = true
				hasAny = true
			} else if tp == "skin_pack" {
				packHadSkin = true
				if strings.TrimSpace(skinDir) != "" {
					targetsMap[filepath.Join(skinDir, baseName)] = true
					hasAny = true
				}
			}
		}
		if !hasAny {
			if packHadSkin {
				hadSkin = true
			}
			continue
		}
		targets := make([]string, 0, len(targetsMap))
		for t := range targetsMap {
			targets = append(targets, t)
		}
		if code := importPackTargets(zr, p.dir, targets, overwrite, existing, false, &extractRemaining); code != "" {
			return code
		}
		imported = true
	}
	if !imported {
		if hadSkin && strings.TrimSpace(skinDir) == "" {
			return "ERR_NO_PLAYER"
		}
		return "ERR_INVALID_PACKAGE"
	}
	return ""
}

func ImportMcaddonToDirs(data []byte, resDir string, bpDir string, overwrite bool) string {
	if len(data) == 0 {
		return "ERR_OPEN_ZIP"
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "ERR_OPEN_ZIP"
	}
	if code := checkImportZipLimits(zr); code != "" {
		return code
	}
	extractRemaining := maxImportZipBytes
	imported := false
	for _, f := range zr.File {
		name := strings.TrimSpace(f.Name)
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".mcpack") {
			rc, er := f.Open()
			if er != nil {
				return "ERR_OPEN_ZIP"
			}
			b, readErr := readImportZipEntry(rc, extractRemaining)
			_ = rc.Close()
			if errors.Is(readErr, errImportArchiveTooLarge) {
				return "ERR_ARCHIVE_TOO_LARGE"
			}
			if readErr != nil {
				return "ERR_OPEN_ZIP"
			}
			extractRemaining -= int64(len(b))
			base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			err2 := ImportMcpackToDirs(b, base, resDir, bpDir, overwrite)
			if err2 != "" {
				return err2
			}
			imported = true
		}
	}

	type packInfo struct {
		dir      string
		manifest packages.RawManifest
	}
	packs := []packInfo{}
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.HasSuffix(nameInZip, "/") {
			continue
		}
		if strings.EqualFold(path.Base(nameInZip), "manifest.json") {
			dir := path.Dir(nameInZip)
			if dir == "." || strings.TrimSpace(dir) == "" {
				continue
			}
			rc, er := f.Open()
			if er != nil {
				continue
			}
			b, _ := readImportZipEntry(rc, maxImportManifestBytes)
			_ = rc.Close()
			var mf packages.RawManifest
			_ = json.Unmarshal(utils.JsonCompatBytes(b), &mf)
			packs = append(packs, packInfo{dir: dir, manifest: mf})
		}
	}
	for _, p := range packs {
		baseName := utils.SanitizeFilename(path.Base(p.dir))
		if strings.TrimSpace(baseName) == "" || baseName == "." || baseName == string(os.PathSeparator) {
			baseName = "pack"
		}
		targets := make([]string, 0, 2)
		hasRes := false
		hasData := false
		for _, m := range p.manifest.Modules {
			tp := strings.ToLower(strings.TrimSpace(m.Type))
			if tp == "resources" && strings.TrimSpace(resDir) != "" {
				targets = append(targets, filepath.Join(resDir, baseName))
				hasRes = true
			} else if (tp == "data" || tp == "script") && strings.TrimSpace(bpDir) != "" {
				targets = append(targets, filepath.Join(bpDir, baseName))
				hasData = true
			}
		}
		if !hasRes && !hasData {
			continue
		}
		if code := importPackTargets(zr, p.dir, targets, overwrite, nil, false, &extractRemaining); code != "" {
			return code
		}
		imported = true
	}
	if !imported {
		return "ERR_INVALID_PACKAGE"
	}
	return ""
}

func ImportMcworldToDir(data []byte, archiveName string, worldsDir string, overwrite bool) string {
	if len(data) == 0 || strings.TrimSpace(worldsDir) == "" {
		return "ERR_OPEN_ZIP"
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "ERR_OPEN_ZIP"
	}
	if code := checkImportZipLimits(zr); code != "" {
		return code
	}
	extractRemaining := maxImportZipBytes
	levelDir := ""
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.HasSuffix(nameInZip, "/") {
			continue
		}
		if strings.EqualFold(path.Base(nameInZip), "level.dat") {
			d := path.Dir(nameInZip)
			if d != "." && strings.TrimSpace(d) != "" {
				levelDir = d
			}
			break
		}
	}
	baseName := strings.TrimSpace(archiveName)
	if baseName != "" {
		baseName = stripKnownArchiveExt(baseName)
	}
	baseName = strings.ReplaceAll(baseName, ".", "_")
	baseName = utils.SanitizeFilename(baseName)
	if strings.TrimSpace(baseName) == "" || baseName == "." || baseName == string(os.PathSeparator) {
		baseName = "world"
	}
	randomDir := generateRandomPackName()
	if strings.TrimSpace(randomDir) == "" {
		randomDir = "world"
	}
	targetRoot := filepath.Join(worldsDir, randomDir)
	staged, code := stageImportTarget(targetRoot)
	if code != "" {
		return code
	}
	if code := extractZipIntoDir(zr, levelDir, staged.stage, &extractRemaining); code != "" {
		discardImportStaging([]importStaging{staged})
		return code
	}
	return staged.commit()
}

func ReadPackInfoFromDir(dir string) types.PackInfo {
	var info types.PackInfo
	d := strings.TrimSpace(dir)
	if d == "" {
		return info
	}
	target := findManifestDir(d)
	if strings.TrimSpace(target) == "" {
		target = d
	}
	manifestPath := filepath.Join(target, "manifest.json")
	if utils.FileExists(manifestPath) {
		if b, err := os.ReadFile(manifestPath); err == nil {
			var mf packages.RawManifest
			_ = json.Unmarshal(utils.JsonCompatBytes(b), &mf)
			info.Name = strings.TrimSpace(mf.Header.Name)
			info.Description = strings.TrimSpace(mf.Header.Description)
			if (info.Name == "pack.name" || info.Description == "pack.description") && utils.DirExists(filepath.Join(target, "texts")) {
				texts := packages.ReadPackTexts(target)
				if texts != nil {
					if info.Name == "pack.name" {
						if v, ok := texts["pack.name"]; ok {
							info.Name = strings.TrimSpace(v)
						}
					}
					if info.Description == "pack.description" {
						if v, ok := texts["pack.description"]; ok {
							info.Description = strings.TrimSpace(v)
						}
					}
				}
			}

			version := packages.ParseVersion(mf.Header.Version)
			if len(version) > 0 {
				var vb strings.Builder
				for i, n := range version {
					if i > 0 {
						vb.WriteRune('.')
					}
					vb.WriteString(fmt.Sprintf("%d", n))
				}
				info.Version = vb.String()
			}

			minEngineVersion := packages.ParseVersion(mf.Header.MinEngineVersion)
			if len(minEngineVersion) > 0 {
				var vb2 strings.Builder
				for i, n := range minEngineVersion {
					if i > 0 {
						vb2.WriteRune('.')
					}
					vb2.WriteString(fmt.Sprintf("%d", n))
				}
				info.MinEngineVersion = vb2.String()
			}
		}
	}
	iconPath := filepath.Join(target, "pack_icon.png")
	if utils.FileExists(iconPath) {
		if b, err := os.ReadFile(iconPath); err == nil {
			enc := base64.StdEncoding.EncodeToString(b)
			info.IconDataUrl = "data:image/png;base64," + enc
		}
	}
	info.Path = target
	return info
}

func GetLevelDat(worldDir string) ([]byte, error) {
	if strings.TrimSpace(worldDir) == "" {
		return nil, os.ErrNotExist
	}
	p := filepath.Join(worldDir, "level.dat")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func PutLevelDat(worldDir string, levelDatData []byte) error {
	if strings.TrimSpace(worldDir) == "" {
		return os.ErrNotExist
	}
	p := filepath.Join(worldDir, "level.dat")
	backup := filepath.Join(worldDir, "level.dat_bedrocknexus")
	legacyBackup := filepath.Join(worldDir, "level.dat_leviold")
	if !utils.FileExists(backup) && !utils.FileExists(legacyBackup) {
		if b, err := os.ReadFile(p); err == nil {
			_ = os.WriteFile(backup, b, 0644)
		}
	}
	return os.WriteFile(p, levelDatData, 0644)
}

func GetLevelDatNbtAndVersion(worldDir string) ([]byte, int32, error) {
	b, err := GetLevelDat(worldDir)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < 8 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	var version int32
	var dataSize int32
	buf := bytes.NewBuffer(b[:4])
	if err = binary.Read(buf, binary.LittleEndian, &version); err != nil {
		return nil, 0, err
	}
	buf2 := bytes.NewBuffer(b[4:8])
	if err = binary.Read(buf2, binary.LittleEndian, &dataSize); err != nil {
		return nil, 0, err
	}
	nbtData := b[8:]
	return nbtData, version, nil
}

func PutLevelDatNbtAndVersion(worldDir string, levelDatData []byte, version int32) error {
	var header []byte
	nbtLen := int32(len(levelDatData))
	buf := bytes.NewBuffer(header)
	if err := binary.Write(buf, binary.LittleEndian, version); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, nbtLen); err != nil {
		return err
	}
	return PutLevelDat(worldDir, append(buf.Bytes(), levelDatData...))
}

func DecodeLevelDat(worldDir string) (map[string]any, int32, error) {
	nbtData, version, err := GetLevelDatNbtAndVersion(worldDir)
	if err != nil {
		return nil, 0, err
	}
	var root map[string]any
	if err = nbt.UnmarshalEncoding(nbtData, &root, nbt.LittleEndian); err != nil {
		return nil, 0, err
	}
	return root, version, nil
}

func EncodeLevelDat(worldDir string, version int32, root map[string]any) error {
	data, err := nbt.MarshalEncoding(root, nbt.LittleEndian)
	if err != nil {
		return err
	}
	return PutLevelDatNbtAndVersion(worldDir, data, version)
}

func DumpLevelDat(worldDir string) (string, error) {
	b, err := GetLevelDat(worldDir)
	if err != nil {
		return "", err
	}
	if len(b) < 8 {
		return "", io.ErrUnexpectedEOF
	}
	nbtData := b[8:]
	return nbt.Dump(nbtData, nbt.LittleEndian)
}

func ReadLevelDatFields(worldDir string) ([]types.LevelDatField, int32, error) {
	root, ver, err := DecodeLevelDat(worldDir)
	if err != nil {
		return nil, 0, err
	}
	var data map[string]any
	data = root
	inData := false
	if v, ok := root["Data"].(map[string]any); ok {
		data = v
		inData = true
	}
	out := make([]types.LevelDatField, 0, len(data))
	for k, v := range data {
		f := types.LevelDatField{Name: k, InData: inData, Path: []string{}}
		switch tv := v.(type) {
		case int8:
			f.Tag = "byte"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case uint8:
			f.Tag = "byte"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case int16:
			f.Tag = "short"
			f.ValueString = fmt.Sprintf("%d", tv)
		case int32:
			f.Tag = "int"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case int64:
			f.Tag = "long"
			f.ValueString = fmt.Sprintf("%d", tv)
		case float32:
			f.Tag = "float"
			f.ValueString = fmt.Sprintf("%g", tv)
		case float64:
			f.Tag = "double"
			f.ValueString = fmt.Sprintf("%g", tv)
		case string:
			f.Tag = "string"
			f.ValueString = tv
		case []any:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []int32:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []int64:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []float32:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []float64:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []string:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case map[string]any:
			f.Tag = "compound"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		default:
			rv := reflect.ValueOf(v)
			if rv.IsValid() && rv.Kind() == reflect.Array {
				ek := rv.Type().Elem().Kind()
				switch ek {
				case reflect.Int32:
					f.Tag = "list"
					s := make([]int32, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = int32(rv.Index(i).Int())
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				case reflect.Int64:
					f.Tag = "list"
					s := make([]int64, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = rv.Index(i).Int()
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				case reflect.Uint8:
					f.Tag = "list"
					s := make([]uint8, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = uint8(rv.Index(i).Uint())
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				default:
					b, _ := json.Marshal(tv)
					f.Tag = "unknown"
					if len(b) > 0 {
						f.ValueJSON = string(b)
					} else {
						f.ValueString = fmt.Sprintf("%v", tv)
					}
				}
			} else {
				b, _ := json.Marshal(tv)
				f.Tag = "unknown"
				if len(b) > 0 {
					f.ValueJSON = string(b)
				} else {
					f.ValueString = fmt.Sprintf("%v", tv)
				}
			}
		}
		out = append(out, f)
	}
	return out, ver, nil
}

func ReadLevelDatOrder(worldDir string) ([]string, int32, error) {
	b, err := GetLevelDat(worldDir)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < 8 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	version := int32(binary.LittleEndian.Uint32(b[:4]))
	nbtData := b[8:]
	keys, err2 := nbt.KeysInOrder(nbtData, nbt.LittleEndian, []string{"Data"})
	if err2 != nil || len(keys) == 0 {
		keys, _ = nbt.KeysInOrder(nbtData, nbt.LittleEndian, nil)
	}
	return keys, version, nil
}

func WriteLevelDatFields(worldDir string, fields []types.LevelDatField, version int32) error {
	root, _, err := DecodeLevelDat(worldDir)
	if err != nil {
		root = map[string]any{}
	}
	var data map[string]any
	data = root
	if v, ok := root["Data"].(map[string]any); ok {
		data = v
	}
	byName := map[string]types.LevelDatField{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	for k, f := range byName {
		switch f.Tag {
		case "byte":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			data[k] = uint8(n)
		case "short":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			data[k] = int16(n)
		case "int":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			data[k] = int32(n)
		case "long":
			n, _ := strconv.ParseInt(strings.TrimSpace(f.ValueString), 10, 64)
			data[k] = n
		case "float":
			f32, _ := strconv.ParseFloat(strings.TrimSpace(f.ValueString), 32)
			data[k] = float32(f32)
		case "double":
			f64, _ := strconv.ParseFloat(strings.TrimSpace(f.ValueString), 64)
			data[k] = f64
		case "string":
			data[k] = f.ValueString
		case "list":
			var arr []any
			vj := strings.TrimSpace(f.ValueJSON)
			if vj == "" {
				arr = []any{}
			} else {
				_ = json.Unmarshal([]byte(vj), &arr)
				if arr == nil {
					arr = []any{}
				}
			}
			old := data[k]
			var out any
			if old != nil {
				ov := reflect.ValueOf(old)
				if ov.IsValid() && ov.Kind() == reflect.Array {
					out = coerceArrayFromSlice(arr, ov.Type().Elem().Kind())
				} else if ov.IsValid() && ov.Kind() == reflect.Slice {
					out = coerceSliceToElemType(arr, ov.Type().Elem().Kind())
				} else {
					if oa, ok := old.([]any); ok && len(oa) > 0 {
						switch oa[0].(type) {
						case int8:
							tmp := make([]int8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int8(toInt64(e)))
							}
							out = tmp
						case uint8:
							tmp := make([]uint8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, uint8(toInt64(e)))
							}
							out = tmp
						case int16:
							tmp := make([]int16, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int16(toInt64(e)))
							}
							out = tmp
						case int32:
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						case int64:
							tmp := make([]int64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toInt64(e))
							}
							out = tmp
						case float32:
							tmp := make([]float32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, float32(toFloat64(e)))
							}
							out = tmp
						case float64:
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						case string:
							tmp := make([]string, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, fmt.Sprintf("%v", e))
							}
							out = tmp
						case map[string]any:
							tmp := make([]map[string]any, 0, len(arr))
							for _, e := range arr {
								if m, ok := e.(map[string]any); ok {
									tmp = append(tmp, m)
								}
							}
							out = tmp
						default:
							out = arr
						}
					} else {
						switch old.(type) {
						case []int8:
							tmp := make([]int8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int8(toInt64(e)))
							}
							out = tmp
						case []uint8:
							tmp := make([]uint8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, uint8(toInt64(e)))
							}
							out = tmp
						case []int16:
							tmp := make([]int16, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int16(toInt64(e)))
							}
							out = tmp
						case []int32:
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						case []int64:
							tmp := make([]int64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toInt64(e))
							}
							out = tmp
						case []float32:
							tmp := make([]float32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, float32(toFloat64(e)))
							}
							out = tmp
						case []float64:
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						case []string:
							tmp := make([]string, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, fmt.Sprintf("%v", e))
							}
							out = tmp
						case []map[string]any:
							tmp := make([]map[string]any, 0, len(arr))
							for _, e := range arr {
								if m, ok := e.(map[string]any); ok {
									tmp = append(tmp, m)
								}
							}
							out = tmp
						default:
							out = arr
						}
					}
				}
			}
			if out == nil {
				if len(arr) == 0 {
					out = []string{}
				} else {
					allStr := true
					allMap := true
					allNum := true
					for _, e := range arr {
						switch e.(type) {
						case string:
						default:
							allStr = false
						}
						if _, ok := e.(map[string]any); !ok {
							allMap = false
						}
						switch e.(type) {
						case float64:
						default:
							allNum = false
						}
					}
					if allStr {
						tmp := make([]string, 0, len(arr))
						for _, e := range arr {
							tmp = append(tmp, fmt.Sprintf("%v", e))
						}
						out = tmp
					} else if allMap {
						tmp := make([]map[string]any, 0, len(arr))
						for _, e := range arr {
							if m, ok := e.(map[string]any); ok {
								tmp = append(tmp, m)
							}
						}
						out = tmp
					} else if allNum {
						ints := true
						for _, e := range arr {
							v := toFloat64(e)
							if math.Trunc(v) != v {
								ints = false
								break
							}
						}
						if ints {
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						} else {
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						}
					} else {
						tmp := make([]string, 0, len(arr))
						for _, e := range arr {
							tmp = append(tmp, fmt.Sprintf("%v", e))
						}
						out = tmp
					}
				}
			}
			data[k] = out
		case "compound":
			var obj map[string]any
			vj := strings.TrimSpace(f.ValueJSON)
			if vj == "" {
				obj = map[string]any{}
			} else {
				_ = json.Unmarshal([]byte(vj), &obj)
				if obj == nil {
					obj = map[string]any{}
				}
			}
			if ov, ok := data[k].(map[string]any); ok {
				data[k] = coerceMapTo(ov, obj)
			} else {
				data[k] = obj
			}
		default:
			if strings.TrimSpace(f.ValueJSON) != "" {
				var anyv any
				_ = json.Unmarshal([]byte(f.ValueJSON), &anyv)
				data[k] = anyv
			} else {
				data[k] = f.ValueString
			}
		}
	}
	if _, ok := root["Data"].(map[string]any); ok {
		root["Data"] = data
	} else {
		root = data
	}
	return EncodeLevelDat(worldDir, version, root)
}

func IsMcpackSkinPack(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	for _, f := range zr.File {
		nameInZip := normalizeZipEntryName(f.Name)
		if strings.EqualFold(path.Base(nameInZip), "manifest.json") {
			rc, er := f.Open()
			if er != nil {
				continue
			}
			b, _ := readImportZipEntry(rc, maxImportManifestBytes)
			_ = rc.Close()
			var mf packages.RawManifest
			_ = json.Unmarshal(utils.JsonCompatBytes(b), &mf)
			for _, m := range mf.Modules {
				tp := strings.ToLower(strings.TrimSpace(m.Type))
				if tp == "skin_pack" {
					return true
				}
			}
		}
	}
	return false
}

func ReadLevelDatFieldsAt(worldDir string, path []string) ([]types.LevelDatField, int32, error) {
	root, ver, err := DecodeLevelDat(worldDir)
	if err != nil {
		return nil, 0, err
	}
	var data map[string]any
	data = root
	inData := false
	if v, ok := root["Data"].(map[string]any); ok {
		data = v
		inData = true
	}
	cur := data
	for _, seg := range path {
		if m, ok := cur[seg].(map[string]any); ok {
			cur = m
		} else {
			return nil, ver, fmt.Errorf("path not found: %s", seg)
		}
	}
	out := make([]types.LevelDatField, 0, len(cur))
	for k, v := range cur {
		f := types.LevelDatField{Name: k, InData: inData, Path: append([]string{}, path...)}
		switch tv := v.(type) {
		case int8:
			f.Tag = "byte"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case uint8:
			f.Tag = "byte"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case int16:
			f.Tag = "short"
			f.ValueString = fmt.Sprintf("%d", tv)
		case int32:
			f.Tag = "int"
			f.ValueString = fmt.Sprintf("%d", tv)
			f.IsBoolLike = tv == 0 || tv == 1
		case int64:
			f.Tag = "long"
			f.ValueString = fmt.Sprintf("%d", tv)
		case float32:
			f.Tag = "float"
			f.ValueString = fmt.Sprintf("%g", tv)
		case float64:
			f.Tag = "double"
			f.ValueString = fmt.Sprintf("%g", tv)
		case string:
			f.Tag = "string"
			f.ValueString = tv
		case []any:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []int32:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []int64:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []float32:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []float64:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case []string:
			f.Tag = "list"
			b, _ := json.Marshal(tv)
			f.ValueJSON = string(b)
		case map[string]any:
			f.Tag = "compound"
			f.ValueJSON = ""
		default:
			rv := reflect.ValueOf(v)
			if rv.IsValid() && rv.Kind() == reflect.Array {
				ek := rv.Type().Elem().Kind()
				switch ek {
				case reflect.Int32:
					f.Tag = "list"
					s := make([]int32, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = int32(rv.Index(i).Int())
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				case reflect.Int64:
					f.Tag = "list"
					s := make([]int64, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = rv.Index(i).Int()
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				case reflect.Uint8:
					f.Tag = "list"
					s := make([]uint8, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						s[i] = uint8(rv.Index(i).Uint())
					}
					b, _ := json.Marshal(s)
					f.ValueJSON = string(b)
				default:
					b, _ := json.Marshal(tv)
					f.Tag = "unknown"
					if len(b) > 0 {
						f.ValueJSON = string(b)
					} else {
						f.ValueString = fmt.Sprintf("%v", tv)
					}
				}
			} else {
				b, _ := json.Marshal(tv)
				f.Tag = "unknown"
				if len(b) > 0 {
					f.ValueJSON = string(b)
				} else {
					f.ValueString = fmt.Sprintf("%v", tv)
				}
			}
		}
		out = append(out, f)
	}
	return out, ver, nil
}

func ReadLevelDatOrderAt(worldDir string, path []string) ([]string, int32, error) {
	b, err := GetLevelDat(worldDir)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < 8 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	version := int32(binary.LittleEndian.Uint32(b[:4]))
	nbtData := b[8:]
	keys, err2 := nbt.KeysInOrder(nbtData, nbt.LittleEndian, path)
	if err2 != nil {
		return nil, version, err2
	}
	return keys, version, nil
}

func WriteLevelDatFieldsAt(worldDir string, path []string, fields []types.LevelDatField, version int32) error {
	root, _, err := DecodeLevelDat(worldDir)
	if err != nil {
		root = map[string]any{}
	}
	var data map[string]any
	data = root
	if v, ok := root["Data"].(map[string]any); ok {
		data = v
	}
	cur := data
	for _, seg := range path {
		if m, ok := cur[seg].(map[string]any); ok {
			cur = m
		} else {
			nm := map[string]any{}
			cur[seg] = nm
			cur = nm
		}
	}
	byName := map[string]types.LevelDatField{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	for k, f := range byName {
		switch f.Tag {
		case "byte":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			cur[k] = uint8(n)
		case "short":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			cur[k] = int16(n)
		case "int":
			n, _ := strconv.Atoi(strings.TrimSpace(f.ValueString))
			cur[k] = int32(n)
		case "long":
			n, _ := strconv.ParseInt(strings.TrimSpace(f.ValueString), 10, 64)
			cur[k] = n
		case "float":
			f32, _ := strconv.ParseFloat(strings.TrimSpace(f.ValueString), 32)
			cur[k] = float32(f32)
		case "double":
			f64, _ := strconv.ParseFloat(strings.TrimSpace(f.ValueString), 64)
			cur[k] = f64
		case "string":
			cur[k] = f.ValueString
		case "list":
			var arr []any
			vj := strings.TrimSpace(f.ValueJSON)
			if vj == "" {
				arr = []any{}
			} else {
				_ = json.Unmarshal([]byte(vj), &arr)
				if arr == nil {
					arr = []any{}
				}
			}
			old := cur[k]
			var out any
			if old != nil {
				ov := reflect.ValueOf(old)
				if ov.IsValid() && ov.Kind() == reflect.Array {
					out = coerceArrayFromSlice(arr, ov.Type().Elem().Kind())
				} else if ov.IsValid() && ov.Kind() == reflect.Slice {
					out = coerceSliceToElemType(arr, ov.Type().Elem().Kind())
				} else {
					if oa, ok := old.([]any); ok && len(oa) > 0 {
						switch oa[0].(type) {
						case int8:
							tmp := make([]int8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int8(toInt64(e)))
							}
							out = tmp
						case uint8:
							tmp := make([]uint8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, uint8(toInt64(e)))
							}
							out = tmp
						case int16:
							tmp := make([]int16, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int16(toInt64(e)))
							}
							out = tmp
						case int32:
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						case int64:
							tmp := make([]int64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toInt64(e))
							}
							out = tmp
						case float32:
							tmp := make([]float32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, float32(toFloat64(e)))
							}
							out = tmp
						case float64:
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						case string:
							tmp := make([]string, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, fmt.Sprintf("%v", e))
							}
							out = tmp
						case map[string]any:
							tmp := make([]map[string]any, 0, len(arr))
							for _, e := range arr {
								if m, ok := e.(map[string]any); ok {
									tmp = append(tmp, m)
								}
							}
							out = tmp
						default:
							out = arr
						}
					} else {
						switch old.(type) {
						case []int8:
							tmp := make([]int8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int8(toInt64(e)))
							}
							out = tmp
						case []uint8:
							tmp := make([]uint8, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, uint8(toInt64(e)))
							}
							out = tmp
						case []int16:
							tmp := make([]int16, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int16(toInt64(e)))
							}
							out = tmp
						case []int32:
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						case []int64:
							tmp := make([]int64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toInt64(e))
							}
							out = tmp
						case []float32:
							tmp := make([]float32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, float32(toFloat64(e)))
							}
							out = tmp
						case []float64:
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						case []string:
							tmp := make([]string, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, fmt.Sprintf("%v", e))
							}
							out = tmp
						case []map[string]any:
							tmp := make([]map[string]any, 0, len(arr))
							for _, e := range arr {
								if m, ok := e.(map[string]any); ok {
									tmp = append(tmp, m)
								}
							}
							out = tmp
						default:
							out = arr
						}
					}
				}
			}
			if out == nil {
				if len(arr) == 0 {
					out = []string{}
				} else {
					allStr := true
					allMap := true
					allNum := true
					for _, e := range arr {
						switch e.(type) {
						case string:
						default:
							allStr = false
						}
						if _, ok := e.(map[string]any); !ok {
							allMap = false
						}
						switch e.(type) {
						case float64:
						default:
							allNum = false
						}
					}
					if allStr {
						tmp := make([]string, 0, len(arr))
						for _, e := range arr {
							tmp = append(tmp, fmt.Sprintf("%v", e))
						}
						out = tmp
					} else if allMap {
						tmp := make([]map[string]any, 0, len(arr))
						for _, e := range arr {
							if m, ok := e.(map[string]any); ok {
								tmp = append(tmp, m)
							}
						}
						out = tmp
					} else if allNum {
						ints := true
						for _, e := range arr {
							v := toFloat64(e)
							if math.Trunc(v) != v {
								ints = false
								break
							}
						}
						if ints {
							tmp := make([]int32, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, int32(toInt64(e)))
							}
							out = tmp
						} else {
							tmp := make([]float64, 0, len(arr))
							for _, e := range arr {
								tmp = append(tmp, toFloat64(e))
							}
							out = tmp
						}
					} else {
						tmp := make([]string, 0, len(arr))
						for _, e := range arr {
							tmp = append(tmp, fmt.Sprintf("%v", e))
						}
						out = tmp
					}
				}
			}
			cur[k] = out
		case "compound":
			var obj map[string]any
			vj := strings.TrimSpace(f.ValueJSON)
			if vj == "" {
				obj = map[string]any{}
			} else {
				_ = json.Unmarshal([]byte(vj), &obj)
				if obj == nil {
					obj = map[string]any{}
				}
			}
			if ov, ok := cur[k].(map[string]any); ok {
				cur[k] = coerceMapTo(ov, obj)
			} else {
				cur[k] = obj
			}
		default:
			if strings.TrimSpace(f.ValueJSON) != "" {
				var anyv any
				_ = json.Unmarshal([]byte(f.ValueJSON), &anyv)
				cur[k] = anyv
			} else {
				cur[k] = f.ValueString
			}
		}
	}
	if _, ok := root["Data"].(map[string]any); ok {
		root["Data"] = data
	} else {
		root = data
	}
	return EncodeLevelDat(worldDir, version, root)
}
