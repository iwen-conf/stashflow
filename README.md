# StashFlow

StashFlow 是一个用 Go 和 Bubble Tea 编写的中文 TUI，用来维护 Stash、Clash、Mihomo 订阅 YAML。

它解决两个常见问题：

- 订阅更新后混入 `uuid` 不标准的代理节点
- 订阅更新后自定义分流策略被覆盖

StashFlow 只处理本地文件，不上传订阅、节点、Token 或 YAML 内容。

## 安装

```bash
brew tap iwen-conf/stashflow
brew install stashflow
```

如果 Homebrew 开启了 tap trust 检查：

```bash
brew trust iwen-conf/stashflow
```

## 使用 TUI

在订阅文件所在目录运行：

```bash
stashflow
```

也可以指定文件：

```bash
stashflow "Starlink.yaml"
```

按键：

- `↑/↓` 或 `j/k`：移动选择
- `Enter`：修复当前文件
- `A`：修复所有需要处理的文件
- `b`：切换是否生成 `.bak` 备份，默认开启
- `r`：重新扫描
- `q`：退出

修复时会先删除异常 UUID 节点和对应策略组引用，再补回内置分流分组和规则。

## 批处理

预览：

```bash
stashflow-clean --fix-all --dry-run "Starlink.yaml"
```

只清理异常 UUID：

```bash
stashflow-clean "Starlink.yaml"
```

清理异常 UUID 并补回分流规则：

```bash
stashflow-clean --fix-all "Starlink.yaml"
```

只补回分流规则：

```bash
stashflow-clean --apply-stash-rules "Starlink.yaml"
```

不创建备份：

```bash
stashflow-clean --fix-all --no-backup "Starlink.yaml"
```

## 分流模板

内置策略组：

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

国内规则明确覆盖微信、腾讯、微信支付、支付宝、银联、淘宝、京东、美团、高德、B 站、抖音、小红书、知乎、快手、网易、微博、小米、华为、OPPO、vivo 等常见服务。国内相关分组默认走 `DIRECT`，但在 Stash 里仍可以手动切到代理。

规则是本地 YAML 规则，不使用远程 `rule-provider`，导入 Stash 时不需要额外拉取远程规则集。

## 从源码运行

```bash
go run ./cmd/stashflow
go run ./cmd/stashflow-clean --fix-all "Starlink.yaml"
```

## 许可证

MIT
