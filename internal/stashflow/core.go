package stashflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	topLevelKey      = regexp.MustCompile(`^[^\s#-][^:]*:\s*`)
	flowProxyPattern = regexp.MustCompile(`^(?P<prefix>\s*proxies:\s*)\[(?P<body>.*)\](?P<suffix>\s*(?:#.*)?)$`)
	qxSectionPattern = regexp.MustCompile(`^\s*\[[^\]]+\]\s*(?:[#;].*)?$`)
	qxTagPattern     = regexp.MustCompile(`(?i)(?:^|,\s*)tag\s*=\s*([^,]+)`)
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
	Text           string
	Clean          CleanResult
	Split          SplitResult
	Changed        bool
	OutputPath     string
	OutputChanged  bool
	ProfilePath    string
	ProfileChanged bool
	ProfileUpdated bool
	BackupPath     string
	BackupMade     bool
	OriginalLen    int
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

func CleanQXText(text string) CleanResult {
	trailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}

	lines, removals := removeQXUnsupportedProxyItems(lines)
	names := make(map[string]bool, len(removals))
	for _, removal := range removals {
		names[removal.Name] = true
	}
	lines, refs := removeQXPolicyReferences(lines, names)

	cleaned := strings.Join(lines, "\n")
	if trailingNewline {
		cleaned += "\n"
	}

	return CleanResult{Text: cleaned, Removals: removals, ReferenceCount: refs}
}

func ApplyQXSplitRules(text string) SplitResult {
	trailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}

	names := make(map[string]bool, len(QXSplitGroupNames))
	for _, name := range QXSplitGroupNames {
		names[name] = true
	}

	lines = upsertQXGeneralLazycatSettings(lines)
	lines = upsertQXPolicyLines(lines, names)
	lines = replaceQXFilterLocal(lines)

	updated := strings.Join(lines, "\n")
	if trailingNewline {
		updated += "\n"
	}

	return SplitResult{
		Text:       updated,
		Changed:    updated != text,
		GroupCount: len(QXSplitGroupNames),
		RuleCount:  len(QXRuleLines),
	}
}

func FixQXText(text string, applySplit bool) FixResult {
	clean := CleanQXText(text)
	finalText := clean.Text
	split := SplitResult{Text: finalText}
	if applySplit {
		split = ApplyQXSplitRules(finalText)
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

func FixQXFile(path string, applySplit bool, backup bool) (FixResult, error) {
	result, err := PreviewQXFile(path, applySplit)
	if err != nil {
		return FixResult{}, err
	}
	if !result.Changed {
		return result, nil
	}

	if result.OutputChanged {
		outputExists := false
		perm := os.FileMode(0o644)
		if info, err := os.Stat(result.OutputPath); err == nil {
			outputExists = true
			perm = info.Mode().Perm()
		} else if !os.IsNotExist(err) {
			return FixResult{}, err
		} else if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}

		if backup && outputExists {
			backupPath := NextBackupPath(result.OutputPath)
			if err := copyFile(result.OutputPath, backupPath); err != nil {
				return FixResult{}, err
			}
			result.BackupPath = backupPath
			result.BackupMade = true
		}

		if err := os.WriteFile(result.OutputPath, []byte(result.Text), perm); err != nil {
			return FixResult{}, err
		}
	}

	if result.ProfileChanged {
		if err := WriteQXLazycatProfile(); err != nil {
			return FixResult{}, err
		}
		result.ProfileUpdated = true
	}

	return result, nil
}

func AnalyzeQXFile(path string) (FixResult, error) {
	return PreviewQXFile(path, true)
}

func PreviewQXFile(path string, applySplit bool) (FixResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FixResult{}, err
	}

	result := FixQXText(string(data), applySplit)
	result.OutputPath = QXOutputPath(path)
	output, err := os.ReadFile(result.OutputPath)
	switch {
	case err == nil:
		result.OutputChanged = string(output) != result.Text
	case os.IsNotExist(err):
		result.OutputChanged = true
	default:
		return FixResult{}, err
	}
	result.Changed = result.OutputChanged

	if applySplit {
		profilePath := QXLazycatProfilePath()
		result.ProfilePath = profilePath
		profile, err := os.ReadFile(profilePath)
		switch {
		case err == nil:
			result.ProfileChanged = string(profile) != QXLazycatProfileText()
		case os.IsNotExist(err):
			result.ProfileChanged = true
		default:
			return FixResult{}, err
		}
		result.Changed = result.Changed || result.ProfileChanged
	}
	return result, nil
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

func QXOutputPath(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+"-QX.yaml")
}

func QXLazycatProfilePath() string {
	if dir := os.Getenv("STASHFLOW_QX_PROFILES_DIR"); dir != "" {
		return filepath.Join(expandHome(dir), "FILTER_LAZYCAT")
	}
	return filepath.Join(
		os.Getenv("HOME"),
		"Library/Mobile Documents/iCloud~com~crossutility~quantumult-x/Documents/Profiles/FILTER_LAZYCAT",
	)
}

func QXLazycatProfileText() string {
	return strings.Join(QXLazycatProfileLines, "\n") + "\n"
}

func WriteQXLazycatProfile() error {
	path := QXLazycatProfilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(QXLazycatProfileText()), 0o600)
}

func DiscoverFiles(args []string) []string {
	return DiscoverFilesForTarget(args, "stash")
}

func DiscoverFilesForTarget(args []string, target string) []string {
	paths, err := ResolveFilesForTarget(args, target)
	if err != nil {
		return []string{err.Error()}
	}
	return paths
}

func ResolveFilesForTarget(args []string, target string) ([]string, error) {
	if len(args) > 0 {
		paths := make([]string, 0, len(args))
		for _, arg := range args {
			if IsHTTPURL(arg) {
				path, err := DownloadSubscription(arg, target)
				if err != nil {
					return nil, err
				}
				paths = append(paths, path)
				continue
			}
			paths = append(paths, expandHome(arg))
		}
		return paths, nil
	}

	seen := map[string]bool{}
	var paths []string
	patterns := []string{"*.yaml", "*.yml"}
	if strings.EqualFold(target, "qx") {
		patterns = []string{"*.conf"}
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() && !seen[match] {
				seen[match] = true
				paths = append(paths, match)
			}
		}
	}
	return paths, nil
}

func HasURLInput(args []string) bool {
	for _, arg := range args {
		if IsHTTPURL(arg) {
			return true
		}
	}
	return false
}

func IsHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func DownloadSubscription(rawURL string, target string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "stashflow")

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载订阅失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return "", fmt.Errorf("下载订阅失败: 响应为空")
	}

	path := SubscriptionDownloadPath(rawURL, target)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func SubscriptionDownloadPath(rawURL string, target string) string {
	parsed, err := url.Parse(rawURL)
	name := "subscription"
	if err == nil {
		host := sanitizeFilename(parsed.Hostname())
		base := sanitizeFilename(strings.TrimSuffix(filepath.Base(parsed.Path), filepath.Ext(parsed.Path)))
		switch {
		case host != "" && base != "" && base != ".":
			name = host + "-" + base
		case host != "":
			name = host + "-subscription"
		case base != "" && base != ".":
			name = base
		}
	}

	sum := sha256.Sum256([]byte(rawURL))
	suffix := hex.EncodeToString(sum[:])[:8]
	ext := ".yaml"
	if strings.EqualFold(target, "qx") {
		ext = ".conf"
	}
	return name + "-" + suffix + ext
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-._")
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

func qxSectionBounds(lines []string, sectionName string) (int, int, bool) {
	expected := "[" + strings.ToLower(sectionName) + "]"
	for start, line := range lines {
		if strings.EqualFold(strings.TrimSpace(stripQXComment(line)), expected) {
			end := len(lines)
			for i := start + 1; i < len(lines); i++ {
				if qxSectionPattern.MatchString(lines[i]) {
					end = i
					break
				}
			}
			return start, end, true
		}
	}
	return 0, 0, false
}

func removeQXUnsupportedProxyItems(lines []string) ([]string, []Removal) {
	sectionStart, sectionEnd, ok := qxSectionBounds(lines, "server_local")
	if !ok {
		return lines, nil
	}

	var removals []Removal
	keep := make([]bool, len(lines))
	for i := range keep {
		keep[i] = true
	}

	for i := sectionStart + 1; i < sectionEnd; i++ {
		if !isQXUnsupportedProxyLine(lines[i]) {
			continue
		}
		name := qxProxyTag(lines[i])
		if name == "" {
			name = fmt.Sprintf("<unsupported:%d>", i+1)
		}
		removals = append(removals, Removal{Name: name, Line: i + 1, UUID: "hy2"})
		keep[i] = false
	}

	if len(removals) == 0 {
		return lines, removals
	}

	filtered := make([]string, 0, len(lines))
	for i, line := range lines {
		if keep[i] {
			filtered = append(filtered, line)
		}
	}
	return filtered, removals
}

func isQXUnsupportedProxyLine(line string) bool {
	value := strings.TrimSpace(stripInlineComment(line))
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "hysteria2=") ||
		strings.HasPrefix(lower, "hy2=") ||
		strings.Contains(lower, "type=hysteria2") ||
		strings.Contains(lower, "type=hy2") ||
		strings.Contains(lower, "protocol=hysteria2") ||
		strings.Contains(lower, "protocol=hy2")
}

func qxProxyTag(line string) string {
	match := qxTagPattern.FindStringSubmatch(stripInlineComment(line))
	if len(match) < 2 {
		return ""
	}
	return unquote(strings.TrimSpace(match[1]))
}

func removeQXPolicyReferences(lines []string, names map[string]bool) ([]string, int) {
	if len(names) == 0 {
		return lines, 0
	}

	sectionStart, sectionEnd, ok := qxSectionBounds(lines, "policy")
	if !ok {
		return lines, 0
	}

	removed := 0
	output := make([]string, 0, len(lines))
	output = append(output, lines[:sectionStart+1]...)
	for _, line := range lines[sectionStart+1 : sectionEnd] {
		updated, count := removeQXPolicyLineReferences(line, names)
		removed += count
		output = append(output, updated)
	}
	output = append(output, lines[sectionEnd:]...)
	return output, removed
}

func removeQXPolicyLineReferences(line string, names map[string]bool) (string, int) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
		return line, 0
	}

	comment := ""
	body := line
	if idx := strings.Index(body, "#"); idx >= 0 {
		comment = body[idx:]
		body = body[:idx]
	}

	parts := strings.Split(body, ",")
	if len(parts) < 2 {
		return line, 0
	}

	removed := 0
	kept := []string{strings.TrimRight(parts[0], " ")}
	for _, part := range parts[1:] {
		value := unquote(strings.TrimSpace(part))
		if names[value] {
			removed++
			continue
		}
		kept = append(kept, strings.TrimSpace(part))
	}

	if removed == 0 {
		return line, 0
	}

	updated := strings.Join(kept, ", ")
	if comment != "" {
		updated += " " + comment
	}
	return updated, removed
}

func upsertQXPolicyLines(lines []string, names map[string]bool) []string {
	start, end, ok := qxSectionBounds(lines, "policy")
	if !ok {
		insertAt := len(lines)
		if filterStart, _, hasFilter := qxSectionBounds(lines, "filter_local"); hasFilter {
			insertAt = filterStart
		}
		insert := append([]string{"[policy]"}, QXPolicyLines...)
		return insertLines(lines, insertAt, insert)
	}

	section := make([]string, 0, end-start+len(QXPolicyLines))
	section = append(section, lines[start])
	for _, line := range lines[start+1 : end] {
		if qxPolicyNameManaged(line, names) {
			continue
		}
		section = append(section, line)
	}
	section = append(section, QXPolicyLines...)

	result := make([]string, 0, len(lines)-end+start+len(section))
	result = append(result, lines[:start]...)
	result = append(result, section...)
	result = append(result, lines[end:]...)
	return result
}

func upsertQXGeneralLazycatSettings(lines []string) []string {
	start, end, ok := qxSectionBounds(lines, "general")
	if !ok {
		insert := []string{
			"[general]",
			"dns_exclusion_list = " + strings.Join(QXLazycatDNSExclusionValues, ", "),
			"excluded_routes = " + strings.Join(QXLazycatExcludedRouteValues, ", "),
			"",
		}
		return insertLines(lines, 0, insert)
	}

	section := append([]string{}, lines[start:end]...)
	section = upsertQXListSetting(section, "dns_exclusion_list", QXLazycatDNSExclusionValues)
	section = upsertQXListSetting(section, "excluded_routes", QXLazycatExcludedRouteValues)

	result := make([]string, 0, len(lines)-end+start+len(section))
	result = append(result, lines[:start]...)
	result = append(result, section...)
	result = append(result, lines[end:]...)
	return result
}

func upsertQXListSetting(section []string, key string, values []string) []string {
	for i := 1; i < len(section); i++ {
		line := section[i]
		trimmed := strings.TrimSpace(stripQXComment(line))
		if trimmed == "" || !strings.Contains(trimmed, "=") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if !strings.EqualFold(strings.TrimSpace(parts[0]), key) {
			continue
		}

		merged := mergeQXListValues(splitQXListValues(parts[1]), values)
		section[i] = strings.Repeat(" ", leadingSpaces(line)) + key + " = " + strings.Join(merged, ", ")
		return section
	}

	return append(section, key+" = "+strings.Join(values, ", "))
}

func mergeQXListValues(existing []string, values []string) []string {
	result := append([]string{}, existing...)
	seen := make(map[string]bool, len(result)+len(values))
	for _, value := range result {
		seen[strings.ToLower(value)] = true
	}
	for _, value := range values {
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func splitQXListValues(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func qxPolicyNameManaged(line string, names map[string]bool) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
		return false
	}
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	eq := strings.Index(trimmed, "=")
	comma := strings.Index(trimmed, ",")
	if eq < 0 || comma < 0 || comma < eq {
		return false
	}
	name := unquote(strings.TrimSpace(trimmed[eq+1 : comma]))
	return names[name]
}

func replaceQXFilterLocal(lines []string) []string {
	replacement := append([]string{"[filter_local]"}, QXRuleLines...)
	start, end, ok := qxSectionBounds(lines, "filter_local")
	if !ok {
		return append(lines, replacement...)
	}
	result := make([]string, 0, len(lines)-end+start+len(replacement))
	result = append(result, lines[:start]...)
	result = append(result, replacement...)
	result = append(result, lines[end:]...)
	return result
}

func qxRuleLinesFromStash(lines []string) []string {
	rules := make([]string, 0, len(lines))
	for _, line := range lines {
		rule, ok := qxRuleLineFromStash(line)
		if ok {
			rules = append(rules, rule)
		}
	}
	return rules
}

func qxRuleLineFromStash(line string) (string, bool) {
	value := strings.TrimSpace(line)
	value = strings.TrimPrefix(value, "- ")
	value = unquote(strings.TrimSpace(value))
	if value == "" {
		return "", false
	}

	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < 2 {
		return "", false
	}

	switch strings.ToUpper(parts[0]) {
	case "DOMAIN-SUFFIX":
		parts[0] = "HOST-SUFFIX"
	case "DOMAIN-KEYWORD":
		parts[0] = "HOST-KEYWORD"
	case "DOMAIN":
		parts[0] = "HOST"
	case "IP-CIDR6":
		parts[0] = "IP6-CIDR"
	case "MATCH":
		parts[0] = "FINAL"
		parts = parts[:2]
	default:
		parts[0] = strings.ToUpper(parts[0])
	}

	for i := 1; i < len(parts); i++ {
		switch strings.ToUpper(parts[i]) {
		case "DIRECT":
			parts[i] = "direct"
		case "REJECT":
			parts[i] = "reject"
		case "NO-RESOLVE":
			parts[i] = "no-resolve"
		}
	}

	return strings.Join(parts, ","), true
}

func stripInlineComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func stripQXComment(line string) string {
	hash := strings.Index(line, "#")
	semicolon := strings.Index(line, ";")
	switch {
	case hash >= 0 && semicolon >= 0 && hash < semicolon:
		return line[:hash]
	case hash >= 0 && semicolon >= 0:
		return line[:semicolon]
	case hash >= 0:
		return line[:hash]
	case semicolon >= 0:
		return line[:semicolon]
	default:
		return line
	}
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
