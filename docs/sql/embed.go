package docsql

import (
	"embed"
	"io/fs"
	"path/filepath"
	"sort"
)

//go:embed chat/*.sql user/*.sql
var Files embed.FS

// MigrationFile 指定嵌入迁移的路径及用于记录校验和的文件名。
type MigrationFile struct {
	Path string
	Name string
}

// ListServiceFiles 按文件名顺序列出一个服务的 SQL 迁移。
func ListServiceFiles(service string) ([]MigrationFile, error) {
	dir := filepath.ToSlash(service)
	entries, err := fs.ReadDir(Files, dir)
	if err != nil {
		return nil, err
	}
	files := make([]MigrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, MigrationFile{
			Path: filepath.ToSlash(filepath.Join(dir, entry.Name())),
			Name: entry.Name(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files, nil
}

// ReadFile 读取编译进二进制的迁移内容，不依赖运行目录。
func ReadFile(path string) ([]byte, error) {
	return Files.ReadFile(path)
}
