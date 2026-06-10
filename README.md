# StashFlow

StashFlow is a small terminal UI for maintaining Stash, Clash, and Mihomo
subscription YAML files after provider updates.

It fixes two common subscription-update problems:

- removes proxy entries whose `uuid` value is not a canonical UUID
- re-applies a local Stash split-routing template that subscription updates may overwrite

StashFlow works on local files only. It does not upload your subscription, proxy
nodes, tokens, or YAML content anywhere.

## Install

```bash
brew tap iwen-conf/stashflow
brew install stashflow
```

You can also run the scripts directly from this repository:

```bash
bin/stashflow
bin/stashflow-clean --fix-all path/to/subscription.yaml
```

## TUI Usage

Open the TUI in a directory that contains `.yaml` or `.yml` subscription files:

```bash
stashflow
```

Or pass specific files:

```bash
stashflow "Starlink.yaml"
```

Keys:

- `Up/Down` or `j/k`: move selection
- `Enter`: fix the selected file
- `A`: fix all files that need work
- `b`: toggle backup files, enabled by default
- `r`: rescan files
- `q`: quit

When fixing a file, StashFlow first removes invalid UUID proxy entries and their
proxy-group references, then re-applies the built-in split-routing groups and
rules. A `.bak` backup is created before writing unless backup is disabled.

## CLI Usage

Preview changes:

```bash
stashflow-clean --fix-all --dry-run "Starlink.yaml"
```

Clean invalid UUID proxies only:

```bash
stashflow-clean "Starlink.yaml"
```

Clean invalid UUID proxies and re-apply Stash split rules:

```bash
stashflow-clean --fix-all "Starlink.yaml"
```

Re-apply only the Stash split rules:

```bash
stashflow-clean --apply-stash-rules "Starlink.yaml"
```

Disable backups:

```bash
stashflow-clean --fix-all --no-backup "Starlink.yaml"
```

## Split Routing Template

The built-in template creates these policy groups:

- `🛑 广告拦截`
- `💬 微信`
- `🐧 腾讯服务`
- `💰 支付服务`
- `🇨🇳 国内流量`
- `🤖 AI服务`
- `💬 Telegram`
- `📺 流媒体`
- `🍎 Apple`
- `Ⓜ️ Microsoft`
- `🎮 游戏平台`
- `🌐 国外流量`
- `🐟 漏网之鱼`

Domestic rules are intentionally explicit for common Chinese services such as
WeChat, Tencent, Alipay, UnionPay, Taobao, JD, Meituan, Amap, Bilibili, Douyin,
Xiaohongshu, Zhihu, Kuaishou, NetEase, Weibo, Xiaomi, Huawei, OPPO, and Vivo.
These groups default to `DIRECT`, but remain selectable in Stash for unusual
network conditions.

The rules are local YAML rules, not remote `rule-provider` entries. This keeps
subscription loading predictable and avoids extra network fetches when importing
the config into Stash.

## Requirements

StashFlow uses only the Python standard library.

## License

MIT
