package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	hexoRootDir  = `D:\Projects\Blog`
	hexoPostsDir = filepath.Join(hexoRootDir, "source", "_posts")
	classRepoDir = `D:\Projects\Class` // 这里配置你的 class 笔记仓库路径

	useSelect bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "clhe",
		Short: "Class to Hexo blog publishing helper",
	}

	var publishCmd = &cobra.Command{
		Use:   "publish <md file path> [tag]",
		Short: "Publish a markdown file from class repo to hexo blog",
		Args:  cobra.RangeArgs(1, 2), // 最少1个参数(md路径)，最多2个（加tag）
		RunE: func(cmd *cobra.Command, args []string) error {
			mdPath := args[0]
			var tag string
			if len(args) == 2 {
				tag = args[1]
			}

			if useSelect {
				var err error
				mdPath, err = selectMarkdownFile(classRepoDir)
				if err != nil {
					return err
				}
			}

			// 1. 检查文件存在
			if _, err := os.Stat(mdPath); os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", mdPath)
			}

			// 2. 确保 Front Matter
			if err := ensureFrontMatter(mdPath, tag); err != nil {
				return err
			}

			// 3. 拷贝到 hexo posts
			dstPath := filepath.Join(hexoPostsDir, filepath.Base(mdPath))
			if err := copyFile(mdPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file: %v", err)
			}
			fmt.Printf("✅ Copied %s -> %s\n", mdPath, dstPath)

			// 4. 生成 & 部署
			if err := runHexoCmd("g"); err != nil {
				return err
			}
			if err := runHexoCmd("d"); err != nil {
				return err
			}

			fmt.Println("🚀 Blog published successfully!")
			return nil
		},
	}

	publishCmd.Flags().BoolVarP(&useSelect, "select", "s", false, "Select file interactively from class repo")

	rootCmd.AddCommand(publishCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("❌ Error:", err)
		os.Exit(1)
	}
}

// ensureFrontMatter 确保 md 文件有 hexo front matter
func ensureFrontMatter(mdPath string, tag string) error {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	text := string(content)

	// 如果已经有 front matter (以 "---" 开头)，就不再插入
	if strings.HasPrefix(text, "---") {
		return nil
	}

	// 从文件名生成 title
	base := filepath.Base(mdPath)
	title := strings.TrimSuffix(base, filepath.Ext(base))

	// 格式化时间
	now := time.Now().Format("2006-01-02 15:04:05")

	// 构建 front matter
	var front string
	if tag != "" {
		front = fmt.Sprintf(`---
title: %s
date: %s
tags: [%s]
---

`, title, now, tag)
	} else {
		front = fmt.Sprintf(`---
title: %s
date: %s
tags:
---

`, title, now)
	}

	newContent := front + text

	// 写回文件（覆盖原文）
	if err := os.WriteFile(mdPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

// 拷贝文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

// 运行 hexo 命令
func runHexoCmd(arg string) error {
	cmd := exec.Command("hexo", arg)
	cmd.Dir = hexoRootDir // 设定 hexo 项目根目录
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("👉 Running: hexo %s\n", arg)
	return cmd.Run()
}

// 交互式文件选择
func selectMarkdownFile(root string) (string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no markdown files found in %s", root)
	}

	fmt.Println("请选择要发布的 Markdown 文件：")
	for i, f := range files {
		fmt.Printf("[%d] %s\n", i+1, f)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("输入序号：")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var idx int
		_, err := fmt.Sscanf(input, "%d", &idx)
		if err != nil || idx <= 0 || idx > len(files) {
			fmt.Println("❌ 输入无效，请重新输入")
			continue
		}
		return files[idx-1], nil
	}
}
