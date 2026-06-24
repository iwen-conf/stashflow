use serde::{Deserialize, Serialize};
use url::Url;
use worker::{
    event, Context, Env, Fetch, Headers, Method, Request, RequestInit, Response, Result, Router,
};

const KV_BINDING: &str = "STASHFLOW_KV";
const SOURCE_KEY: &str = "source";
const SESSION_COOKIE: &str = "sf_session";

const QX_POLICY_LINES: &[&str] = &[
    "static=🛑 广告拦截, reject, direct, ✨ 星链Starlink",
    "static=💬 微信, direct, ✨ 星链Starlink",
    "static=🐧 腾讯服务, direct, ✨ 星链Starlink",
    "static=💰 支付服务, direct, ✨ 星链Starlink",
    "static=🇨🇳 国内流量, direct, ✨ 星链Starlink",
    "static=🤖 AI服务, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct",
    "static=💬 Telegram, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct",
    "static=📺 流媒体, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct",
    "static=🍎 Apple, direct, ✨ 星链Starlink",
    "static=Ⓜ️ Microsoft, direct, ✨ 星链Starlink",
    "static=🎮 游戏平台, direct, ✨ 星链Starlink",
    "static=🌐 国外流量, ✨ 星链Starlink, 🚀 高速优选, 💠 最低延迟, direct",
    "static=🐟 漏网之鱼, ✨ 星链Starlink, direct",
];

const QX_POLICY_NAMES: &[&str] = &[
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
];

const QX_LAZYCAT_RULES: &[&str] = &[
    "HOST-SUFFIX,heiyu.space,direct",
    "HOST-SUFFIX,lazycat.cloud,direct",
    "IP-CIDR,6.6.6.6/32,direct,no-resolve",
    "IP6-CIDR,2000::6666/128,direct,no-resolve",
    "IP6-CIDR,fc03:1136:3800::/40,direct,no-resolve",
];

#[derive(Debug, Clone, Serialize, Deserialize)]
struct SourceConfig {
    name: String,
    url: String,
    updated_at: String,
}

#[derive(Debug, Deserialize)]
struct LoginRequest {
    password: String,
}

#[derive(Debug, Deserialize)]
struct SaveSourceRequest {
    name: Option<String>,
    url: String,
}

#[derive(Debug, Deserialize)]
struct TokenQuery {
    token: Option<String>,
}

#[derive(Debug, Serialize)]
struct ApiMessage {
    ok: bool,
    message: String,
}

#[derive(Debug, Serialize)]
struct StateResponse {
    authenticated: bool,
    source: Option<SourceConfig>,
    qx_url: String,
    stash_url: String,
}

#[event(fetch)]
pub async fn main(req: Request, env: Env, _ctx: Context) -> Result<Response> {
    console_error_panic_hook::set_once();

    Router::new()
        .get_async("/", page)
        .post_async("/api/login", login)
        .post_async("/api/logout", logout)
        .get_async("/api/state", state)
        .put_async("/api/subscription", save_source)
        .delete_async("/api/subscription", delete_source)
        .get_async("/sub/qx", qx_subscription)
        .get_async("/sub/stash", stash_subscription)
        .or_else_any_method_async("/*path", not_found)
        .run(req, env)
        .await
}

async fn page(_req: Request, _ctx: worker::RouteContext<()>) -> Result<Response> {
    no_store(Response::from_html(APP_HTML)?)
}

async fn login(mut req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    let body: LoginRequest = req.json().await?;
    let expected = secret(&ctx.env, "ADMIN_PASSWORD")?;
    if body.password != expected {
        return json_error("密码不正确", 401);
    }

    let session = secret(&ctx.env, "SESSION_TOKEN")?;
    let mut res = Response::from_json(&ApiMessage {
        ok: true,
        message: "已登录".to_string(),
    })?;
    res.headers_mut().set(
        "Set-Cookie",
        &format!(
            "{SESSION_COOKIE}={session}; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=2592000"
        ),
    )?;
    no_store(res)
}

async fn logout(_req: Request, _ctx: worker::RouteContext<()>) -> Result<Response> {
    let mut res = Response::from_json(&ApiMessage {
        ok: true,
        message: "已退出".to_string(),
    })?;
    res.headers_mut().set(
        "Set-Cookie",
        &format!("{SESSION_COOKIE}=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0"),
    )?;
    no_store(res)
}

async fn state(req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    if !is_authenticated(&req, &ctx.env)? {
        return json_error("未登录", 401);
    }
    let source = load_source(&ctx.env).await?;
    let token = secret(&ctx.env, "PUBLIC_TOKEN")?;
    let origin = request_origin(&req)?;
    no_store(Response::from_json(&StateResponse {
        authenticated: true,
        source,
        qx_url: format!("{origin}/sub/qx?token={token}"),
        stash_url: format!("{origin}/sub/stash?token={token}"),
    })?)
}

async fn save_source(mut req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    if !is_authenticated(&req, &ctx.env)? {
        return json_error("未登录", 401);
    }
    let body: SaveSourceRequest = req.json().await?;
    validate_http_url(&body.url).map_err(|message| worker::Error::RustError(message))?;

    let source = SourceConfig {
        name: body
            .name
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| "Default subscription".to_string()),
        url: body.url.trim().to_string(),
        updated_at: js_sys::Date::new_0().to_iso_string().into(),
    };
    save_source_config(&ctx.env, &source).await?;
    no_store(Response::from_json(&source)?)
}

async fn delete_source(req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    if !is_authenticated(&req, &ctx.env)? {
        return json_error("未登录", 401);
    }
    ctx.env.kv(KV_BINDING)?.delete(SOURCE_KEY).await?;
    no_store(Response::from_json(&ApiMessage {
        ok: true,
        message: "已删除订阅源".to_string(),
    })?)
}

async fn qx_subscription(req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    if !has_public_token(&req, &ctx.env)? {
        return Response::error("订阅 token 不正确", 403);
    }
    let Some(source) = load_source(&ctx.env).await? else {
        return Response::error("尚未配置订阅源", 404);
    };
    let upstream = match fetch_subscription(&source.url, "Quantumult X").await {
        Ok(text) => text,
        Err(err) => return Response::error(format!("上游订阅失败: {err}"), 502),
    };
    text_subscription(fix_qx_config(&upstream), "text/plain; charset=utf-8")
}

async fn stash_subscription(req: Request, ctx: worker::RouteContext<()>) -> Result<Response> {
    if !has_public_token(&req, &ctx.env)? {
        return Response::error("订阅 token 不正确", 403);
    }
    let Some(source) = load_source(&ctx.env).await? else {
        return Response::error("尚未配置订阅源", 404);
    };
    let upstream = match fetch_subscription(&source.url, "Stash").await {
        Ok(text) => text,
        Err(err) => return Response::error(format!("上游订阅失败: {err}"), 502),
    };
    text_subscription(ensure_trailing_newline(&upstream), "application/yaml; charset=utf-8")
}

async fn not_found(_req: Request, _ctx: worker::RouteContext<()>) -> Result<Response> {
    Response::error("Not found", 404)
}

async fn load_source(env: &Env) -> Result<Option<SourceConfig>> {
    Ok(env.kv(KV_BINDING)?.get(SOURCE_KEY).json().await?)
}

async fn save_source_config(env: &Env, source: &SourceConfig) -> Result<()> {
    env.kv(KV_BINDING)?
        .put(SOURCE_KEY, serde_json::to_string(source)?)?
        .execute()
        .await?;
    Ok(())
}

async fn fetch_subscription(url: &str, user_agent: &str) -> Result<String> {
    let headers = Headers::new();
    headers.set("User-Agent", user_agent)?;
    headers.set("Accept", "*/*")?;

    let mut init = RequestInit::new();
    init.with_method(Method::Get).with_headers(headers);

    let request = Request::new_with_init(url, &init)?;
    let mut response = Fetch::Request(request).send().await?;
    let status = response.status_code();
    if !(200..300).contains(&status) {
        return Err(worker::Error::RustError(format!(
            "上游订阅返回 HTTP {status}"
        )));
    }

    let text = response.text().await?;
    if text.trim().is_empty() {
        return Err(worker::Error::RustError("上游订阅响应为空".to_string()));
    }
    Ok(text)
}

fn is_authenticated(req: &Request, env: &Env) -> Result<bool> {
    let expected = secret(env, "SESSION_TOKEN")?;
    let cookie = req.headers().get("Cookie")?.unwrap_or_default();
    Ok(cookie_has_value(&cookie, SESSION_COOKIE, &expected))
}

fn has_public_token(req: &Request, env: &Env) -> Result<bool> {
    let query: TokenQuery = req.query()?;
    let expected = secret(env, "PUBLIC_TOKEN")?;
    Ok(query.token.as_deref() == Some(expected.as_str()))
}

fn secret(env: &Env, name: &str) -> Result<String> {
    Ok(env.secret(name)?.to_string())
}

fn request_origin(req: &Request) -> Result<String> {
    let url = req.url()?;
    Ok(format!(
        "{}://{}",
        url.scheme(),
        url.host_str().unwrap_or_default()
    ))
}

fn validate_http_url(value: &str) -> std::result::Result<(), String> {
    let parsed = Url::parse(value.trim()).map_err(|_| "订阅链接格式不正确".to_string())?;
    match parsed.scheme() {
        "http" | "https" => Ok(()),
        _ => Err("订阅链接必须以 http:// 或 https:// 开头".to_string()),
    }
}

fn cookie_has_value(cookie: &str, name: &str, expected: &str) -> bool {
    cookie.split(';').any(|part| {
        let mut pieces = part.trim().splitn(2, '=');
        pieces.next() == Some(name) && pieces.next() == Some(expected)
    })
}

fn json_error(message: &str, status: u16) -> Result<Response> {
    let mut response = Response::from_json(&ApiMessage {
        ok: false,
        message: message.to_string(),
    })?;
    response = response.with_status(status);
    no_store(response)
}

fn no_store(mut response: Response) -> Result<Response> {
    response
        .headers_mut()
        .set("Cache-Control", "no-store, max-age=0")?;
    Ok(response)
}

fn text_subscription(text: String, content_type: &str) -> Result<Response> {
    let mut response = Response::ok(text)?;
    response.headers_mut().set("Content-Type", content_type)?;
    response
        .headers_mut()
        .set("Cache-Control", "no-store, max-age=0")?;
    Ok(response)
}

fn fix_qx_config(text: &str) -> String {
    let trailing = text.ends_with('\n');
    let mut lines: Vec<String> = text.trim_end_matches('\n').split('\n').map(str::to_string).collect();
    if text.is_empty() {
        lines.clear();
    }

    let mut removed_tags = Vec::new();
    lines = remove_qx_hy2_servers(&lines, &mut removed_tags);
    lines = remove_qx_policy_references(&lines, &removed_tags);
    lines = upsert_qx_general(lines);
    lines = upsert_qx_policy(lines);
    lines = upsert_qx_filter_local(lines);

    let mut output = lines.join("\n");
    if trailing || !output.is_empty() {
        output.push('\n');
    }
    output
}

fn remove_qx_hy2_servers(lines: &[String], removed_tags: &mut Vec<String>) -> Vec<String> {
    let Some((start, end)) = qx_section_bounds(lines, "server_local") else {
        return lines.to_vec();
    };

    lines
        .iter()
        .enumerate()
        .filter_map(|(index, line)| {
            if index > start && index < end && is_qx_hy2_line(line) {
                if let Some(tag) = qx_tag(line) {
                    removed_tags.push(tag);
                }
                None
            } else {
                Some(line.clone())
            }
        })
        .collect()
}

fn remove_qx_policy_references(lines: &[String], removed_tags: &[String]) -> Vec<String> {
    if removed_tags.is_empty() {
        return lines.to_vec();
    }
    let Some((start, end)) = qx_section_bounds(lines, "policy") else {
        return lines.to_vec();
    };

    lines
        .iter()
        .enumerate()
        .map(|(index, line)| {
            if index > start && index < end {
                remove_policy_refs_from_line(line, removed_tags)
            } else {
                line.clone()
            }
        })
        .collect()
}

fn remove_policy_refs_from_line(line: &str, removed_tags: &[String]) -> String {
    let trimmed = line.trim();
    if trimmed.is_empty() || trimmed.starts_with('#') || trimmed.starts_with(';') {
        return line.to_string();
    }

    let mut body = line;
    let mut comment = "";
    if let Some(index) = line.find('#') {
        body = &line[..index];
        comment = &line[index..];
    }

    let mut parts = body.split(',').map(str::trim).collect::<Vec<_>>();
    if parts.len() < 2 {
        return line.to_string();
    }

    let first = parts.remove(0).to_string();
    let mut kept = vec![first];
    for part in parts {
        let value = unquote(part);
        if !removed_tags.iter().any(|tag| tag == value) {
            kept.push(part.to_string());
        }
    }

    let mut updated = kept.join(", ");
    if !comment.is_empty() {
        updated.push(' ');
        updated.push_str(comment);
    }
    updated
}

fn upsert_qx_general(mut lines: Vec<String>) -> Vec<String> {
    let additions = [
        ("dns_exclusion_list", vec!["*.heiyu.space", "*.lazycat.cloud"]),
        ("excluded_routes", vec!["6.6.6.6/32", "2000::6666/128"]),
    ];

    if let Some((start, end)) = qx_section_bounds(&lines, "general") {
        for (key, values) in additions {
            upsert_qx_list_setting(&mut lines, start, end, key, &values);
        }
        return lines;
    }

    let mut insert = vec![
        "[general]".to_string(),
        "dns_exclusion_list = *.heiyu.space, *.lazycat.cloud".to_string(),
        "excluded_routes = 6.6.6.6/32, 2000::6666/128".to_string(),
    ];
    insert.extend(lines);
    insert
}

fn upsert_qx_list_setting(
    lines: &mut Vec<String>,
    start: usize,
    end: usize,
    key: &str,
    values: &[&str],
) {
    for index in start + 1..end {
        let trimmed = strip_qx_comment(&lines[index]).trim().to_string();
        let Some((existing_key, existing_value)) = trimmed.split_once('=') else {
            continue;
        };
        if !existing_key.trim().eq_ignore_ascii_case(key) {
            continue;
        }

        let mut merged = existing_value
            .split(',')
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(str::to_string)
            .collect::<Vec<_>>();
        for value in values {
            if !merged.iter().any(|item| item.eq_ignore_ascii_case(value)) {
                merged.push((*value).to_string());
            }
        }
        lines[index] = format!("{key} = {}", merged.join(", "));
        return;
    }

    lines.insert(end, format!("{key} = {}", values.join(", ")));
}

fn upsert_qx_policy(lines: Vec<String>) -> Vec<String> {
    if let Some((start, end)) = qx_section_bounds(&lines, "policy") {
        let mut result = Vec::with_capacity(lines.len() + QX_POLICY_LINES.len());
        result.extend_from_slice(&lines[..=start]);
        for line in &lines[start + 1..end] {
            if !is_managed_qx_policy(line) {
                result.push(line.clone());
            }
        }
        result.extend(QX_POLICY_LINES.iter().map(|line| (*line).to_string()));
        result.extend_from_slice(&lines[end..]);
        return result;
    }

    let insert_at = qx_section_bounds(&lines, "filter_local")
        .map(|(start, _)| start)
        .unwrap_or(lines.len());
    let mut result = Vec::with_capacity(lines.len() + QX_POLICY_LINES.len() + 1);
    result.extend_from_slice(&lines[..insert_at]);
    result.push("[policy]".to_string());
    result.extend(QX_POLICY_LINES.iter().map(|line| (*line).to_string()));
    result.extend_from_slice(&lines[insert_at..]);
    result
}

fn upsert_qx_filter_local(lines: Vec<String>) -> Vec<String> {
    if let Some((start, end)) = qx_section_bounds(&lines, "filter_local") {
        let mut result = Vec::with_capacity(lines.len() + QX_LAZYCAT_RULES.len());
        result.extend_from_slice(&lines[..=start]);
        result.extend(QX_LAZYCAT_RULES.iter().map(|line| (*line).to_string()));
        for line in &lines[start + 1..end] {
            if !is_lazycat_rule(line) {
                result.push(line.clone());
            }
        }
        result.extend_from_slice(&lines[end..]);
        return result;
    }

    let mut result = lines;
    result.push("[filter_local]".to_string());
    result.extend(QX_LAZYCAT_RULES.iter().map(|line| (*line).to_string()));
    result
}

fn qx_section_bounds(lines: &[String], name: &str) -> Option<(usize, usize)> {
    let expected = format!("[{}]", name.to_ascii_lowercase());
    for (index, line) in lines.iter().enumerate() {
        if strip_qx_comment(line).trim().eq_ignore_ascii_case(&expected) {
            let end = lines[index + 1..]
                .iter()
                .position(|line| is_qx_section(line))
                .map(|offset| index + 1 + offset)
                .unwrap_or(lines.len());
            return Some((index, end));
        }
    }
    None
}

fn is_qx_section(line: &str) -> bool {
    let trimmed = strip_qx_comment(line).trim().to_string();
    trimmed.starts_with('[') && trimmed.ends_with(']')
}

fn is_qx_hy2_line(line: &str) -> bool {
    let value = strip_inline_comment(line).trim().to_ascii_lowercase();
    value.starts_with("hysteria2=")
        || value.starts_with("hy2=")
        || value.contains("type=hysteria2")
        || value.contains("type=hy2")
        || value.contains("protocol=hysteria2")
        || value.contains("protocol=hy2")
}

fn qx_tag(line: &str) -> Option<String> {
    strip_inline_comment(line).split(',').find_map(|part| {
        let (key, value) = part.split_once('=')?;
        if key.trim().eq_ignore_ascii_case("tag") {
            Some(unquote(value.trim()).to_string())
        } else {
            None
        }
    })
}

fn is_managed_qx_policy(line: &str) -> bool {
    let trimmed = strip_qx_comment(line).trim().to_string();
    if trimmed.is_empty() || trimmed.starts_with('#') || trimmed.starts_with(';') {
        return false;
    }
    let Some((_, rest)) = trimmed.split_once('=') else {
        return false;
    };
    let name = rest.split(',').next().map(str::trim).unwrap_or_default();
    QX_POLICY_NAMES.iter().any(|managed| *managed == unquote(name))
}

fn is_lazycat_rule(line: &str) -> bool {
    let normalized = strip_qx_comment(line)
        .split(',')
        .map(|part| part.trim().to_ascii_lowercase())
        .collect::<Vec<_>>()
        .join(",");
    QX_LAZYCAT_RULES.iter().any(|rule| {
        rule.split(',')
            .map(|part| part.trim().to_ascii_lowercase())
            .collect::<Vec<_>>()
            .join(",")
            == normalized
    })
}

fn strip_inline_comment(line: &str) -> &str {
    line.split_once('#').map(|(body, _)| body).unwrap_or(line)
}

fn strip_qx_comment(line: &str) -> &str {
    let hash = line.find('#');
    let semicolon = line.find(';');
    match (hash, semicolon) {
        (Some(h), Some(s)) => &line[..h.min(s)],
        (Some(h), None) => &line[..h],
        (None, Some(s)) => &line[..s],
        (None, None) => line,
    }
}

fn unquote(value: &str) -> &str {
    let trimmed = value.trim();
    if trimmed.len() >= 2 {
        let first = trimmed.as_bytes()[0];
        let last = trimmed.as_bytes()[trimmed.len() - 1];
        if first == last && (first == b'\'' || first == b'"') {
            return &trimmed[1..trimmed.len() - 1];
        }
    }
    trimmed
}

fn ensure_trailing_newline(text: &str) -> String {
    if text.ends_with('\n') {
        text.to_string()
    } else {
        format!("{text}\n")
    }
}

const APP_HTML: &str = r#"<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>StashFlow Edge</title>
  <style>
    :root {
      color-scheme: light;
      --bg: oklch(0.985 0 0);
      --surface: oklch(0.955 0.004 242);
      --surface-strong: oklch(0.925 0.010 242);
      --ink: oklch(0.185 0.018 248);
      --muted: oklch(0.455 0.022 248);
      --primary: oklch(0.515 0.140 242);
      --primary-strong: oklch(0.430 0.135 242);
      --accent: oklch(0.620 0.145 168);
      --danger: oklch(0.550 0.170 28);
      --line: oklch(0.875 0.008 242);
      --focus: oklch(0.690 0.130 242);
      --white: oklch(1 0 0);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--ink);
      font: 450 14px/1.55 Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    button, input { font: inherit; }
    .shell {
      min-height: 100vh;
      display: grid;
      grid-template-rows: auto 1fr;
    }
    header {
      border-bottom: 1px solid var(--line);
      background: var(--white);
    }
    .bar {
      max-width: 1120px;
      margin: 0 auto;
      padding: 18px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      min-width: 0;
    }
    .mark {
      width: 34px;
      height: 34px;
      border-radius: 9px;
      background: var(--primary);
      color: var(--white);
      display: grid;
      place-items: center;
      font-weight: 800;
      letter-spacing: 0;
    }
    h1 {
      margin: 0;
      font-size: 18px;
      line-height: 1.2;
      letter-spacing: 0;
    }
    .sub {
      margin: 2px 0 0;
      color: var(--muted);
      font-size: 12px;
    }
    main {
      width: min(1120px, 100%);
      margin: 0 auto;
      padding: 30px 24px 56px;
    }
    .grid {
      display: grid;
      grid-template-columns: minmax(0, 0.92fr) minmax(340px, 1.08fr);
      gap: 18px;
      align-items: start;
    }
    .panel {
      background: var(--white);
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 18px;
    }
    .panel h2 {
      margin: 0 0 14px;
      font-size: 16px;
      line-height: 1.3;
      letter-spacing: 0;
    }
    .field { display: grid; gap: 7px; margin-bottom: 13px; }
    label {
      color: var(--muted);
      font-size: 12px;
      font-weight: 650;
    }
    input {
      width: 100%;
      min-height: 42px;
      padding: 9px 11px;
      border: 1px solid var(--line);
      border-radius: 10px;
      color: var(--ink);
      background: var(--bg);
      outline: none;
    }
    input:focus {
      border-color: var(--focus);
      box-shadow: 0 0 0 3px oklch(0.690 0.130 242 / 0.18);
    }
    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 12px;
      line-height: 1.45;
    }
    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 16px;
    }
    button {
      min-height: 38px;
      border: 1px solid transparent;
      border-radius: 10px;
      padding: 8px 13px;
      cursor: pointer;
      transition: background 160ms ease, border-color 160ms ease, transform 160ms ease;
    }
    button:focus-visible {
      outline: 3px solid oklch(0.690 0.130 242 / 0.26);
      outline-offset: 2px;
    }
    button:hover { transform: translateY(-1px); }
    .primary { background: var(--primary); color: var(--white); }
    .primary:hover { background: var(--primary-strong); }
    .secondary { background: var(--white); border-color: var(--line); color: var(--ink); }
    .danger { background: transparent; color: var(--danger); }
    .logout { display: none; }
    .status {
      margin-top: 12px;
      min-height: 22px;
      color: var(--muted);
    }
    .status.error { color: var(--danger); }
    .status.ok { color: var(--success); }
    .empty {
      padding: 18px;
      background: var(--surface);
      border-radius: 12px;
      color: var(--muted);
    }
    .source {
      display: grid;
      gap: 8px;
      padding: 14px;
      background: var(--surface);
      border-radius: 12px;
    }
    .source strong { font-size: 14px; }
    .source span { color: var(--muted); overflow-wrap: anywhere; }
    .endpoint {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 10px;
      align-items: center;
      padding: 12px;
      border: 1px solid var(--line);
      border-radius: 12px;
      margin-top: 10px;
    }
    .endpoint code {
      color: var(--ink);
      overflow-wrap: anywhere;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      min-height: 24px;
      padding: 3px 8px;
      border-radius: 999px;
      background: oklch(0.620 0.145 168 / 0.13);
      color: oklch(0.365 0.115 168);
      font-size: 12px;
      font-weight: 700;
    }
    .login {
      max-width: 430px;
      margin: 10vh auto 0;
    }
    .hidden { display: none !important; }
    @media (max-width: 820px) {
      .bar { padding: 16px; }
      main { padding: 20px 16px 44px; }
      .grid { grid-template-columns: 1fr; }
      .endpoint { grid-template-columns: 1fr; }
    }
    @media (prefers-reduced-motion: reduce) {
      *, *::before, *::after { transition: none !important; }
      button:hover { transform: none; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header>
      <div class="bar">
        <div class="brand">
          <div class="mark">SF</div>
          <div>
            <h1>StashFlow Edge</h1>
            <p class="sub">私有订阅修复面板</p>
          </div>
        </div>
        <button id="logout" class="secondary logout" type="button">退出</button>
      </div>
    </header>
    <main>
      <section id="login-view" class="panel login">
        <h2>登录</h2>
        <form id="login-form">
          <div class="field">
            <label for="password">访问密码</label>
            <input id="password" name="password" type="password" autocomplete="current-password" required>
          </div>
          <div class="actions">
            <button class="primary" type="submit">进入面板</button>
          </div>
          <div id="login-status" class="status"></div>
        </form>
      </section>

      <section id="app-view" class="grid hidden">
        <div class="panel">
          <h2>订阅源</h2>
          <form id="source-form">
            <div class="field">
              <label for="source-name">名称</label>
              <input id="source-name" name="name" placeholder="Starlink">
            </div>
            <div class="field">
              <label for="source-url">上游订阅链接</label>
              <input id="source-url" class="mono" name="url" placeholder="https://example.com/v1/subscribe?token=..." required>
            </div>
            <div class="actions">
              <button class="primary" type="submit">保存订阅源</button>
              <button id="delete-source" class="danger" type="button">删除</button>
            </div>
            <div id="source-status" class="status"></div>
          </form>
        </div>

        <div class="panel">
          <h2>输出地址</h2>
          <div id="source-current" class="empty">还没有订阅源。保存后这里会出现 QX 和 Stash 地址。</div>
          <div id="endpoints" class="hidden">
            <div class="source">
              <strong id="current-name"></strong>
              <span id="current-url" class="mono"></span>
              <span id="current-time"></span>
            </div>
            <div class="endpoint">
              <div>
                <span class="badge">QX</span>
                <code id="qx-url" class="mono"></code>
              </div>
              <button class="secondary copy" type="button" data-target="qx-url">复制</button>
            </div>
            <div class="endpoint">
              <div>
                <span class="badge">Stash</span>
                <code id="stash-url" class="mono"></code>
              </div>
              <button class="secondary copy" type="button" data-target="stash-url">复制</button>
            </div>
          </div>
          <div id="copy-status" class="status"></div>
        </div>
      </section>
    </main>
  </div>

  <script>
    const loginView = document.querySelector('#login-view');
    const appView = document.querySelector('#app-view');
    const logoutBtn = document.querySelector('#logout');
    const loginStatus = document.querySelector('#login-status');
    const sourceStatus = document.querySelector('#source-status');
    const copyStatus = document.querySelector('#copy-status');

    async function api(path, options = {}) {
      const res = await fetch(path, {
        headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
        ...options
      });
      const text = await res.text();
      const data = text ? JSON.parse(text) : null;
      if (!res.ok) throw new Error(data?.message || text || `HTTP ${res.status}`);
      return data;
    }

    function setStatus(el, message, kind = '') {
      el.textContent = message || '';
      el.className = `status ${kind}`.trim();
    }

    function showLogin() {
      loginView.classList.remove('hidden');
      appView.classList.add('hidden');
      logoutBtn.style.display = 'none';
    }

    function showApp() {
      loginView.classList.add('hidden');
      appView.classList.remove('hidden');
      logoutBtn.style.display = 'inline-flex';
    }

    async function loadState() {
      try {
        const state = await api('/api/state');
        showApp();
        renderState(state);
      } catch {
        showLogin();
      }
    }

    function renderState(state) {
      const empty = document.querySelector('#source-current');
      const endpoints = document.querySelector('#endpoints');
      if (!state.source) {
        empty.classList.remove('hidden');
        endpoints.classList.add('hidden');
        return;
      }
      empty.classList.add('hidden');
      endpoints.classList.remove('hidden');
      document.querySelector('#source-name').value = state.source.name;
      document.querySelector('#source-url').value = state.source.url;
      document.querySelector('#current-name').textContent = state.source.name;
      document.querySelector('#current-url').textContent = state.source.url;
      document.querySelector('#current-time').textContent = `更新于 ${state.source.updated_at}`;
      document.querySelector('#qx-url').textContent = state.qx_url;
      document.querySelector('#stash-url').textContent = state.stash_url;
    }

    document.querySelector('#login-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      setStatus(loginStatus, '正在登录...');
      try {
        await api('/api/login', {
          method: 'POST',
          body: JSON.stringify({ password: document.querySelector('#password').value })
        });
        setStatus(loginStatus, '');
        await loadState();
      } catch (error) {
        setStatus(loginStatus, error.message, 'error');
      }
    });

    document.querySelector('#source-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      setStatus(sourceStatus, '正在保存...');
      try {
        await api('/api/subscription', {
          method: 'PUT',
          body: JSON.stringify({
            name: document.querySelector('#source-name').value,
            url: document.querySelector('#source-url').value
          })
        });
        setStatus(sourceStatus, '已保存', 'ok');
        await loadState();
      } catch (error) {
        setStatus(sourceStatus, error.message, 'error');
      }
    });

    document.querySelector('#delete-source').addEventListener('click', async () => {
      setStatus(sourceStatus, '正在删除...');
      try {
        await api('/api/subscription', { method: 'DELETE' });
        document.querySelector('#source-name').value = '';
        document.querySelector('#source-url').value = '';
        setStatus(sourceStatus, '已删除', 'ok');
        await loadState();
      } catch (error) {
        setStatus(sourceStatus, error.message, 'error');
      }
    });

    logoutBtn.addEventListener('click', async () => {
      await api('/api/logout', { method: 'POST' });
      showLogin();
    });

    document.querySelectorAll('.copy').forEach((button) => {
      button.addEventListener('click', async () => {
        const target = document.querySelector(`#${button.dataset.target}`);
        await navigator.clipboard.writeText(target.textContent);
        setStatus(copyStatus, '已复制', 'ok');
      });
    });

    loadState();
  </script>
</body>
</html>"#;

#[cfg(test)]
mod tests {
    use super::*;
    use pretty_assertions::assert_eq;

    #[test]
    fn qx_fix_removes_hy2_and_adds_lazycat_rules() {
        let input = [
            "[general]",
            "server_check_url=http://example.com",
            "[server_local]",
            "trojan=example.com:443, password=p, tag=Keep",
            "hysteria2=hy2.example.com:443, password=p, tag=Hy2 One",
            "[policy]",
            "static=Proxy, Keep, Hy2 One, direct",
            "[filter_local]",
            "HOST-SUFFIX,old.example,direct",
            "",
        ]
        .join("\n");

        let output = fix_qx_config(&input);

        assert!(!output.contains("hysteria2="));
        assert!(!output.contains("Hy2 One"));
        assert!(output.contains("static=Proxy, Keep, direct"));
        assert!(output.contains("dns_exclusion_list = *.heiyu.space, *.lazycat.cloud"));
        assert!(output.contains("excluded_routes = 6.6.6.6/32, 2000::6666/128"));
        assert!(output.contains("HOST-SUFFIX,lazycat.cloud,direct"));
        assert!(output.contains("HOST-SUFFIX,old.example,direct"));
    }

    #[test]
    fn cookie_match_is_exact() {
        assert!(cookie_has_value("a=1; sf_session=abc; z=9", "sf_session", "abc"));
        assert!(!cookie_has_value("sf_session=abcd", "sf_session", "abc"));
    }

    #[test]
    fn validates_http_urls_only() {
        assert!(validate_http_url("https://example.com/sub").is_ok());
        assert!(validate_http_url("file:///tmp/sub").is_err());
    }

    #[test]
    fn leaves_stash_text_with_newline() {
        assert_eq!(ensure_trailing_newline("proxies: []"), "proxies: []\n");
    }
}
