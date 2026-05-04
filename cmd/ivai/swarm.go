package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/IvanBern/ivai-os/internal/tools"
)

func callVMBridge(endpoint string, body map[string]string) (string, error) {
	jsonBody, _ := json.Marshal(body)
	return tools.HttpRequest("POST", "http://host.orb.internal:9877"+endpoint, string(jsonBody), map[string]string{"Content-Type": "application/json"})
}

func executeSwarmClone(argsJSON string) (string, error) {
	var a struct{ Name string `json:"name"` }
	json.Unmarshal([]byte(argsJSON), &a)
	return callVMBridge("/vm/clone", map[string]string{"name": a.Name})
}

func executeSwarmDeploy(argsJSON string) (string, error) {
	var a struct{ Name string `json:"name"` }
	json.Unmarshal([]byte(argsJSON), &a)
	return callVMBridge("/vm/deploy", map[string]string{"name": a.Name})
}

func executeSwarmDispatch(argsJSON string) (string, error) {
	var a struct {
		Worker string `json:"worker"`
		Task   string `json:"instruction"`
	}
	json.Unmarshal([]byte(argsJSON), &a)
	return tools.HttpRequest("POST", resolveWorkerURL(a.Worker, "/api/task"),
		fmt.Sprintf(`{"instruction":%q}`, a.Task),
		map[string]string{"Content-Type": "application/json"})
}

func executeSwarmGather(argsJSON string) (string, error) {
	var a struct{ Worker string `json:"worker"` }
	json.Unmarshal([]byte(argsJSON), &a)
	return tools.HttpRequest("GET", resolveWorkerURL(a.Worker, "/api/task-results?limit=5"), "", nil)
}

func executeSwarmStatus(argsJSON string) (string, error) {
	var a struct{ Name string `json:"name"` }
	json.Unmarshal([]byte(argsJSON), &a)
	if a.Name != "" {
		return callVMBridge("/vm/status", map[string]string{"name": a.Name})
	}
	return callVMBridge("/vm/list", nil)
}

func executeSwarmSpawn(argsJSON string) (string, error) {
	var a struct {
		Port string `json:"port"`
		Name string `json:"name"`
	}
	json.Unmarshal([]byte(argsJSON), &a)
	if a.Port == "" {
		a.Port = "8081"
	}
	dataDir := "/tmp/ivai-" + a.Name

	// Write .env with IVAI_PORT + API keys from parent process (0600 to avoid secret leaks)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("IVAI_PORT=%s\n", a.Port))
	for _, key := range []string{"DEEPSEEK_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY"} {
		if val := os.Getenv(key); val != "" {
			sb.WriteString(fmt.Sprintf("%s=%s\n", key, val))
		}
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dataDir, err)
	}
	if err := os.WriteFile(dataDir+"/.env", []byte(sb.String()), 0600); err != nil {
		return "", fmt.Errorf("write .env: %w", err)
	}

	cmd := fmt.Sprintf("IVAI_DATA_DIR=%s IVAI_PORT=%s setsid /usr/local/bin/ivai-os < /dev/null > /tmp/ivai-%s.log 2>&1 & sleep 3 && curl -s http://localhost:%s/api/status", dataDir, a.Port, a.Name, a.Port)
	out, err := tools.ExecuteCommand(cmd)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"worker":"localhost:%s","status":%s}`, a.Port, out), nil
}

func executeSwarmKill(argsJSON string) (string, error) {
	var a struct {
		Port string `json:"port"`
		Name string `json:"name"`
	}
	json.Unmarshal([]byte(argsJSON), &a)
	if a.Port != "" {
		tools.ExecuteCommand(fmt.Sprintf("fuser -k %s/tcp 2>/dev/null", a.Port))
		return fmt.Sprintf("Killed worker on port %s", a.Port), nil
	}
	if a.Name != "" {
		tools.ExecuteCommand(fmt.Sprintf("pkill -f 'ivai-%s' 2>/dev/null", a.Name))
		return fmt.Sprintf("Killed worker %s", a.Name), nil
	}
	return "No port or name specified", nil
}

// workerURL ensures a worker address has a port, defaulting to 8080.
// resolveWorkerURL builds an HTTP URL for a worker, defaulting to port 8080
// only when the worker string doesn't already include a port.
func resolveWorkerURL(worker, path string) string {
	if strings.Contains(worker, "://") {
		return worker + path
	}
	if strings.Contains(worker, ":") {
		return "http://" + worker + path
	}
	return "http://" + worker + ":8080" + path
}

// workerURL returns just the address with default port.
func workerURL(addr string) string {
	if strings.Contains(addr, ":") {
		return addr
	}
	return addr + ":8080"
}

// localWorkerPort reads the IVAI_PORT from a worker's .env file.
func localWorkerPort(name string) string {
	data, err := os.ReadFile("/tmp/ivai-" + name + "/.env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "IVAI_PORT=") {
			return strings.TrimPrefix(strings.TrimSpace(line), "IVAI_PORT=")
		}
	}
	return ""
}

// mergeWorkerLists combines VM and local worker lists into a single JSON response.
func mergeWorkerLists(vmResult, localResult string) string {
	var vmWorkers, localWorkers []any
	json.Unmarshal([]byte(vmResult), &vmWorkers)
	json.Unmarshal([]byte(localResult), &localWorkers)
	merged, _ := json.Marshal(map[string]any{
		"vm_workers":    vmWorkers,
		"local_workers": localWorkers,
	})
	return string(merged)
}

// checkLocalWorker checks if a local worker is running and returns its status.
func checkLocalWorker(name string) string {
	port := localWorkerPort(name)
	if port == "" {
		return ""
	}
	resp, err := tools.HttpRequest("GET", "http://localhost:"+port+"/api/status", "", nil)
	if err != nil {
		return ""
	}
	return resp
}

// listLocalWorkers scans /tmp for ivai- workers and returns their statuses.
func listLocalWorkers() string {
	entries, err := os.ReadDir("/tmp")
	if err != nil {
		return "[]"
	}
	var results []string
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "ivai-") {
			continue
		}
		name := strings.TrimPrefix(e.Name(), "ivai-")
		status := checkLocalWorker(name)
		if status == "" {
			continue
		}
		results = append(results, fmt.Sprintf(`{"name":%q,"type":"local","port":%q,"status":%s}`, name, localWorkerPort(name), status))
	}
	return "[" + strings.Join(results, ",") + "]"
}

// logFileInfo returns the size and last modified time of a worker's log file.
func logFileInfo(name string) (size int64, modTime string) {
	fi, err := os.Stat("/tmp/ivai-" + name + ".log")
	if err != nil {
		return 0, ""
	}
	return fi.Size(), fi.ModTime().Format("2006-01-02T15:04:05Z07:00")
}

// readWorkerLog returns the last N lines from a worker's log file.
func readWorkerLog(name string, lines int) string {
	f, err := os.Open("/tmp/ivai-" + name + ".log")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error reading log: %v", err)
	}

	if lines <= 0 || lines >= len(allLines) {
		return strings.Join(allLines, "\n")
	}
	return strings.Join(allLines[len(allLines)-lines:], "\n")
}

type workerInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Port      string `json:"port"`
	Status    any    `json:"status"`
	LogSize   int64  `json:"log_size"`
	LogMod    string `json:"log_modified"`
	UptimeSec int    `json:"uptime_sec"`
}

// collectWorkerData builds a workerInfo from a discovered worker name.
func collectWorkerData(name string) *workerInfo {
	port := localWorkerPort(name)
	statusRaw := checkLocalWorker(name)
	if statusRaw == "" {
		return nil
	}
	size, mod := logFileInfo(name)
	return &workerInfo{
		Name:      name,
		Type:      "local",
		Port:      port,
		Status:    parseStatus(statusRaw),
		LogSize:   size,
		LogMod:    mod,
		UptimeSec: extractUptimeSec(statusRaw),
	}
}

// parseStatus unmarshals a JSON status string into a map.
func parseStatus(raw string) any {
	var data map[string]any
	json.Unmarshal([]byte(raw), &data)
	return data
}

// extractUptimeSec reads the uptime_sec field from a raw status JSON string.
func extractUptimeSec(raw string) int {
	var data map[string]any
	if json.Unmarshal([]byte(raw), &data) != nil {
		return 0
	}
	u, ok := data["uptime_sec"]
	if !ok {
		return 0
	}
	fu, ok := u.(float64)
	if !ok {
		return 0
	}
	return int(fu)
}

// listWorkersWithMeta returns a JSON array of local workers with log metadata.
func listWorkersWithMeta() string {
	entries, err := os.ReadDir("/tmp")
	if err != nil {
		return "[]"
	}
	results := make([]workerInfo, 0)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "ivai-") {
			continue
		}
		w := collectWorkerData(strings.TrimPrefix(e.Name(), "ivai-"))
		if w == nil {
			continue
		}
		results = append(results, *w)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	data, _ := json.Marshal(results)
	return string(data)
}

// dispatchToWorker sends a task to a worker and returns the cleaned response.
func dispatchToWorker(worker, instruction string) (string, error) {
	return executeSwarmDispatch(fmt.Sprintf(`{"worker":%q,"instruction":%q}`, worker, instruction))
}
