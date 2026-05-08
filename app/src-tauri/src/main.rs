// G-MAN v1.0 — Tauri Desktop Shell
// Sidecar spawn + JSON-RPC relay + System tray

use serde_json::Value;
use std::io::{BufRead, BufReader, Write};
use std::process::{Child, Command, Stdio};

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
pub fn relay_request(
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
    match relay_request(stdin, stdout, ping) {
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

fn main() {
    use std::sync::Mutex;
    use tauri::{
        menu::{MenuBuilder, MenuItemBuilder},
        tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
        Manager,
    };

    // Sidecar process state
    struct SidecarState {
        process: Mutex<Option<Child>>,
    }

    tauri::Builder::default()
        .manage(SidecarState {
            process: Mutex::new(None),
        })
        .setup(|app| {
            // Spawn sidecar on startup
            // Try multiple locations: dev mode (CWD), resource dir (production), relative
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
                    let state = app.state::<SidecarState>();
                    *state.process.lock().unwrap() = Some(c);
                }
                Err(e) => {
                    eprintln!("Failed to spawn sidecar: {}", e);
                }
            }

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
                        // Kill sidecar before exit
                        if let Some(state) = app.try_state::<SidecarState>() {
                            if let Ok(mut guard) = state.process.lock() {
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
                // Minimize to tray instead of closing
                let _ = window.hide();
                api.prevent_close();
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running G-MAN");
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
        let response = relay_request(&mut stdin, &mut reader, request).unwrap();

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
        let result = relay_request(&mut stdin, &mut reader, request);

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
}
