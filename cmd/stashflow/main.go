package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iwen-conf/stashflow/internal/stashflow"
)

type target struct {
	path   string
	result stashflow.FixResult
	err    error
}

func (t target) badCount() int {
	return len(t.result.Clean.Removals)
}

func (t target) referenceCount() int {
	return t.result.Clean.ReferenceCount
}

func (t target) splitNeeded() bool {
	return t.result.Split.Changed
}

func (t target) needsWork() bool {
	return t.err == nil && t.result.Changed
}

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmOne
	confirmAll
)

type model struct {
	args        []string
	paths       []string
	targets     []target
	target      string
	selected    int
	offset      int
	width       int
	height      int
	backup      bool
	message     string
	scanErr     error
	urlInput    bool
	urlText     string
	urlError    string
	confirm     confirmKind
	confirmText string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("44")).Padding(0, 1)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("252")).Bold(true)
	sectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printUsage()
		return
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "不支持命令行参数: %s\n\n", arg)
			printUsage()
			os.Exit(2)
		}
	}

	targetName := "stash"
	if stashflow.HasURLInput(args) {
		targetName = "qx"
	}
	paths, scanErr := stashflow.ResolveFilesForTarget(args, targetName)
	m := model{
		args:    args,
		paths:   paths,
		target:  targetName,
		backup:  true,
		message: "已扫描订阅文件",
		scanErr: scanErr,
	}
	if scanErr != nil {
		m.message = scanErr.Error()
	} else if stashflow.HasURLInput(args) {
		m.message = "已下载订阅并切换到 QX"
	}
	m.refresh()

	program := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 TUI 失败: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "用法: %s [配置文件 ...]\n\n", filepath.Base(os.Args[0]))
	fmt.Fprintln(os.Stdout, "中文 TUI，用于清理 Stash 异常 UUID 或 QX 不支持的 hy2 节点，并补回分流规则。")
	fmt.Fprintln(os.Stdout, "运行后在界面内按 t 切换 Stash/QX，按 u 输入订阅链接，按 Enter 或 A 保存。")
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if m.confirm != confirmNone {
			switch key {
			case "y", "Y":
				if m.confirm == confirmOne {
					m.fixSelected()
				} else {
					m.fixAll()
				}
				m.confirm = confirmNone
				m.confirmText = ""
			case "n", "N", "esc", "q":
				m.message = "已取消"
				m.confirm = confirmNone
				m.confirmText = ""
			}
			return m, nil
		}

		if m.urlInput {
			switch key {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.urlInput = false
				m.urlText = ""
				m.urlError = ""
				m.message = "已取消输入订阅链接"
			case "enter":
				m.submitURLInput()
			case "backspace", "ctrl+h":
				runes := []rune(m.urlText)
				if len(runes) > 0 {
					m.urlText = string(runes[:len(runes)-1])
				}
			case "ctrl+u":
				m.urlText = ""
				m.urlError = ""
			default:
				if len(msg.Runes) > 0 {
					m.urlText += string(msg.Runes)
					m.urlError = ""
				}
			}
			return m, nil
		}

		switch key {
		case "ctrl+c", "q", "Q":
			return m, tea.Quit
		case "up", "k", "K":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j", "J":
			if m.selected < len(m.targets)-1 {
				m.selected++
			}
		case "r", "R":
			m.refresh()
			m.message = "已重新扫描"
		case "t", "T":
			m.switchTarget()
		case "u", "U":
			m.urlInput = true
			m.urlText = ""
			m.urlError = ""
			m.message = "粘贴订阅链接后按 Enter"
		case "b", "B":
			m.backup = !m.backup
			if m.backup {
				m.message = "备份已开启"
			} else {
				m.message = "备份已关闭"
			}
		case "enter":
			if len(m.targets) == 0 {
				m.message = "没有可处理的" + m.fileKindLabel() + "文件"
				break
			}
			current := m.targets[m.selected]
			if !current.needsWork() {
				m.message = m.noWorkMessage(current)
				break
			}
			m.confirm = confirmOne
			m.confirmText = fmt.Sprintf("确认保存 %s？", m.saveTargetLabel(current))
		case "A", "a":
			count := m.dirtyCount()
			if count == 0 {
				m.message = "没有需要处理的文件"
				break
			}
			m.confirm = confirmAll
			m.confirmText = fmt.Sprintf("确认保存全部 %d 个%s文件？", count, m.targetDisplayName())
		}
	}
	return m, nil
}

func (m model) View() string {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 28
	}
	if width < 70 || height < 14 {
		return "窗口太小，请至少调整到 70x14。\n"
	}

	var b strings.Builder
	totalBad, totalSplit, totalRefs := m.totals()
	backupText := "开"
	if !m.backup {
		backupText = "关"
	}

	b.WriteString(titleStyle.Render(m.title()))
	b.WriteByte('\n')
	b.WriteString(mutedStyle.Render(fmt.Sprintf("目标: %s | 文件: %d | %s: %d | 待补分流: %d | 引用: %d | 备份: %s", m.targetDisplayName(), len(m.targets), m.issueLabel(), totalBad, totalSplit, totalRefs, backupText)))
	b.WriteByte('\n')
	b.WriteString(mutedStyle.Render("↑/↓/j/k 选择 · t 切换 Stash/QX · u 输入订阅链接 · Enter 保存当前 · A 保存全部 · b 切换备份 · r 重新扫描 · q 退出"))
	b.WriteString("\n\n")

	listWidth := width / 2
	if listWidth < 38 {
		listWidth = 38
	}
	if listWidth > 62 {
		listWidth = 62
	}
	detailWidth := width - listWidth - 4
	listHeight := height - 9
	if listHeight < 4 {
		listHeight = 4
	}

	m.adjustOffset(listHeight)
	list := m.renderList(listWidth, listHeight)
	detail := m.renderDetail(detailWidth, listHeight)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail))
	b.WriteByte('\n')
	b.WriteString(mutedStyle.Render(strings.Repeat("-", max(0, width-1))))
	b.WriteByte('\n')
	if m.urlInput {
		if m.urlError != "" {
			b.WriteString(errorStyle.Render(truncate("下载失败: "+m.urlError, max(12, width-1))))
			b.WriteByte('\n')
		}
		b.WriteString(warnStyle.Render(truncate(m.urlPrompt(), max(12, width-26)) + "  Enter 下载，Esc 取消"))
	} else if m.confirm != confirmNone {
		b.WriteString(warnStyle.Render(m.confirmText + "  按 y 确认，n 取消"))
	} else {
		b.WriteString(mutedStyle.Render(m.message))
	}
	return b.String()
}

func (m *model) submitURLInput() {
	rawURL := strings.TrimSpace(m.urlText)
	if rawURL == "" {
		m.urlError = "请输入订阅链接"
		return
	}
	if !stashflow.IsHTTPURL(rawURL) {
		m.urlError = "订阅链接必须以 http:// 或 https:// 开头"
		return
	}

	paths, scanErr := stashflow.ResolveFilesForTarget([]string{rawURL}, "qx")
	if scanErr != nil {
		m.urlError = scanErr.Error()
		return
	}

	m.args = []string{rawURL}
	m.paths = paths
	m.scanErr = nil
	m.target = "qx"
	m.selected = 0
	m.offset = 0
	m.urlInput = false
	m.urlText = ""
	m.urlError = ""
	m.refresh()
	m.message = "已下载订阅并切换到 QX"
}

func (m model) urlPrompt() string {
	return "订阅链接: " + m.urlText + "|"
}

func (m *model) refresh() {
	m.targets = make([]target, 0, len(m.paths))
	if m.scanErr != nil {
		m.targets = append(m.targets, target{path: m.scanErr.Error(), err: m.scanErr})
		return
	}
	for _, path := range m.paths {
		result, err := stashflow.AnalyzeFile(path)
		if m.target == "qx" {
			result, err = stashflow.AnalyzeQXFile(path)
		}
		m.targets = append(m.targets, target{path: path, result: result, err: err})
	}
	if m.selected >= len(m.targets) {
		m.selected = max(0, len(m.targets)-1)
	}
}

func (m *model) switchTarget() {
	if m.target == "qx" {
		m.target = "stash"
	} else {
		m.target = "qx"
	}
	paths, scanErr := stashflow.ResolveFilesForTarget(m.args, m.target)
	m.paths = paths
	m.scanErr = scanErr
	m.selected = 0
	m.offset = 0
	m.refresh()
	if scanErr != nil {
		m.message = scanErr.Error()
		return
	}
	m.message = "已切换到 " + m.targetDisplayName() + " 并重新扫描"
}

func (m *model) fixSelected() {
	if len(m.targets) == 0 {
		m.message = "没有可处理的文件"
		return
	}
	t := m.targets[m.selected]
	result, err := stashflow.FixFile(t.path, true, m.backup)
	if m.target == "qx" {
		result, err = stashflow.FixQXFile(t.path, true, m.backup)
	}
	if err != nil {
		m.message = filepath.Base(t.path) + ": " + err.Error()
		return
	}
	m.refresh()
	m.message = m.fixMessage(t.path, result)
}

func (m *model) fixAll() {
	count := 0
	var last string
	for _, t := range m.targets {
		if !t.needsWork() {
			continue
		}
		result, err := stashflow.FixFile(t.path, true, m.backup)
		if m.target == "qx" {
			result, err = stashflow.FixQXFile(t.path, true, m.backup)
		}
		if err != nil {
			last = filepath.Base(t.path) + ": " + err.Error()
			continue
		}
		count++
		last = m.fixMessage(t.path, result)
	}
	m.refresh()
	if count == 0 {
		if last == "" {
			m.message = "没有需要处理的文件"
		} else {
			m.message = last
		}
		return
	}
	m.message = fmt.Sprintf("已修复 %d 个文件", count)
}

func (m model) fixMessage(path string, result stashflow.FixResult) string {
	if !result.Changed {
		if result.OutputPath != "" {
			return filepath.Base(path) + ": 输出已是最新 " + filepath.Base(result.OutputPath)
		}
		return filepath.Base(path) + ": 无需处理"
	}
	parts := []string{}
	if len(result.Clean.Removals) > 0 {
		parts = append(parts, fmt.Sprintf("删除 %d 个%s", len(result.Clean.Removals), m.issueItemLabel()))
	}
	if result.Split.Changed {
		parts = append(parts, fmt.Sprintf("补回 %d 个分组/%d 条规则", result.Split.GroupCount, result.Split.RuleCount))
	}
	if result.BackupMade {
		parts = append(parts, "备份 "+filepath.Base(result.BackupPath))
	}
	if result.OutputChanged && result.OutputPath != "" {
		parts = append(parts, "保存 "+filepath.Base(result.OutputPath))
	}
	if result.ProfileUpdated {
		parts = append(parts, "更新 "+filepath.Base(result.ProfilePath))
	}
	return filepath.Base(path) + ": " + strings.Join(parts, "，")
}

func (m model) totals() (bad int, split int, refs int) {
	for _, t := range m.targets {
		bad += t.badCount()
		refs += t.referenceCount()
		if t.splitNeeded() {
			split++
		}
	}
	return bad, split, refs
}

func (m model) dirtyCount() int {
	count := 0
	for _, t := range m.targets {
		if t.needsWork() {
			count++
		}
	}
	return count
}

func (m *model) adjustOffset(listHeight int) {
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+listHeight {
		m.offset = m.selected - listHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m model) renderList(width int, height int) string {
	lines := []string{sectionStyle.Render("文件")}
	if len(m.targets) == 0 {
		lines = append(lines, errorStyle.Render("未找到"+m.fileKindLabel()+"文件"))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	end := min(len(m.targets), m.offset+height)
	for i := m.offset; i < end; i++ {
		t := m.targets[i]
		status, style := m.targetStatus(t)
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		line := fmt.Sprintf("%s %-8s %s", prefix, status, filepath.Base(t.path))
		line = truncate(line, width)
		if i == m.selected {
			lines = append(lines, selectedStyle.Width(width).Render(line))
		} else {
			lines = append(lines, style.Render(line))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) targetStatus(t target) (string, lipgloss.Style) {
	if t.err != nil {
		return "错误", errorStyle
	}
	if m.target == "qx" && !t.needsWork() {
		return "已保存", okStyle
	}
	if m.target == "qx" && t.result.ProfileChanged && !t.result.OutputChanged {
		return "分流数据", warnStyle
	}
	if t.badCount() > 0 && t.splitNeeded() {
		return fmt.Sprintf("%d%s+分流", t.badCount(), m.shortIssueLabel()), warnStyle
	}
	if t.badCount() > 0 {
		return fmt.Sprintf("%d%s", t.badCount(), m.shortIssueLabel()), warnStyle
	}
	if t.splitNeeded() {
		return "补分流", warnStyle
	}
	return "正常", okStyle
}

func (m model) renderDetail(width int, height int) string {
	lines := []string{sectionStyle.Render("详情")}
	if len(m.targets) == 0 {
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	t := m.targets[m.selected]
	if t.err != nil {
		lines = append(lines, errorStyle.Render(t.err.Error()))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	if m.target == "qx" && !t.needsWork() {
		lines = append(lines, okStyle.Render("输出和分流数据已是最新："+filepath.Base(t.result.OutputPath)))
		lines = append(lines, mutedStyle.Render("源文件: "+t.path))
		lines = append(lines, mutedStyle.Render("输出: "+t.result.OutputPath))
		lines = append(lines, mutedStyle.Render("QX 数据: "+t.result.ProfilePath))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	if m.target == "qx" && t.result.ProfileChanged && !t.result.OutputChanged && !t.splitNeeded() && t.badCount() == 0 {
		lines = append(lines, warnStyle.Render("需要更新 QX 分流数据："+filepath.Base(t.result.ProfilePath)))
		lines = append(lines, mutedStyle.Render("路径: "+t.path))
		lines = append(lines, mutedStyle.Render("QX 数据: "+t.result.ProfilePath))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	if t.badCount() == 0 && !t.splitNeeded() {
		lines = append(lines, okStyle.Render("无需处理："+m.issueLabel()+"已清理，分流模板已应用。"))
		lines = append(lines, mutedStyle.Render("路径: "+t.path))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}

	if t.badCount() > 0 {
		lines = append(lines, warnStyle.Render(fmt.Sprintf("%s: %d，需删除引用: %d", m.issueLabel(), t.badCount(), t.referenceCount())))
		remaining := max(0, height-len(lines)-5)
		for i, removal := range t.result.Clean.Removals {
			if i >= remaining {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("... 还有 %d 个", t.badCount()-i)))
				break
			}
			detail := removal.UUID
			if m.target == "qx" {
				detail = "protocol=hy2"
			}
			if len([]rune(detail)) > 30 {
				detail = string([]rune(detail)[:27]) + "..."
			}
			lines = append(lines, truncate(fmt.Sprintf("第 %-4d 行 %s  %s", removal.Line, removal.Name, detail), width))
		}
	}

	if t.splitNeeded() {
		lines = append(lines, warnStyle.Render(fmt.Sprintf("需要补回分流模板：%d 个分组，%d 条规则", t.result.Split.GroupCount, t.result.Split.RuleCount)))
		remaining := max(0, height-len(lines)-3)
		groupNames := stashflow.StashSplitGroupNames
		if m.target == "qx" {
			groupNames = stashflow.QXSplitGroupNames
		}
		for i, name := range groupNames {
			if i >= remaining {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("... 还有 %d 个分组", len(groupNames)-i)))
				break
			}
			lines = append(lines, "- "+name)
		}
	}

	lines = append(lines, mutedStyle.Render("路径: "+t.path))
	if t.result.OutputPath != "" {
		lines = append(lines, mutedStyle.Render("保存为: "+t.result.OutputPath))
	}
	if t.result.ProfilePath != "" {
		lines = append(lines, mutedStyle.Render("QX 数据: "+t.result.ProfilePath))
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) noWorkMessage(t target) string {
	if m.target == "qx" && t.result.OutputPath != "" {
		return filepath.Base(t.path) + ": 输出和分流数据已是最新 " + filepath.Base(t.result.OutputPath)
	}
	return filepath.Base(t.path) + ": 无需处理"
}

func (m model) saveTargetLabel(t target) string {
	if m.target == "qx" && t.result.ProfileChanged && !t.result.OutputChanged {
		return filepath.Base(t.result.ProfilePath)
	}
	if m.target == "qx" && t.result.OutputPath != "" {
		return filepath.Base(t.result.OutputPath)
	}
	return filepath.Base(t.path) + " 的" + m.targetDisplayName() + "修复"
}

func (m model) title() string {
	if m.target == "qx" {
		return "StashFlow QX 订阅修复"
	}
	return "StashFlow Stash 订阅修复"
}

func (m model) targetDisplayName() string {
	if m.target == "qx" {
		return "QX"
	}
	return "Stash"
}

func (m model) issueLabel() string {
	if m.target == "qx" {
		return "不支持 hy2"
	}
	return "异常 UUID"
}

func (m model) issueItemLabel() string {
	if m.target == "qx" {
		return "hy2 节点"
	}
	return "异常节点"
}

func (m model) shortIssueLabel() string {
	if m.target == "qx" {
		return "hy2"
	}
	return "坏"
}

func (m model) fileKindLabel() string {
	if m.target == "qx" {
		return " QX "
	}
	return " YAML "
}

func truncate(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
