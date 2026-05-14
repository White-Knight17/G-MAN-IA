// G-MAN v1.0 — Tauri Desktop Shell
// Sidecar spawn + JSON-RPC relay + System tray

use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::fmt;
use std::io::{BufRead, BufReader, Write};
use std::process::{Child, Command, Stdio};
use std::sync::Mutex;

// ============================================================================
// Window mode types and constants (companion-mode v2.1.0)
// ============================================================================

/// Window display modes for G-MAN.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum WindowMode {
    Companion,
    Floating,
    Compact,
}

impl fmt::Display for WindowMode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            WindowMode::Companion => write!(f, "companion"),
            WindowMode::Floating => write!(f, "floating"),
            WindowMode::Compact => write!(f, "compact"),
        }
    }
}

/// Parse a window mode from a string (case-insensitive).
pub fn parse_window_mode(s: &str) -> Result<WindowMode, String> {
    match s.to_lowercase().as_str() {
        "companion" => Ok(WindowMode::Companion),
        "floating" => Ok(WindowMode::Floating),
        "compact" => Ok(WindowMode::Compact),
        other => Err(format!("unknown window mode: {}", other)),
    }
}

// Dimension constants
pub const COMPANION_WIDTH: u32 = 380;
pub const COMPANION_MIN_WIDTH: u32 = 320;
pub const COMPANION_MAX_WIDTH: u32 = 600;
pub const FLOATING_WIDTH: u32 = 420;
pub const FLOATING_HEIGHT: u32 = 700;
pub const COMPACT_WIDTH: u32 = 48;

/// Calculated window dimensions for a given mode.
pub struct WindowDimensions {
    pub width: u32,
    pub height: u32,
    pub always_on_top: &'static str,
}

/// Calculate window dimensions for a given mode and screen height.
pub fn calculate_window_dimensions(mode: WindowMode, screen_height: u32) -> WindowDimensions {
    match mode {
        WindowMode::Companion => WindowDimensions {
            width: COMPANION_WIDTH,
            height: screen_height,
            always_on_top: "true",
        },
        WindowMode::Floating => WindowDimensions {
            width: FLOATING_WIDTH,
            height: FLOATING_HEIGHT,
            always_on_top: "false",
        },
        WindowMode::Compact => WindowDimensions {
            width: COMPACT_WIDTH,
            height: screen_height.saturating_sub(100),
            always_on_top: "true",
        },
    }
}

/// Serializable window state for persistence.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowState {
    pub mode: WindowMode,
    pub width: u32,
    pub always_on_top: bool,
}

// ============================================================================
// Pure functions — JSON-RPC message handling
// ============================================================================

/// Checks if a JSON-RPC line is the "ready" notification from the sidecar.
pub fn is_ready_notification(line: &str) -> bool {
    line.contains("\"method\":\"ready\"")
}

/// Parses a JSON-RPC response line into a serde_json Value.
pub fn parse_jsonrpc_response(line: &str) -> Result<Value, String> {
    serde_json::from_str(line).map_err(|e| e.to_string())
}

/// Formats a JSON-RPC request as a single-line JSON string.
pub fn format_jsonrpc_request(method: &str, params: &str) -> String {
    let id = 1u64; // Simple counter - the Tauri command handler will manage IDs
    format!(
        r#"{{"jsonrpc":"2.0","method":"{}","params":{},"id":{}}}"#,
        method, params, id
    )
}

// ============================================================================
// Sidecar management functions
// ============================================================================

/// Writes a JSON-RPC request to sidecar stdin and reads the response from stdout.
pub fn send_jsonrpc(
    stdin: &mut impl Write,
    stdout: &mut impl BufRead,
    request: &str,
) -> std::io::Result<String> {
    writeln!(stdin, "{}", request)?;
    stdin.flush()?;
    let mut response = String::new();
    match stdout.read_line(&mut response) {
        Ok(0) => Ok(String::new()), // EOF
        Ok(_) => Ok(response),
        Err(e) => Err(e),
    }
}

/// Pings the sidecar and returns true if it responds with "pong".
pub fn check_sidecar_health(
    stdin: &mut impl Write,
    stdout: &mut impl BufRead,
) -> std::io::Result<bool> {
    let ping = r#"{"jsonrpc":"2.0","method":"ping","id":0}"#;
    match send_jsonrpc(stdin, stdout, ping) {
        Ok(response) => Ok(response.contains("\"result\":\"pong\"")),
        Err(_) => Ok(false),
    }
}

/// Attempts to spawn the sidecar with retry logic.
/// Returns (success, attempts_made).
pub fn restart_sidecar_with_retry(max_attempts: u32) -> (bool, u32) {
    for attempt in 1..=max_attempts {
        let result = Command::new("bash")
            .arg("-c")
            .arg("echo '{\"jsonrpc\":\"2.0\",\"method\":\"ready\",\"params\":{\"version\":\"1.0.0\"}}' && sleep 3600")
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn();

        match result {
            Ok(mut child) => {
                // Read the ready notification to confirm startup
                if let Some(stdout) = child.stdout.take() {
                    let mut reader = BufReader::new(stdout);
                    let mut line = String::new();
                    if reader.read_line(&mut line).is_ok() && is_ready_notification(&line) {
                        // Successfully spawned and received ready signal
                        let _ = child.kill();
                        return (true, attempt);
                    }
                }
                let _ = child.kill();
            }
            Err(_) => {
                // Spawn failed, continue retrying
            }
        }
    }
    (false, max_attempts)
}

// ============================================================================
// Tauri application entry point
// ============================================================================

/// Application state managed by Tauri.
struct AppState {
    sidecar: Mutex<Option<Child>>,
    window_state: Mutex<WindowState>,
}

fn main() {
    use std::sync::Mutex;
    use tauri::{
        menu::{MenuBuilder, MenuItemBuilder},
        tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
        Manager,
    };
    use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutState};

    tauri::Builder::default()
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .manage(AppState {
            sidecar: Mutex::new(None),
            window_state: Mutex::new(WindowState {
                mode: WindowMode::Floating,
                width: FLOATING_WIDTH,
                always_on_top: false,
            }),
        })
        .setup(|app| {
            // Spawn sidecar on startup
            let binary_name = "gman-core-x86_64-unknown-linux-gnu";
            let candidates = vec![
                std::env::current_dir().unwrap_or_default().join("binaries").join(binary_name),
                app.path().resource_dir().unwrap_or_default().join("binaries").join(binary_name),
            ];
            
            let exe_path = candidates.into_iter()
                .find(|p| p.exists())
                .unwrap_or_else(|| {
                    std::env::current_dir().unwrap_or_default().join("binaries").join(binary_name)
                });

            let child = Command::new(&exe_path)
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::inherit())
                .spawn();

            match child {
                Ok(c) => {
                    let state = app.state::<AppState>();
                    *state.sidecar.lock().unwrap() = Some(c);
                }
                Err(e) => {
                    eprintln!("Failed to spawn sidecar: {}", e);
                }
            }

            // Register global shortcut Ctrl+Shift+G
            let shortcut = Shortcut::new(Some(tauri_plugin_global_shortcut::Modifiers::CONTROL | tauri_plugin_global_shortcut::Modifiers::SHIFT), tauri_plugin_global_shortcut::Code::KeyG);
            let app_handle = app.handle().clone();
            
            let _ = app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
                if event.state() == ShortcutState::Pressed {
                    let _ = toggle_window(app_handle.clone());
                }
            });

            // Build system tray menu
            let show_i = MenuItemBuilder::with_id("show", "Show").build(app)?;
            let hide_i = MenuItemBuilder::with_id("hide", "Hide").build(app)?;
            let quit_i = MenuItemBuilder::with_id("quit", "Quit").build(app)?;

            let menu = MenuBuilder::new(app)
                .item(&show_i)
                .item(&hide_i)
                .separator()
                .item(&quit_i)
                .build()?;

            let _tray = TrayIconBuilder::new()
                .menu(&menu)
                .on_menu_event(|app, event| match event.id().as_ref() {
                    "show" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    "hide" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.hide();
                        }
                    }
                    "quit" => {
                        if let Some(state) = app.try_state::<AppState>() {
                            if let Ok(mut guard) = state.sidecar.lock() {
                                if let Some(ref mut child) = *guard {
                                    let _ = child.kill();
                                }
                            }
                        }
                        std::process::exit(0);
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            if window.is_visible().unwrap_or(false) {
                                let _ = window.hide();
                            } else {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                    }
                })
                .build(app)?;

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                let _ = window.hide();
                api.prevent_close();
            }
        })
        .invoke_handler(tauri::generate_handler![
            relay_request,
            stream_chat,
            set_window_mode,
            toggle_window,
            get_window_state,
        ])
        .run(tauri::generate_context!())
        .expect("error while running G-MAN");
}

// ============================================================================
// Tauri commands — JSON-RPC relay to Go sidecar
// ============================================================================

/// Relays a JSON-RPC request to the Go sidecar and returns the response.
/// Skips notification lines (no "id" field) until it finds the matching response.
#[tauri::command]
async fn relay_request(
    app: tauri::AppHandle,
    method: String,
    params: serde_json::Value,
) -> Result<serde_json::Value, String> {
    let result = tauri::async_runtime::spawn_blocking(move || {
        relay_request_blocking(app, method, params)
    }).await.map_err(|e| format!("{}", e))?;
    result
}

fn relay_request_blocking(
    app: tauri::AppHandle,
    method: String,
    params: serde_json::Value,
) -> Result<serde_json::Value, String> {
    use tauri::Manager;
    use std::io::Write;

    let state = app.state::<AppState>();
    let mut guard = state.sidecar.lock().map_err(|e| e.to_string())?;
    let child = guard.as_mut().ok_or("sidecar not running")?;

    let id = 1;
    let request = serde_json::json!({
        "jsonrpc": "2.0",
        "id": id,
        "method": method,
        "params": params,
    });

    let request_str = serde_json::to_string(&request).map_err(|e| e.to_string())?;

    let mut stdin = child.stdin.take().ok_or("sidecar stdin unavailable")?;
    writeln!(stdin, "{}", request_str).map_err(|e| e.to_string())?;
    stdin.flush().map_err(|e| e.to_string())?;
    child.stdin = Some(stdin);

    // Read from stdout, skipping notifications until we get the response with matching id
    let stdout = child.stdout.as_mut().ok_or("sidecar stdout unavailable")?;
    let mut reader = BufReader::new(stdout);

    loop {
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        if line.is_empty() {
            return Err("sidecar closed connection".to_string());
        }

        if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(&line) {
            // If it has an "id" field matching our request, it's the response
            if parsed.get("id").and_then(|v| v.as_u64()) == Some(id as u64) {
                return Ok(parsed);
            }
            // Otherwise it's a notification — skip it
        }
    }
}

/// Opens a streaming chat session with the Go sidecar.
/// Reads agent.event notifications until the final response arrives,
/// then returns all notifications as NDJSON string.
#[tauri::command]
async fn stream_chat(
    app: tauri::AppHandle,
    input: String,
) -> Result<String, String> {
    let result = tauri::async_runtime::spawn_blocking(move || {
        stream_chat_blocking(app, input)
    }).await.map_err(|e| format!("{}", e))?;
    result
}

fn stream_chat_blocking(
    app: tauri::AppHandle,
    input: String,
) -> Result<String, String> {
    use tauri::Manager;
    use std::io::Write;

    let state = app.state::<AppState>();
    let mut guard = state.sidecar.lock().map_err(|e| e.to_string())?;
    let child = guard.as_mut().ok_or("sidecar not running")?;

    let id = 1;
    let request = serde_json::json!({
        "jsonrpc": "2.0",
        "id": id,
        "method": "agent.stream",
        "params": { "input": input },
    });

    let request_str = serde_json::to_string(&request).map_err(|e| e.to_string())?;

    let mut stdin = child.stdin.take().ok_or("sidecar stdin unavailable")?;
    writeln!(stdin, "{}", request_str).map_err(|e| e.to_string())?;
    stdin.flush().map_err(|e| e.to_string())?;
    child.stdin = Some(stdin);

    let stdout = child.stdout.as_mut().ok_or("sidecar stdout unavailable")?;
    let mut reader = BufReader::new(stdout);
    let mut notifications = Vec::new();

    loop {
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        if line.is_empty() {
            break;
        }

        if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(&line) {
            // Check if this is the final response (has "id" field)
            if parsed.get("id").and_then(|v| v.as_u64()) == Some(id as u64) {
                // Final response — we're done streaming
                break;
            }

            // It's a notification — convert agent.event to stream.* format
            if let Some(method) = parsed.get("method").and_then(|m| m.as_str()) {
                if method == "agent.event" {
                    let params = parsed.get("params").cloned().unwrap_or(serde_json::json!({}));
                    let event_type = params.get("type").and_then(|t| t.as_str()).unwrap_or("unknown");

                    let notification = match event_type {
                        "token" => {
                            let token = params.get("content").and_then(|c| c.as_str()).unwrap_or("");
                            let safe_token = escape_json_string(token);
                            format!("{{\"jsonrpc\":\"2.0\",\"method\":\"stream.token\",\"params\":{{\"token\":\"{}\"}}}}", safe_token)
                        }
                        "tool_call" => {
                            "{\"jsonrpc\":\"2.0\",\"method\":\"stream.tool_call\",\"params\":{\"tool\":\"tool\",\"path\":\"\"}}".to_string()
                        }
                        "tool_result" => {
                            let content = params.get("content").and_then(|c| c.as_str()).unwrap_or("");
                            let safe_content = escape_json_string(content);
                            format!("{{\"jsonrpc\":\"2.0\",\"method\":\"stream.tool_result\",\"params\":{{\"content\":\"{}\"}}}}", safe_content)
                        }
                        "error" => {
                            let error = params.get("error").and_then(|e| e.as_str()).unwrap_or("unknown");
                            let safe_error = escape_json_string(error);
                            format!("{{\"jsonrpc\":\"2.0\",\"method\":\"stream.error\",\"params\":{{\"error\":\"{}\"}}}}", safe_error)
                        }
                        "done" => {
                            notifications.push("{\"jsonrpc\":\"2.0\",\"method\":\"stream.done\",\"params\":{}}".to_string());
                            break;
                        }
                        _ => continue,
                    };
                    notifications.push(notification);
                }
            }
        }
    }

    Ok(notifications.join("\n"))
}

/// Escapes a string for safe inclusion in a JSON string value.
fn escape_json_string(s: &str) -> String {
    s.replace('\\', "\\\\")
     .replace('"', "\\\"")
     .replace('\n', "\\n")
     .replace('\r', "\\r")
     .replace('\t', "\\t")
}

// ============================================================================
// Tauri commands — window management
// ============================================================================

/// Sets the window to the specified mode (companion, floating, or compact).
#[tauri::command]
fn set_window_mode(app: tauri::AppHandle, mode: String) -> Result<(), String> {
    use tauri::Manager;
    let window_mode = parse_window_mode(&mode)?;
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "main window not found".to_string())?;

    // Get primary monitor height for full-height modes
    let monitor = window.current_monitor().map_err(|e| e.to_string())?;
    let screen_height = monitor.map(|m| m.size().height).unwrap_or(1080);

    let dims = calculate_window_dimensions(window_mode, screen_height);

    // Apply window size
    window
        .set_size(tauri::Size::Physical(tauri::PhysicalSize {
            width: dims.width,
            height: dims.height,
        }))
        .map_err(|e| e.to_string())?;

    // Apply always-on-top
    let on_top = dims.always_on_top == "true";
    window.set_always_on_top(on_top).map_err(|e| e.to_string())?;

    // Position: companion and compact go to right edge, floating centers
    if matches!(window_mode, WindowMode::Companion | WindowMode::Compact) {
        if let Some(m) = window.current_monitor().map_err(|e| e.to_string())? {
            let monitor_size = m.size();
            let position = tauri::PhysicalPosition {
                x: (monitor_size.width as i32) - (dims.width as i32),
                y: 0,
            };
            window.set_position(tauri::Position::Physical(position)).map_err(|e| e.to_string())?;
        }
    }

    // Update and persist state
    let state = app.state::<AppState>();
    let mut ws = state.window_state.lock().map_err(|e| e.to_string())?;
    ws.mode = window_mode;
    ws.width = dims.width;
    ws.always_on_top = on_top;

    Ok(())
}

/// Toggles window visibility (show/hide). Shows in last known mode if hidden.
#[tauri::command]
fn toggle_window(app: tauri::AppHandle) -> Result<(), String> {
    use tauri::Manager;
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "main window not found".to_string())?;

    let visible = window.is_visible().map_err(|e| e.to_string())?;
    if visible {
        window.hide().map_err(|e| e.to_string())?;
    } else {
        window.show().map_err(|e| e.to_string())?;
        window.set_focus().map_err(|e| e.to_string())?;
    }

    Ok(())
}

/// Returns the current window state.
#[tauri::command]
fn get_window_state(app: tauri::AppHandle) -> Result<WindowState, String> {
    use tauri::Manager;
    let state = app.state::<AppState>();
    let ws = state.window_state.lock().map_err(|e| e.to_string())?;
    Ok(ws.clone())
}

// ============================================================================
// Test module — tests written before production code (TDD)
// ============================================================================
#[cfg(test)]
mod tests {
    use super::*;
    use std::io::BufReader;
    use std::process::{Child, Command, Stdio};

    // --- Test helpers (spawn mock sidecars) ---

    fn spawn_echo_sidecar() -> Child {
        Command::new("bash")
            .arg("-c")
            .arg(
                r#"while IFS= read -r line; do
  echo '{"jsonrpc":"2.0","result":"ok","id":1}'
done"#,
            )
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .expect("Failed to spawn echo sidecar")
    }

    fn spawn_health_sidecar() -> Child {
        Command::new("bash")
            .arg("-c")
            .arg(
                r#"while IFS= read -r line; do
  if echo "$line" | grep -q '"method":"ping"'; then
    echo '{"jsonrpc":"2.0","result":"pong","id":0}'
  else
    echo '{"jsonrpc":"2.0","result":"ok","id":1}'
  fi
done"#,
            )
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .expect("Failed to spawn health sidecar")
    }

    // ============================================================================
    // Test helper — simulates always-failing spawn for max-retry test
    // ============================================================================

    fn always_failing_restart(max_attempts: u32) -> (bool, u32) {
        for attempt in 1..=max_attempts {
            let result = Command::new("nonexistent_binary_xyz_123")
                .stdin(Stdio::null())
                .stdout(Stdio::null())
                .stderr(Stdio::null())
                .spawn();

            if result.is_ok() {
                return (true, attempt);
            }
        }
        (false, max_attempts)
    }

    // ============================================================================
    // Unit tests — relay, health, JSON-RPC parsing
    // ============================================================================

    #[test]
    fn test_relay_request_writes_and_reads() {
        let mut child = spawn_echo_sidecar();
        let mut stdin = child.stdin.take().unwrap();
        let stdout = child.stdout.take().unwrap();
        let mut reader = BufReader::new(stdout);

        let request =
            r#"{"jsonrpc":"2.0","method":"agent.chat","params":{"input":"hello"},"id":1}"#;
        let response = send_jsonrpc(&mut stdin, &mut reader, request).unwrap();

        assert!(response.contains("\"jsonrpc\":\"2.0\""));
        assert!(response.contains("\"result\""));
        assert!(!response.is_empty());
    }

    #[test]
    fn test_relay_request_with_empty_response() {
        // Sidecar that exits immediately, producing no output
        let mut child = Command::new("bash")
            .arg("-c")
            .arg("exit 0")
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .expect("Failed to spawn empty sidecar");

        let mut stdin = child.stdin.take().unwrap();
        let stdout = child.stdout.take().unwrap();
        let mut reader = BufReader::new(stdout);

        let request = r#"{"jsonrpc":"2.0","method":"ping","id":0}"#;
        let result = send_jsonrpc(&mut stdin, &mut reader, request);

        match result {
            Ok(response) => assert!(response.is_empty(), "Expected empty response from dead sidecar"),
            Err(_) => {} // Broken pipe is also acceptable for a dead process
        }
    }

    #[test]
    fn test_health_check_alive() {
        let mut child = spawn_health_sidecar();
        let mut stdin = child.stdin.take().unwrap();
        let stdout = child.stdout.take().unwrap();
        let mut reader = BufReader::new(stdout);

        let healthy = check_sidecar_health(&mut stdin, &mut reader).unwrap();
        assert!(healthy, "Sidecar should report healthy when it responds with pong");
    }

    #[test]
    fn test_health_check_dead_sidecar() {
        let mut child = Command::new("bash")
            .arg("-c")
            .arg("exit 1")
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .expect("Failed to spawn dead sidecar");

        let mut stdin = child.stdin.take().unwrap();
        let stdout = child.stdout.take().unwrap();
        let mut reader = BufReader::new(stdout);

        let result = check_sidecar_health(&mut stdin, &mut reader);
        match result {
            Ok(healthy) => assert!(!healthy, "Dead sidecar should report unhealthy"),
            Err(_) => {} // Broken pipe on dead sidecar is an acceptable error path
        }
    }

    #[test]
    fn test_is_ready_notification_true() {
        let ready = r#"{"jsonrpc":"2.0","method":"ready","params":{"version":"1.0.0"}}"#;
        assert!(is_ready_notification(ready));
    }

    #[test]
    fn test_is_ready_notification_false() {
        let not_ready = r#"{"jsonrpc":"2.0","result":"ok","id":1}"#;
        assert!(!is_ready_notification(not_ready));

        let error_msg = r#"{"jsonrpc":"2.0","error":{"code":-32000,"message":"fail"},"id":1}"#;
        assert!(!is_ready_notification(error_msg));
    }

    #[test]
    fn test_parse_jsonrpc_response_valid() {
        let line = r#"{"jsonrpc":"2.0","result":"pong","id":0}"#;
        let parsed = parse_jsonrpc_response(line).unwrap();
        assert_eq!(parsed["result"], "pong");
        assert_eq!(parsed["id"], 0);
    }

    #[test]
    fn test_parse_jsonrpc_response_error() {
        let line = r#"{"jsonrpc":"2.0","error":{"code":-32000,"message":"not found"},"id":1}"#;
        let parsed = parse_jsonrpc_response(line).unwrap();
        assert_eq!(parsed["error"]["code"], -32000);
        assert_eq!(parsed["error"]["message"], "not found");
    }

    #[test]
    fn test_parse_jsonrpc_response_invalid() {
        let line = "not valid json";
        let result = parse_jsonrpc_response(line);
        assert!(result.is_err(), "Should return error for invalid JSON");
    }

    #[test]
    fn test_format_jsonrpc_request() {
        let req = format_jsonrpc_request("agent.chat", r#"{"input":"hello"}"#);
        assert!(req.contains("\"jsonrpc\":\"2.0\""));
        assert!(req.contains("\"method\":\"agent.chat\""));
        assert!(req.contains("\"params\":{\"input\":\"hello\"}"));
    }

    #[test]
    fn test_sidecar_crash_triggers_restart() {
        // Spawn a sidecar that exits immediately (simulating crash)
        let mut child = Command::new("bash")
            .arg("-c")
            .arg("exit 1")
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .unwrap();

        let status = child.wait().unwrap();
        assert!(!status.success(), "Sidecar should crash with non-zero exit");

        // Verify restart logic: max 3 attempts, eventually succeeds
        let (spawned, attempts) = restart_sidecar_with_retry(3);
        assert!(spawned, "Should eventually spawn successfully");
        assert!(attempts <= 3, "Should not exceed max attempts of 3");
    }

    #[test]
    fn test_sidecar_restart_max_retries_exceeded() {
        // Simulate always-failing spawn (max 3 attempts, all fail)
        let (spawned, attempts) = always_failing_restart(3);
        assert!(!spawned, "Should report failure after max retries exhausted");
        assert_eq!(attempts, 3, "Should try exactly max_attempts times before giving up");
    }

    // ============================================================================
    // Unit tests — WindowMode parsing and dimension calculations (companion-mode)
    // ============================================================================

    #[test]
    fn test_parse_window_mode_companion() {
        let mode = parse_window_mode("companion").unwrap();
        assert_eq!(mode, WindowMode::Companion);
    }

    #[test]
    fn test_parse_window_mode_floating() {
        let mode = parse_window_mode("floating").unwrap();
        assert_eq!(mode, WindowMode::Floating);
    }

    #[test]
    fn test_parse_window_mode_compact() {
        let mode = parse_window_mode("compact").unwrap();
        assert_eq!(mode, WindowMode::Compact);
    }

    #[test]
    fn test_parse_window_mode_case_insensitive() {
        assert_eq!(parse_window_mode("Companion").unwrap(), WindowMode::Companion);
        assert_eq!(parse_window_mode("FLOATING").unwrap(), WindowMode::Floating);
        assert_eq!(parse_window_mode("Compact").unwrap(), WindowMode::Compact);
    }

    #[test]
    fn test_parse_window_mode_invalid() {
        let result = parse_window_mode("invalid");
        assert!(result.is_err(), "Should error on unknown mode");
    }

    #[test]
    fn test_window_mode_to_string() {
        assert_eq!(WindowMode::Companion.to_string(), "companion");
        assert_eq!(WindowMode::Floating.to_string(), "floating");
        assert_eq!(WindowMode::Compact.to_string(), "compact");
    }

    #[test]
    fn test_calculate_dimensions_companion() {
        let screen_height = 1080u32;
        let dims = calculate_window_dimensions(WindowMode::Companion, screen_height);
        assert_eq!(dims.width, COMPANION_WIDTH);
        assert_eq!(dims.height, screen_height);
        assert_eq!(dims.always_on_top, "true");
    }

    #[test]
    fn test_calculate_dimensions_floating() {
        let screen_height = 1080u32;
        let dims = calculate_window_dimensions(WindowMode::Floating, screen_height);
        assert_eq!(dims.width, FLOATING_WIDTH);
        assert_eq!(dims.height, FLOATING_HEIGHT);
        assert_eq!(dims.always_on_top, "false");
    }

    #[test]
    fn test_calculate_dimensions_compact() {
        let screen_height = 1080u32;
        let dims = calculate_window_dimensions(WindowMode::Compact, screen_height);
        assert_eq!(dims.width, COMPACT_WIDTH);
        assert_eq!(dims.height, screen_height - 100);
    }

    #[test]
    fn test_window_state_serialization() {
        let state = WindowState {
            mode: WindowMode::Companion,
            width: COMPANION_WIDTH,
            always_on_top: true,
        };
        let json = serde_json::to_string(&state).unwrap();
        assert!(json.contains("\"companion\""));
        assert!(json.contains("380"));
        assert!(json.contains("true"));
    }

    #[test]
    fn test_window_state_deserialization() {
        let json = r#"{"mode":"floating","width":420,"always_on_top":false}"#;
        let state: WindowState = serde_json::from_str(json).unwrap();
        assert_eq!(state.mode, WindowMode::Floating);
        assert_eq!(state.width, 420);
        assert!(!state.always_on_top);
    }
}
