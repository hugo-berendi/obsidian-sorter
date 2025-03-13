package main

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/teilomillet/gollm"
)

func getEnvVar(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("Error: %s environment variable is not set", name)
	}
	return value
}

func getExistingFolders(vaultPath string) ([]string, error) {
	var folders []string
	err := filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != vaultPath {
			relativePath, _ := filepath.Rel(vaultPath, path)
			folders = append(folders, relativePath)
		}
		return nil
	})
	return folders, err
}

func classifyFile(ctx context.Context, content string, folders []string, llm gollm.LLM) (string, error) {
	folderList := strings.Join(folders, ", ")
	promptText := fmt.Sprintf(`Classify the following Obsidian note into one of these existing folders: %s.
Return only the folder name (relative to the vault root).
Content:
%s`, folderList, content)

	prompt := gollm.NewPrompt(promptText)
	response, err := llm.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

func processFiles(vaultPath string, llm gollm.LLM) {
	ctx := context.Background()

	folders, err := getExistingFolders(vaultPath)
	if err != nil {
		log.Fatalf("Failed to read existing folders: %v", err)
	}

	files, err := ioutil.ReadDir(vaultPath)
	if err != nil {
		log.Fatalf("Failed to read vault directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(vaultPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Failed to read file %s: %v", filePath, err)
			continue
		}

		category, err := classifyFile(ctx, string(content), folders, llm)
		if err != nil {
			log.Printf("Classification error for %s: %v", file.Name(), err)
			continue
		}

		if category == "" {
			log.Printf("Skipping %s, no valid folder found.", file.Name())
			continue
		}

		destPath := filepath.Join(vaultPath, category, file.Name())

		err = os.Rename(filePath, destPath)
		if err != nil {
			log.Printf("Failed to move %s to %s: %v", file.Name(), destPath, err)
		} else {
			fmt.Printf("Moved %s to %s\n", file.Name(), category)
		}
	}
}

func main() {
	vaultPath := getEnvVar("VAULT_PATH")
	ollamaURL := getEnvVar("OLLAMA_URL")

	llm, err := gollm.NewLLM(
		gollm.SetProvider("ollama"),
		gollm.SetModel("mistral"),
		// gollm.SetDebugLevel(gollm.LogLevelDebug),
		gollm.SetOllamaEndpoint(ollamaURL),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	processFiles(vaultPath, llm)
}
