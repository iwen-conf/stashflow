package stashflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	topLevelKey      = regexp.MustCompile(`^[^\s#-][^:]*:\s*`)
	flowProxyPattern = regexp.MustCompile(`^(?P<prefix>\s*proxies:\s*)\[(?P<body>.*)\](?P<suffix>\s*(?:#.*)?)$`)
)

type Removal struct {
	Name string
	Line int
	UUID string
}

type CleanResult struct {
	Text           string
	Removals       []Removal
	ReferenceCount int
}

type SplitResult struct {
	Text       string
	Changed    bool
	GroupCount int
	RuleCount  int
}

type FixResult struct {
	Text        string
	Clean       CleanResult
	Split       SplitResult
	Changed     bool
	BackupPath  string
	BackupMade  bool
	OriginalLen int
}

func CleanText(text string) CleanResult {
	trailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}

	lines, removals := removeBadProxyItems(lines)
	names := make(map[string]bool, len(removals))
	for _, removal := range removals {
		names[removal.Name] = true
	}
	lines, refs := removeReferences(lines, names)

	cleaned := strings.Join(lines, "\n")
	if trailingNewline {
		cleaned += "\n"
	}

	return CleanResult{Text: cleaned, Removals: removals, ReferenceCount: refs}
}

func ApplyStashSplitRules(text string) SplitResult {
	trailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}

	names := make(map[string]bool, len(StashSplitGroupNames))
	for _, name := range StashSplitGroupNames {
		names[name] = true
	}

	lines = removeProxyGroupsByName(lines, names)
	lines = insertStashSplitGroups(lines)
	lines = replaceRules(lines)

	updated := strings.Join(lines, "\n")
	if trailingNewline {
		updated += "\n"
	}

	return SplitResult{
		Text:       updated,
		Changed:    updated != text,
		GroupCount: len(StashSplitGroupNames),
		RuleCount:  len(StashRuleLines),
	}
}

func FixText(text string, applySplit bool) FixResult {
	clean := CleanText(text)
	finalText := clean.Text
	split := SplitResult{Text: finalText}
	if applySplit {
		split = ApplyStashSplitRules(finalText)
		finalText = split.Text
	}

	return FixResult{
		Text:        finalText,
		Clean:       clean,
		Split:       split,
		Changed:     finalText != text,
		OriginalLen: len(text),
	}
}

func FixFile(path string, applySplit bool, backup bool) (FixResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FixResult{}, err
	}

	result := FixText(string(data), applySplit)
	if !result.Changed {
		return result, nil
	}

	if backup {
		backupPath := NextBackupPath(path)
		if err := copyFile(path, backupPath); err != nil {
			return FixResult{}, err
		}
		result.BackupPath = backupPath
		result.BackupMade = true
	}

	info, statErr := os.Stat(path)
	perm := os.FileMode(0o644)
	if statErr == nil {
		perm = info.Mode().Perm()
	}
	if err := os.WriteFile(path, []byte(result.Text), perm); err != nil {
		return FixResult{}, err
	}

	return result, nil
}

func AnalyzeFile(path string) (FixResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FixResult{}, err
	}
	return FixText(string(data), true), nil
}

func NextBackupPath(path string) string {
	candidate := path + ".bak"
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}

	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s.bak%d", path, i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func DiscoverFiles(args []string) []string {
	if len(args) > 0 {
		paths := make([]string, 0, len(args))
		for _, arg := range args {
			paths = append(paths, expandHome(arg))
		}
		return paths
	}

	seen := map[string]bool{}
	var paths []string
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() && !seen[match] {
				seen[match] = true
				paths = append(paths, match)
			}
		}
	}
	return paths
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func sectionBounds(lines []string, sectionName string) (int, int, bool) {
	sectionPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(sectionName) + `:\s*(?:#.*)?$`)
	for start, line := range lines {
		if sectionPattern.MatchString(line) {
			end := len(lines)
			for i := start + 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" && topLevelKey.MatchString(lines[i]) {
					end = i
					break
				}
			}
			return start, end, true
		}
	}
	return 0, 0, false
}

func itemBounds(lines []string, itemStart int, sectionEnd int) (int, int) {
	indent := leadingSpaces(lines[itemStart])
	end := sectionEnd
	prefix := strings.Repeat(" ", indent) + "- "
	for i := itemStart + 1; i < sectionEnd; i++ {
		if strings.HasPrefix(lines[i], prefix) && leadingSpaces(lines[i]) == indent {
			end = i
			break
		}
	}
	return itemStart, end
}

func parseInlineValue(text string, key string) (string, bool) {
	pattern := regexp.MustCompile(`(?:^|[{\s,])` + regexp.QuoteMeta(key) + `:\s*([^,}\n]+)`)
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", false
	}
	return unquote(strings.TrimSpace(match[1])), true
}

func parseBlockValue(lines []string, key string) (string, bool) {
	pattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `:\s*(.+?)\s*(?:#.*)?$`)
	for _, line := range lines {
		match := pattern.FindStringSubmatch(line)
		if len(match) >= 2 {
			return unquote(strings.TrimSpace(match[1])), true
		}
	}
	return "", false
}

func unquote(value string) string {
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if first == last && (first == '\'' || first == '"') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func proxyName(itemLines []string) (string, bool) {
	text := strings.Join(itemLines, "\n")
	if value, ok := parseInlineValue(text, "name"); ok {
		return value, true
	}
	return parseBlockValue(itemLines, "name")
}

func proxyUUID(itemLines []string) (string, bool) {
	text := strings.Join(itemLines, "\n")
	if value, ok := parseInlineValue(text, "uuid"); ok {
		return value, true
	}
	return parseBlockValue(itemLines, "uuid")
}

func removeBadProxyItems(lines []string) ([]string, []Removal) {
	sectionStart, sectionEnd, ok := sectionBounds(lines, "proxies")
	if !ok {
		return lines, nil
	}

	var removals []Removal
	var ranges [][2]int

	for i := sectionStart + 1; i < sectionEnd; {
		if strings.HasPrefix(strings.TrimLeft(lines[i], " "), "- ") {
			start, end := itemBounds(lines, i, sectionEnd)
			item := lines[start:end]
			if uuid, ok := proxyUUID(item); ok && !uuidPattern.MatchString(uuid) {
				name, ok := proxyName(item)
				if !ok {
					name = fmt.Sprintf("<unnamed:%d>", start+1)
				}
				removals = append(removals, Removal{Name: name, Line: start + 1, UUID: uuid})
				ranges = append(ranges, [2]int{start, end})
			}
			i = end
			continue
		}
		i++
	}

	if len(ranges) == 0 {
		return lines, removals
	}

	keep := make([]bool, len(lines))
	for i := range keep {
		keep[i] = true
	}
	for _, r := range ranges {
		for i := r[0]; i < r[1]; i++ {
			keep[i] = false
		}
	}

	filtered := make([]string, 0, len(lines))
	for i, line := range lines {
		if keep[i] {
			filtered = append(filtered, line)
		}
	}
	return filtered, removals
}

func removeReferences(lines []string, names map[string]bool) ([]string, int) {
	if len(names) == 0 {
		return lines, 0
	}

	removed := 0
	output := make([]string, 0, len(lines))
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "- ") {
			value := unquote(strings.TrimSpace(strings.TrimPrefix(stripped, "- ")))
			if names[value] {
				removed++
				continue
			}
		}

		if match := flowProxyPattern.FindStringSubmatch(line); len(match) > 0 {
			prefix := match[flowProxyPattern.SubexpIndex("prefix")]
			body := match[flowProxyPattern.SubexpIndex("body")]
			suffix := match[flowProxyPattern.SubexpIndex("suffix")]
			var kept []string
			for _, item := range strings.Split(body, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				if names[unquote(item)] {
					removed++
					continue
				}
				kept = append(kept, item)
			}
			line = prefix + "[" + strings.Join(kept, ", ") + "]" + suffix
		}
		output = append(output, line)
	}

	return output, removed
}

func removeProxyGroupsByName(lines []string, names map[string]bool) []string {
	sectionStart, sectionEnd, ok := sectionBounds(lines, "proxy-groups")
	if !ok {
		return lines
	}

	var ranges [][2]int
	for i := sectionStart + 1; i < sectionEnd; {
		if leadingSpaces(lines[i]) == 0 && strings.HasPrefix(strings.TrimLeft(lines[i], " "), "- ") {
			start, end := itemBounds(lines, i, sectionEnd)
			if name, ok := proxyName(lines[start:end]); ok && names[name] {
				ranges = append(ranges, [2]int{start, end})
			}
			i = end
			continue
		}
		i++
	}

	if len(ranges) == 0 {
		return lines
	}

	keep := make([]bool, len(lines))
	for i := range keep {
		keep[i] = true
	}
	for _, r := range ranges {
		for i := r[0]; i < r[1]; i++ {
			keep[i] = false
		}
	}

	filtered := make([]string, 0, len(lines))
	for i, line := range lines {
		if keep[i] {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func insertStashSplitGroups(lines []string) []string {
	_, groupEnd, hasGroups := sectionBounds(lines, "proxy-groups")
	if !hasGroups {
		insertAt := len(lines)
		if rulesStart, _, hasRules := sectionBounds(lines, "rules"); hasRules {
			insertAt = rulesStart
		}
		insert := append([]string{"proxy-groups:"}, StashSplitGroupLines...)
		return insertLines(lines, insertAt, insert)
	}
	return insertLines(lines, groupEnd, StashSplitGroupLines)
}

func replaceRules(lines []string) []string {
	replacement := append([]string{"rules:"}, StashRuleLines...)
	start, end, ok := sectionBounds(lines, "rules")
	if !ok {
		return append(lines, replacement...)
	}
	result := make([]string, 0, len(lines)-end+start+len(replacement))
	result = append(result, lines[:start]...)
	result = append(result, replacement...)
	result = append(result, lines[end:]...)
	return result
}

func insertLines(lines []string, index int, insert []string) []string {
	result := make([]string, 0, len(lines)+len(insert))
	result = append(result, lines[:index]...)
	result = append(result, insert...)
	result = append(result, lines[index:]...)
	return result
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

func expandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
