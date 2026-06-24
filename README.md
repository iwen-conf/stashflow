# StashFlow

StashFlow 是一个用 Go 和 Bubble Tea 编写的中文 TUI，用来维护 Stash、Clash、Mihomo 订阅 YAML。
也可以维护 Quantumult X 配置，清理 QX 不支持的 hy2 节点并补回 QX 分流模板。

它解决几个常见问题：

- 订阅更新后混入 `uuid` 不标准的代理节点
- 订阅更新后自定义分流策略被覆盖
- QX 配置里混入 `hysteria2`/`hy2` 节点导致客户端不可用

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

运行 `stashflow` 后在 TUI 内按 `t` 在 Stash/QX 之间切换；切到 QX 后会重新扫描当前目录的 `.conf` 文件。

QX 模式不会覆盖源文件，会保存为同目录下的 `源文件名-QX.yaml`，例如 `Starlink.conf` 会输出 `Starlink-QX.yaml`。如果同名输出文件已存在且备份开启，会先备份旧输出文件。

QX 输出会自动加入懒猫微服打洞优化：`*.heiyu.space`、`*.lazycat.cloud` 使用真实 DNS，`6.6.6.6/32`、`2000::6666/128` 加入 QX `excluded_routes`，并将 `heiyu.space`、`lazycat.cloud` 和 `fc03:1136:3800::/40` 直连，减少 QX TUN 对懒猫直连/穿透逻辑的干扰。

保存 QX 输出时也会同步维护 Quantumult X 的 iCloud 分流数据文件 `Documents/Profiles/FILTER_LAZYCAT`，可在 QX 的本地/iCloud 分流资源中引用。

按键：

- `↑/↓` 或 `j/k`：移动选择
- `t`：切换 Stash/QX 目标并重新扫描
- `Enter`：保存当前文件的修复
- `A`：保存所有需要处理的文件
- `b`：切换是否生成 `.bak` 备份，默认开启
- `r`：重新扫描
- `q`：退出

修复时会先删除异常 UUID 节点和对应策略组引用，再补回内置分流分组和规则。

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
```

## 许可证

MIT
