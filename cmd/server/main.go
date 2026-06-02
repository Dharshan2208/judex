package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Dharshan2208/code-compiler/internal/executor"
	"github.com/Dharshan2208/code-compiler/internal/models"
	"github.com/Dharshan2208/code-compiler/internal/workspace"
)

func runHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RunRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", 400)
		return
	}

	dir, err := workspace.CreateWorkspace()
	if err != nil {
		http.Error(w, "workspace error", 500)
		return
	}

	defer workspace.Cleanup(dir)

	var (
		filename string
		execLang executor.Executor
	)

	switch req.Language {
	case "python":
		filename = "main.py"
		execLang = executor.PythonExecutor{}

	case "cpp":
		filename = "main.cpp"
		execLang = executor.CppExecutor{}

	default:
		http.Error(w, "unsupported language", 400)
		return
	}

	file, err := workspace.WriteFile(dir, filename, req.Code)
	if err != nil {
		http.Error(w, "file error", 500)
		return
	}

	result := execLang.Execute(file)

	resp := models.RunResponse{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Status:   result.Status,
		Language: req.Language,
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/run", runHandler)

	log.Println("server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
