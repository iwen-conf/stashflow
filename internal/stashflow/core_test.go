package stashflow

import (
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
		"[policy]",
		"static=🛑 广告拦截, reject, direct, ✨ 星链Starlink",
		"[filter_local]",
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
