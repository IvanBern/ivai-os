package main

import (
	"encoding/json"
	"fmt"

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
	var a struct{ Worker, Task string `json:"worker,instruction"` }
	json.Unmarshal([]byte(argsJSON), &a)
	return tools.HttpRequest("POST", "http://"+a.Worker+":8080/api/task",
		fmt.Sprintf(`{"instruction":%q}`, a.Task),
		map[string]string{"Content-Type": "application/json"})
}

func executeSwarmGather(argsJSON string) (string, error) {
	var a struct{ Worker string `json:"worker"` }
	json.Unmarshal([]byte(argsJSON), &a)
	return tools.HttpRequest("GET", "http://"+a.Worker+":8080/api/task-results?limit=5", "", nil)
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
	cmd := fmt.Sprintf("mkdir -p %s && cp /etc/ivai/.env %s/.env 2>/dev/null; IVAI_DATA_DIR=%s IVAI_PORT=%s setsid /usr/local/bin/ivai-os < /dev/null > /tmp/ivai-%s.log 2>&1 & sleep 3 && curl -s http://localhost:%s/api/status", dataDir, dataDir, dataDir, a.Port, a.Name, a.Port)
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
