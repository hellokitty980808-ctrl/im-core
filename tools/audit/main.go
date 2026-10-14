// audit 检查准备发布的文本文件是否含明显凭据；只输出路径和行号，不输出值。
package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`),
	regexp.MustCompile(`(?i)^\s*(?:password|token|secret_key|api_key|access_key)\s*:\s*[^\s$"']{8,}\s*$`),
}

func main() {
	findings := 0
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".tools", "bin", "logs":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".env") && entry.Name() != ".env.example" {
			return nil
		}
		if path == "tools/audit/main.go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		line := 0
		for scanner.Scan() {
			line++
			for index, pattern := range patterns {
				if index == 3 && filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
					continue
				}
				if pattern.MatchString(scanner.Text()) {
					fmt.Printf("需检查: %s:%d\n", path, line)
					findings++
					break
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if findings != 0 {
		os.Exit(1)
	}
	fmt.Println("发布文本检查通过：未发现规则覆盖的明显凭据；.env、工具和运行产物已排除")
}
