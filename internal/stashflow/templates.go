package stashflow

import "strings"

var StashSplitGroupNames = []string{
	"🛑 广告拦截",
	"💬 微信",
	"🐧 腾讯服务",
	"💰 支付服务",
	"🇨🇳 国内流量",
	"🤖 AI服务",
	"💬 Telegram",
	"📺 流媒体",
	"🍎 Apple",
	"Ⓜ️ Microsoft",
	"🎮 游戏平台",
	"🌐 国外流量",
	"🐟 漏网之鱼",
}

var StashSplitGroupLines = splitTemplate(`
- name: 🛑 广告拦截
  type: select
  proxies:
    - REJECT
    - DIRECT
    - ✨ 星链Starlink
- name: 💬 微信
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 🐧 腾讯服务
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 💰 支付服务
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 🇨🇳 国内流量
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 🤖 AI服务
  type: select
  proxies:
    - ✨ 星链Starlink
    - 🚀 高速优选
    - 💠 最低延迟
    - DIRECT
- name: 💬 Telegram
  type: select
  proxies:
    - ✨ 星链Starlink
    - 🚀 高速优选
    - 💠 最低延迟
    - DIRECT
- name: 📺 流媒体
  type: select
  proxies:
    - ✨ 星链Starlink
    - 🚀 高速优选
    - 💠 最低延迟
    - DIRECT
- name: 🍎 Apple
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: Ⓜ️ Microsoft
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 🎮 游戏平台
  type: select
  proxies:
    - DIRECT
    - ✨ 星链Starlink
- name: 🌐 国外流量
  type: select
  proxies:
    - ✨ 星链Starlink
    - 🚀 高速优选
    - 💠 最低延迟
    - DIRECT
- name: 🐟 漏网之鱼
  type: select
  proxies:
    - ✨ 星链Starlink
    - DIRECT
`)

var StashRuleLines = splitTemplate(`
- 'IP-CIDR,127.0.0.0/8,DIRECT'
- 'IP-CIDR,172.16.0.0/12,DIRECT'
- 'IP-CIDR,192.168.0.0/16,DIRECT'
- 'IP-CIDR,10.0.0.0/8,DIRECT'
- 'IP-CIDR,100.64.0.0/10,DIRECT'
- 'IP-CIDR,224.0.0.0/4,DIRECT'
- 'IP-CIDR6,fe80::/10,DIRECT'
- 'DOMAIN-SUFFIX,local,DIRECT'
- 'DOMAIN-SUFFIX,lan,DIRECT'
- 'DOMAIN-SUFFIX,localdomain,DIRECT'
- 'DOMAIN-SUFFIX,arpa,DIRECT'
- 'DOMAIN-SUFFIX,googlesyndication.com,🛑 广告拦截'
- 'DOMAIN-SUFFIX,googleadservices.com,🛑 广告拦截'
- 'DOMAIN-SUFFIX,doubleclick.net,🛑 广告拦截'
- 'DOMAIN-SUFFIX,adnxs.com,🛑 广告拦截'
- 'DOMAIN-SUFFIX,ads-twitter.com,🛑 广告拦截'
- 'DOMAIN-SUFFIX,analytics.google.com,🛑 广告拦截'
- 'DOMAIN-KEYWORD,adservice,🛑 广告拦截'
- 'DOMAIN-SUFFIX,xship.top,✨ 星链Starlink'
- 'DOMAIN-SUFFIX,2513142.xyz,DIRECT'
- 'DOMAIN-SUFFIX,spaceship.com,DIRECT'
- 'DOMAIN-SUFFIX,itdog.cn,✨ 星链Starlink'
- 'DOMAIN-SUFFIX,wechat.com,💬 微信'
- 'DOMAIN-SUFFIX,wechatapp.com,💬 微信'
- 'DOMAIN-SUFFIX,weixin.qq.com,💬 微信'
- 'DOMAIN-SUFFIX,wx.qq.com,💬 微信'
- 'DOMAIN-SUFFIX,servicewechat.com,💬 微信'
- 'DOMAIN-SUFFIX,weixinbridge.com,💬 微信'
- 'DOMAIN-SUFFIX,qpic.cn,💬 微信'
- 'DOMAIN-SUFFIX,qlogo.cn,💬 微信'
- 'DOMAIN-SUFFIX,tenpay.com,💰 支付服务'
- 'DOMAIN-SUFFIX,wechatpay.com,💰 支付服务'
- 'DOMAIN-SUFFIX,alipay.com,💰 支付服务'
- 'DOMAIN-SUFFIX,alipayobjects.com,💰 支付服务'
- 'DOMAIN-SUFFIX,antgroup.com,💰 支付服务'
- 'DOMAIN-SUFFIX,95516.com,💰 支付服务'
- 'DOMAIN-SUFFIX,unionpay.com,💰 支付服务'
- 'DOMAIN-SUFFIX,unionpayintl.com,💰 支付服务'
- 'DOMAIN-SUFFIX,qq.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,gtimg.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,idqqimg.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,imqq.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,myapp.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,tencent.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,tencent-cloud.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,qcloud.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,myqcloud.com,🐧 腾讯服务'
- 'DOMAIN-SUFFIX,baidu.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,bdstatic.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,bcebos.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,taobao.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,tmall.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,tbcdn.cn,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,alicdn.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,aliyun.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,jd.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,360buyimg.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,meituan.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,dianping.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,amap.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,gaode.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,bilibili.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,hdslb.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,bilivideo.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,douyin.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,byteimg.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,snssdk.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,toutiao.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,ixigua.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,pstatp.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,xiaohongshu.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,xhscdn.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,zhihu.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,zhimg.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,kuaishou.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,gifshow.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,ksapisrv.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,163.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,126.net,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,netease.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,weibo.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,sina.com.cn,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,xiaomi.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,mi.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,miui.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,huawei.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,honor.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,oppo.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,vivo.com,🇨🇳 国内流量'
- 'DOMAIN-SUFFIX,openai.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,chatgpt.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,oaistatic.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,oaiusercontent.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,ai.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,anthropic.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,claude.ai,🤖 AI服务'
- 'DOMAIN-SUFFIX,perplexity.ai,🤖 AI服务'
- 'DOMAIN-SUFFIX,poe.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,deepseek.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,groq.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,gemini.google.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,generativelanguage.googleapis.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,copilot.microsoft.com,🤖 AI服务'
- 'DOMAIN-SUFFIX,telegram.org,💬 Telegram'
- 'DOMAIN-SUFFIX,t.me,💬 Telegram'
- 'DOMAIN-SUFFIX,tdesktop.com,💬 Telegram'
- 'IP-CIDR,91.108.4.0/22,💬 Telegram,no-resolve'
- 'IP-CIDR,91.108.8.0/21,💬 Telegram,no-resolve'
- 'IP-CIDR,91.108.16.0/22,💬 Telegram,no-resolve'
- 'IP-CIDR,91.108.56.0/22,💬 Telegram,no-resolve'
- 'IP-CIDR,149.154.160.0/20,💬 Telegram,no-resolve'
- 'IP-CIDR6,2001:67c:4e8::/48,💬 Telegram,no-resolve'
- 'IP-CIDR6,2001:b28:f23d::/48,💬 Telegram,no-resolve'
- 'IP-CIDR6,2001:b28:f23f::/48,💬 Telegram,no-resolve'
- 'DOMAIN-SUFFIX,netflix.com,📺 流媒体'
- 'DOMAIN-SUFFIX,netflix.net,📺 流媒体'
- 'DOMAIN-SUFFIX,nflxext.com,📺 流媒体'
- 'DOMAIN-SUFFIX,nflximg.com,📺 流媒体'
- 'DOMAIN-SUFFIX,nflxso.net,📺 流媒体'
- 'DOMAIN-SUFFIX,nflxvideo.net,📺 流媒体'
- 'DOMAIN-SUFFIX,youtube.com,📺 流媒体'
- 'DOMAIN-SUFFIX,youtu.be,📺 流媒体'
- 'DOMAIN-SUFFIX,googlevideo.com,📺 流媒体'
- 'DOMAIN-SUFFIX,ytimg.com,📺 流媒体'
- 'DOMAIN-SUFFIX,disneyplus.com,📺 流媒体'
- 'DOMAIN-SUFFIX,disney-plus.net,📺 流媒体'
- 'DOMAIN-SUFFIX,hulu.com,📺 流媒体'
- 'DOMAIN-SUFFIX,huluim.com,📺 流媒体'
- 'DOMAIN-SUFFIX,hbomax.com,📺 流媒体'
- 'DOMAIN-SUFFIX,max.com,📺 流媒体'
- 'DOMAIN-SUFFIX,primevideo.com,📺 流媒体'
- 'DOMAIN-SUFFIX,spotify.com,📺 流媒体'
- 'DOMAIN-SUFFIX,twitch.tv,📺 流媒体'
- 'DOMAIN-SUFFIX,ttvnw.net,📺 流媒体'
- 'DOMAIN-SUFFIX,apple.com,🍎 Apple'
- 'DOMAIN-SUFFIX,icloud.com,🍎 Apple'
- 'DOMAIN-SUFFIX,icloud-content.com,🍎 Apple'
- 'DOMAIN-SUFFIX,mzstatic.com,🍎 Apple'
- 'IP-CIDR,17.0.0.0/8,🍎 Apple,no-resolve'
- 'DOMAIN-SUFFIX,microsoft.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,windowsupdate.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,office.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,office365.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,live.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,msn.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,bing.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,skype.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,xboxlive.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,msftconnecttest.com,Ⓜ️ Microsoft'
- 'DOMAIN-SUFFIX,msftncsi.com,Ⓜ️ Microsoft'
- 'DOMAIN,fastly-download.epicgames.com,DIRECT'
- 'DOMAIN,epicgames-download1.akamaized.net,DIRECT'
- 'DOMAIN,steamcdn-a.akamaihd.net,DIRECT'
- 'DOMAIN-SUFFIX,steamserver.net,🎮 游戏平台'
- 'DOMAIN-SUFFIX,steampowered.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,steamcommunity.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,steamstatic.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,epicgames.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,unrealengine.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,battle.net,🎮 游戏平台'
- 'DOMAIN-SUFFIX,blizzard.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,playstation.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,nintendo.com,🎮 游戏平台'
- 'DOMAIN-SUFFIX,google.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,googleapis.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,gstatic.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,googleusercontent.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,ggpht.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,gmail.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,github.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,githubusercontent.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,githubassets.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,gitlab.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,twitter.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,x.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,t.co,🌐 国外流量'
- 'DOMAIN-SUFFIX,facebook.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,fbcdn.net,🌐 国外流量'
- 'DOMAIN-SUFFIX,instagram.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,whatsapp.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,discord.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,discord.gg,🌐 国外流量'
- 'DOMAIN-SUFFIX,discordapp.net,🌐 国外流量'
- 'DOMAIN-SUFFIX,reddit.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,tiktok.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,tiktokcdn.com,🌐 国外流量'
- 'DOMAIN-SUFFIX,cn,🇨🇳 国内流量'
- 'DOMAIN-KEYWORD,-cn,🇨🇳 国内流量'
- 'GEOIP,CN,🇨🇳 国内流量'
- 'MATCH,🐟 漏网之鱼'
`)

var QXSplitGroupNames = StashSplitGroupNames

var QXPolicyLines = splitTemplate(`
static=🛑 广告拦截, reject, direct, ✨ 星链Starlink
static=💬 微信, direct, ✨ 星链Starlink
static=🐧 腾讯服务, direct, ✨ 星链Starlink
static=💰 支付服务, direct, ✨ 星链Starlink
static=🇨🇳 国内流量, direct, ✨ 星链Starlink
static=🤖 AI服务, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct
static=💬 Telegram, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct
static=📺 流媒体, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct
static=🍎 Apple, direct, ✨ 星链Starlink
static=Ⓜ️ Microsoft, direct, ✨ 星链Starlink
static=🎮 游戏平台, direct, ✨ 星链Starlink
static=🌐 国外流量, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct
static=🐟 漏网之鱼, ✨ 星链Starlink, direct
`)

var QXLazycatDNSExclusionValues = []string{
	"*.heiyu.space",
	"*.lazycat.cloud",
}

var QXLazycatExcludedRouteValues = []string{
	"6.6.6.6/32",
	"2000::6666/128",
}

var QXLazycatRuleLines = splitTemplate(`
HOST-SUFFIX,heiyu.space,direct
HOST-SUFFIX,lazycat.cloud,direct
IP-CIDR,6.6.6.6/32,direct,no-resolve
IP6-CIDR,2000::6666/128,direct,no-resolve
IP6-CIDR,fc03:1136:3800::/40,direct,no-resolve
`)

var QXRuleLines = append(QXLazycatRuleLines, qxRuleLinesFromStash(StashRuleLines)...)

func splitTemplate(value string) []string {
	value = strings.Trim(value, "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}
