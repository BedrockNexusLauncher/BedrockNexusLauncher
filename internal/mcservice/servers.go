package mcservice

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/types"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/utils"
)

func ListServers(versionName string, player string) ([]types.Server, error) {
	roots := GetContentRoots(versionName)
	if roots.UsersRoot == "" || player == "" {
		return []types.Server{}, nil
	}

	serverFile := filepath.Join(roots.UsersRoot, player, "games", "com.mojang", "minecraftpe", "external_servers.txt")
	if !utils.FileExists(serverFile) {
		return []types.Server{}, nil
	}

	file, err := os.Open(serverFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []types.Server
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		// Format: Index:ServerName:IP:Port:Timestamp
		// Example: 1:12312:127.0.0.1:19132:1768910786
		if len(parts) >= 5 {
			ts, _ := strconv.ParseInt(parts[4], 10, 64)
			servers = append(servers, types.Server{
				Index:     parts[0],
				Name:      parts[1],
				IP:        parts[2],
				Port:      parts[3],
				Timestamp: ts,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

// serverFilePath resolves external_servers.txt for a player. The game itself
// owns this file (Play → Servers → Add Server); we only read/append it.
func serverFilePath(versionName string, player string) (string, string) {
	roots := GetContentRoots(versionName)
	if roots.UsersRoot == "" || strings.TrimSpace(player) == "" {
		return "", "ERR_NO_GAME_DATA"
	}
	return filepath.Join(roots.UsersRoot, strings.TrimSpace(player), "games", "com.mojang", "minecraftpe", "external_servers.txt"), ""
}

func validateServerName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" || strings.ContainsAny(n, ":\r\n") {
		return "ERR_SERVER_INVALID_NAME"
	}
	return ""
}

func validateServerAddress(address string) string {
	a := strings.TrimSpace(address)
	if a == "" || strings.ContainsAny(a, " \t\r\n:/\\") {
		return "ERR_SERVER_INVALID_ADDRESS"
	}
	return ""
}

func validateServerPort(port string) string {
	p, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || p < 1 || p > 65535 {
		return "ERR_SERVER_INVALID_PORT"
	}
	return ""
}

func readServerFileLines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return []string{}
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	// Server names are short; still allow long MOTD-ish lines.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func writeServerFileLines(path string, lines []string) string {
	if len(lines) == 0 {
		// No servers left: remove the file so the game (and ListServers)
		// treat it as an empty list instead of a blank file.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return "ERR_WRITE_TARGET"
		}
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "ERR_WRITE_TARGET"
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return "ERR_WRITE_TARGET"
	}
	return ""
}

// AddServer appends a server entry for the player. Returns "" on success,
// otherwise an ERR_* code for the frontend to translate.
func AddServer(versionName string, player string, name string, address string, port string) string {
	if code := validateServerName(name); code != "" {
		return code
	}
	if code := validateServerAddress(address); code != "" {
		return code
	}
	if code := validateServerPort(port); code != "" {
		return code
	}
	path, code := serverFilePath(versionName, player)
	if code != "" {
		return code
	}
	lines := readServerFileLines(path)
	next := 1
	for _, line := range lines {
		if parts := strings.Split(line, ":"); len(parts) >= 5 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && n >= next {
				next = n + 1
			}
		}
	}
	lines = append(lines, strings.Join([]string{
		strconv.Itoa(next),
		strings.TrimSpace(name),
		strings.TrimSpace(address),
		strings.TrimSpace(port),
		strconv.FormatInt(time.Now().Unix(), 10),
	}, ":"))
	return writeServerFileLines(path, lines)
}

// UpdateServer rewrites the entry with the given index, keeping its original
// added-timestamp. Returns "" on success, otherwise an ERR_* code.
func UpdateServer(versionName string, player string, index string, name string, address string, port string) string {
	if code := validateServerName(name); code != "" {
		return code
	}
	if code := validateServerAddress(address); code != "" {
		return code
	}
	if code := validateServerPort(port); code != "" {
		return code
	}
	path, code := serverFilePath(versionName, player)
	if code != "" {
		return code
	}
	lines := readServerFileLines(path)
	found := false
	for i, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 5 && strings.TrimSpace(parts[0]) == strings.TrimSpace(index) {
			lines[i] = strings.Join([]string{
				strings.TrimSpace(parts[0]),
				strings.TrimSpace(name),
				strings.TrimSpace(address),
				strings.TrimSpace(port),
				strings.TrimSpace(parts[4]),
			}, ":")
			found = true
			break
		}
	}
	if !found {
		return "ERR_SERVER_NOT_FOUND"
	}
	return writeServerFileLines(path, lines)
}

// DeleteServer removes the entry with the given index. Returns "" on success,
// otherwise an ERR_* code.
func DeleteServer(versionName string, player string, index string) string {
	path, code := serverFilePath(versionName, player)
	if code != "" {
		return code
	}
	lines := readServerFileLines(path)
	kept := make([]string, 0, len(lines))
	found := false
	for _, line := range lines {
		if parts := strings.Split(line, ":"); len(parts) >= 5 && strings.TrimSpace(parts[0]) == strings.TrimSpace(index) {
			found = true
			continue
		}
		kept = append(kept, line)
	}
	if !found {
		return "ERR_SERVER_NOT_FOUND"
	}
	return writeServerFileLines(path, kept)
}
