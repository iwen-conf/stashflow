package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iwen-conf/stashflow/internal/stashflow"
)

func main() {
	var dryRun bool
	var noBackup bool
	var applyRules bool
	var fixAll bool

	flag.BoolVar(&dryRun, "dry-run", false, "预览将要修改的内容，不写入文件")
	flag.BoolVar(&noBackup, "no-backup", false, "写入前不创建 .bak 备份")
	flag.BoolVar(&applyRules, "apply-stash-rules", false, "重新应用内置 Stash 分流模板")
	flag.BoolVar(&fixAll, "fix-all", false, "清理异常 UUID 并重新应用 Stash 分流模板")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "用法: %s [选项] <订阅.yaml> [更多文件...]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "清理 Stash/Clash/Mihomo 订阅中的异常 UUID 节点，并可补回内置分流规则。")
		fmt.Fprintln(flag.CommandLine.Output(), "\n选项:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	status := 0
	for _, path := range flag.Args() {
		if err := processFile(path, applyRules || fixAll, !noBackup, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			status = 1
		}
	}
	os.Exit(status)
}

func processFile(path string, applySplit bool, backup bool, dryRun bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("不是文件")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := stashflow.FixText(string(data), applySplit)
	if !result.Changed {
		if applySplit {
			fmt.Printf("%s: 未发现异常 UUID，分流模板已应用\n", path)
		} else {
			fmt.Printf("%s: 未发现异常 UUID\n", path)
		}
		return nil
	}

	if len(result.Clean.Removals) > 0 {
		fmt.Printf("%s: 删除 %d 个异常 UUID 节点\n", path, len(result.Clean.Removals))
		for _, removal := range result.Clean.Removals {
			fmt.Printf("  第 %d 行: %s (uuid: %s)\n", removal.Line, removal.Name, removal.UUID)
		}
		fmt.Printf("  删除 %d 个策略组引用\n", result.Clean.ReferenceCount)
	} else {
		fmt.Printf("%s: 未发现异常 UUID\n", path)
	}

	if applySplit {
		if result.Split.Changed {
			fmt.Printf("  已应用 Stash 分流模板（%d 个分组，%d 条规则）\n", result.Split.GroupCount, result.Split.RuleCount)
		} else {
			fmt.Println("  Stash 分流模板已应用")
		}
	}

	if dryRun {
		fmt.Println("  预览模式：文件未修改")
		return nil
	}

	if backup {
		backupPath := stashflow.NextBackupPath(path)
		if err := copyFile(path, backupPath); err != nil {
			return err
		}
		fmt.Printf("  备份: %s\n", backupPath)
	}

	if err := os.WriteFile(path, []byte(result.Text), info.Mode().Perm()); err != nil {
		return err
	}
	fmt.Println("  已更新")
	return nil
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(src)
	perm := os.FileMode(0o644)
	if statErr == nil {
		perm = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, perm)
}
