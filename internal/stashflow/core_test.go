package stashflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanQXTextRemovesUnsupportedHy2AndPolicyReferences(t *testing.T) {
	input := strings.Join([]string{
		"[server_local]",
		"trojan=example.com:443, password=p, over-tls=true, tag=Keep",
		"hysteria2=hy2.example.com:443, password=p, tag=Hy2 One",
		"hy2=hy2b.example.com:443, password=p, tag=Hy2 Two",
		"[policy]",
		"static=Proxy, Keep, Hy2 One, Hy2 Two, direct",
		"",
	}, "\n")

	result := CleanQXText(input)

	if len(result.Removals) != 2 {
		t.Fatalf("expected 2 removals, got %d", len(result.Removals))
	}
	if result.ReferenceCount != 2 {
		t.Fatalf("expected 2 removed references, got %d", result.ReferenceCount)
	}
	if strings.Contains(result.Text, "hysteria2=") || strings.Contains(result.Text, "hy2=") {
		t.Fatalf("expected hy2 server lines to be removed:\n%s", result.Text)
	}
	if strings.Contains(result.Text, "Hy2 One") || strings.Contains(result.Text, "Hy2 Two") {
		t.Fatalf("expected hy2 policy references to be removed:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "static=Proxy, Keep, direct") {
		t.Fatalf("expected existing policy line to be preserved without hy2 references:\n%s", result.Text)
	}
}

func TestApplyQXSplitRulesAddsPoliciesAndConvertedRules(t *testing.T) {
	result := ApplyQXSplitRules("[general]\nserver_check_url=http://example.com\n")

	for _, want := range []string{
		"dns_exclusion_list = *.heiyu.space, *.lazycat.cloud",
		"excluded_routes = 6.6.6.6/32, 2000::6666/128",
		"[policy]",
		"static=🛑 广告拦截, reject, direct, ✨ 星链Starlink",
		"[filter_local]",
		"HOST-SUFFIX,heiyu.space,direct",
		"HOST-SUFFIX,lazycat.cloud,direct",
		"IP-CIDR,6.6.6.6/32,direct,no-resolve",
		"IP6-CIDR,2000::6666/128,direct,no-resolve",
		"IP6-CIDR,fc03:1136:3800::/40,direct,no-resolve",
		"HOST-SUFFIX,googlesyndication.com,🛑 广告拦截",
		"IP6-CIDR,fe80::/10,direct",
		"FINAL,🐟 漏网之鱼",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("expected QX output to contain %q:\n%s", want, result.Text)
		}
	}
	if !result.Changed {
		t.Fatal("expected QX split rules to change the text")
	}
	if result.GroupCount != len(QXSplitGroupNames) {
		t.Fatalf("expected group count %d, got %d", len(QXSplitGroupNames), result.GroupCount)
	}
	if result.RuleCount != len(QXRuleLines) {
		t.Fatalf("expected rule count %d, got %d", len(QXRuleLines), result.RuleCount)
	}
}

func TestApplyQXSplitRulesMergesLazycatGeneralSettings(t *testing.T) {
	input := strings.Join([]string{
		"[general]",
		"dns_exclusion_list = example.com, *.heiyu.space",
		"excluded_routes = 10.0.0.0/8",
		"[filter_local]",
		"HOST-SUFFIX,old.example,direct",
		"",
	}, "\n")

	result := ApplyQXSplitRules(input)

	for _, want := range []string{
		"dns_exclusion_list = example.com, *.heiyu.space, *.lazycat.cloud",
		"excluded_routes = 10.0.0.0/8, 6.6.6.6/32, 2000::6666/128",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("expected QX output to contain %q:\n%s", want, result.Text)
		}
	}
	if strings.Count(result.Text, "*.heiyu.space") != 1 {
		t.Fatalf("expected heiyu.space DNS exclusion to be deduplicated:\n%s", result.Text)
	}
}

func TestQXRuleLineFromStash(t *testing.T) {
	cases := map[string]string{
		"- 'DOMAIN-SUFFIX,local,DIRECT'":                       "HOST-SUFFIX,local,direct",
		"- 'DOMAIN-KEYWORD,adservice,🛑 广告拦截'":                  "HOST-KEYWORD,adservice,🛑 广告拦截",
		"- 'DOMAIN,fastly-download.epicgames.com,DIRECT'":      "HOST,fastly-download.epicgames.com,direct",
		"- 'IP-CIDR6,2001:67c:4e8::/48,💬 Telegram,no-resolve'": "IP6-CIDR,2001:67c:4e8::/48,💬 Telegram,no-resolve",
		"- 'MATCH,🐟 漏网之鱼'":                                     "FINAL,🐟 漏网之鱼",
	}

	for input, want := range cases {
		got, ok := qxRuleLineFromStash(input)
		if !ok {
			t.Fatalf("expected %q to convert", input)
		}
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestQXOutputPathUsesSourceNameWithQXSuffix(t *testing.T) {
	got := QXOutputPath(filepath.Join("configs", "Starlink.conf"))
	want := filepath.Join("configs", "Starlink-QX.yaml")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFixQXFileSavesAsQXYAMLWithoutOverwritingSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "Starlink.conf")
	original := strings.Join([]string{
		"[server_local]",
		"hysteria2=hy2.example.com:443, password=p, tag=Hy2",
		"[policy]",
		"static=Proxy, Hy2, direct",
		"",
	}, "\n")
	if err := os.WriteFile(source, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := FixQXFile(source, true, true)
	if err != nil {
		t.Fatal(err)
	}

	if result.OutputPath != filepath.Join(dir, "Starlink-QX.yaml") {
		t.Fatalf("unexpected output path: %s", result.OutputPath)
	}
	if result.BackupMade {
		t.Fatalf("did not expect a backup when creating a new output file")
	}
	sourceData, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceData) != original {
		t.Fatalf("expected source file to remain unchanged:\n%s", sourceData)
	}
	outputData, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outputData), "hysteria2=") || strings.Contains(string(outputData), "Hy2") {
		t.Fatalf("expected output to remove unsupported QX proxy and references:\n%s", outputData)
	}

	preview, err := PreviewQXFile(source, true)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Changed {
		t.Fatalf("expected generated QX output to be up to date")
	}
}

func TestFixQXFileBacksUpExistingOutput(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "Starlink.conf")
	if err := os.WriteFile(source, []byte("[server_local]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "Starlink-QX.yaml")
	if err := os.WriteFile(output, []byte("old output\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := FixQXFile(source, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.BackupMade {
		t.Fatalf("expected existing output to be backed up")
	}
	backupData, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backupData) != "old output\n" {
		t.Fatalf("expected backup to contain old output, got:\n%s", backupData)
	}
}
