package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
)

func CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func RemoveDir(path string) error {
	return os.RemoveAll(path)
}

func DirExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func GetDirNames(path string) []string {
	dir, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer dir.Close()
	names, _ := dir.Readdirnames(-1)
	return names
}

func GetLastDirName(path string) string {
	return filepath.Base(path)
}

func GetAppDataPath() string {
	return apppath.AppData()
}

func GetMinecraftGDKDataPath(ispreview bool) string {
	return apppath.MinecraftData(ispreview)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ZipDir(srcDir, destZip string) error {
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(out)
	walkErr := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if info.IsDir() {
			return nil
		}
		// Skip symlinks and other non-regular files instead of
		// archiving the contents of whatever they point at:
		// filepath.Walk reports them via Lstat, but os.Open would
		// follow the link. Matches CopyDir's behaviour.
		if !info.Mode().IsRegular() {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		// Close per file, not via defer: a defer inside the walk
		// callback would only run when ZipDir returns, accumulating
		// one open handle per file in the tree.
		_, copyErr := io.Copy(writer, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if walkErr != nil {
		_ = zw.Close()
		_ = out.Close()
		return walkErr
	}
	// The writers' Close errors are the final flush (central
	// directory + file buffers): a failure here means the zip is
	// corrupt and must not be reported as success.
	if err := zw.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func DirSize(path string) int64 {
	var total int64
	root := filepath.Clean(strings.TrimSpace(path))
	if root == "" {
		return 0
	}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func SanitizeFilename(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "world"
	}
	invalid := "<>:\"/\\|?*"
	var b strings.Builder
	for _, r := range s {
		if r < 32 || strings.ContainsRune(invalid, r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	cleaned := strings.TrimRight(b.String(), " .")
	if cleaned == "" {
		return "world"
	}
	return cleaned
}

func CopyDir(src, dst string) error {
	src = filepath.Clean(strings.TrimSpace(src))
	dst = filepath.Clean(strings.TrimSpace(dst))
	if src == "" || dst == "" {
		return os.ErrInvalid
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		// Close both files per iteration, not via defer: defers
		// inside the walk callback would only run when CopyDir
		// returns, accumulating two open handles per copied file.
		_, copyErr := io.Copy(out, in)
		inCloseErr := in.Close()
		outCloseErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if inCloseErr != nil {
			return inCloseErr
		}
		return outCloseErr
	})
}

func JsonCompatBytes(data []byte) []byte {
	src := data
	n := len(src)
	out := make([]byte, 0, n)
	inString := false
	quote := byte(0)
	esc := false
	inLineComment := false
	inBlockComment := false
	for i := 0; i < n; i++ {
		b := src[i]
		if inLineComment {
			if b == '\n' || b == '\r' {
				inLineComment = false
				out = append(out, b)
			}
			continue
		}
		if inBlockComment {
			if b == '*' && i+1 < n && src[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inString {
			out = append(out, b)
			if esc {
				esc = false
				continue
			}
			if b == '\\' {
				esc = true
			} else if b == quote {
				inString = false
				quote = 0
			}
			continue
		}
		if b == '"' || b == '\'' {
			inString = true
			quote = b
			out = append(out, b)
			continue
		}
		if b == '/' && i+1 < n {
			nb := src[i+1]
			if nb == '/' {
				inLineComment = true
				i++
				continue
			}
			if nb == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		if b == ',' {
			j := i + 1
			for j < n {
				c := src[j]
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					j++
					continue
				}
				if c == '}' || c == ']' {
					goto next
				}
				break
			}
			out = append(out, b)
		next:
			continue
		}
		out = append(out, b)
	}
	return out
}
