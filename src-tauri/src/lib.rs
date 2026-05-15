use std::sync::Mutex;
use tauri::Emitter;
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;

struct ServerState {
    child: Mutex<Option<CommandChild>>,
    port: Mutex<u16>,
}

#[tauri::command]
async fn start_server(
    state: tauri::State<'_, ServerState>,
    app: tauri::AppHandle,
    port: u16,
    dir: String,
    ip: Option<String>,
) -> Result<String, String> {
    if let Some(child) = state.child.lock().unwrap().take() {
        let _ = child.kill();
    }

    let sidecar = app
        .shell()
        .sidecar("go-engine")
        .map_err(|e| e.to_string())?;

    let mut args = vec![
        "serve".to_string(),
        format!("--port={}", port),
        format!("--dir={}", dir),
    ];
    let ip_arg = ip.unwrap_or_default();
    if !ip_arg.is_empty() {
        args.push(format!("--ip={}", ip_arg));
    }

    let (mut rx, child) = sidecar
        .args(args)
        .spawn()
        .map_err(|e| e.to_string())?;

    // Wait for the server to signal readiness (5s timeout)
    let app_handle = app.clone();
    let deadline = tokio::time::sleep(std::time::Duration::from_secs(5));
    tokio::pin!(deadline);

    let ready = loop {
        tokio::select! {
            Some(event) = rx.recv() => {
                match event {
                    tauri_plugin_shell::process::CommandEvent::Stdout(line) => {
                        let text = String::from_utf8_lossy(&line).to_string();
                        if text.contains(r#""type":"ready""#) {
                            for l in text.lines() {
                                let _ = app_handle.emit("server-event", l.to_string());
                            }
                            break true;
                        }
                        for l in text.lines() {
                            let _ = app_handle.emit("server-event", l.to_string());
                        }
                    }
                    tauri_plugin_shell::process::CommandEvent::Stderr(line) => {
                        let text = String::from_utf8_lossy(&line).to_string();
                        let _ = app_handle.emit("server-error", text);
                    }
                    tauri_plugin_shell::process::CommandEvent::Terminated(status) => {
                        let _ = app_handle.emit("server-terminated", status.code);
                        break false;
                    }
                    tauri_plugin_shell::process::CommandEvent::Error(err) => {
                        let _ = app_handle.emit("server-error", err);
                        break false;
                    }
                    _ => {}
                }
            }
            _ = &mut deadline => {
                break false;
            }
        }
    };

    if !ready {
        let _ = child.kill();
        return Err("Server process failed to start".to_string());
    }

    *state.child.lock().unwrap() = Some(child);
    *state.port.lock().unwrap() = port;

    // Continue listening for remaining events
    tauri::async_runtime::spawn(async move {
        use tauri_plugin_shell::process::CommandEvent;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    for line in text.lines() {
                        let _ = app_handle.emit("server-event", line.to_string());
                    }
                }
                CommandEvent::Stderr(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    let _ = app_handle.emit("server-error", text);
                }
                CommandEvent::Terminated(status) => {
                    let _ = app_handle.emit("server-terminated", status.code);
                }
                CommandEvent::Error(err) => {
                    let _ = app_handle.emit("server-error", err);
                }
                _ => {}
            }
        }
    });

    Ok(format!("Server started on port {}", port))
}

#[tauri::command]
async fn stop_server(state: tauri::State<'_, ServerState>) -> Result<String, String> {
    if let Some(child) = state.child.lock().unwrap().take() {
        child.kill().map_err(|e| e.to_string())?;
        Ok("Server stopped".to_string())
    } else {
        Err("No server running".to_string())
    }
}

#[tauri::command]
async fn get_server_port(state: tauri::State<'_, ServerState>) -> Result<u16, String> {
    Ok(*state.port.lock().unwrap())
}

#[tauri::command]
async fn discover_peers(app: tauri::AppHandle, timeout: u64, ip: Option<String>) -> Result<String, String> {
    let sidecar = app
        .shell()
        .sidecar("go-engine")
        .map_err(|e| e.to_string())?;

    let ip_arg = ip.unwrap_or_default();
    let output = if ip_arg.is_empty() {
        sidecar
            .args(["discover", &format!("--timeout={}", timeout)])
            .output()
            .await
            .map_err(|e| e.to_string())?
    } else {
        sidecar
            .args([
                "discover",
                &format!("--timeout={}", timeout),
                &format!("--ip={}", ip_arg),
            ])
            .output()
            .await
            .map_err(|e| e.to_string())?
    };

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

#[tauri::command]
async fn send_files(app: tauri::AppHandle, peer: String, path: String) -> Result<String, String> {
    let sidecar = app
        .shell()
        .sidecar("go-engine")
        .map_err(|e| e.to_string())?;

    let (mut rx, _child) = sidecar
        .args(["send", &format!("--peer={}", peer), &format!("--file={}", path)])
        .spawn()
        .map_err(|e| e.to_string())?;

    let app_handle = app.clone();
    let (tx, rx_done) = tokio::sync::oneshot::channel::<Result<i32, String>>();
    let mut tx = Some(tx);

    tauri::async_runtime::spawn(async move {
        use tauri_plugin_shell::process::CommandEvent;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    for line in text.lines() {
                        let _ = app_handle.emit("transfer-progress", line.to_string());
                    }
                }
                CommandEvent::Stderr(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    let _ = app_handle.emit("transfer-error", text);
                }
                CommandEvent::Terminated(status) => {
                    let code = status.code.unwrap_or(-1);
                    if let Some(tx) = tx.take() {
                        if code == 0 {
                            let _ = tx.send(Ok(0));
                        } else {
                            let _ = tx.send(Err(format!(
                                "Send process exited with code {}",
                                code
                            )));
                        }
                    }
                    let _ = app_handle.emit("transfer-complete", status.code);
                }
                CommandEvent::Error(err) => {
                    if let Some(tx) = tx.take() {
                        let _ = tx.send(Err(err.clone()));
                    }
                    let _ = app_handle.emit("transfer-error", err);
                }
                _ => {}
            }
        }
    });

    match rx_done.await {
        Ok(Ok(_)) => Ok("Transfer completed".to_string()),
        Ok(Err(e)) => Err(e),
        Err(_) => Err("Send process failed to start".to_string()),
    }
}

#[tauri::command]
async fn list_ips(app: tauri::AppHandle) -> Result<String, String> {
    let sidecar = app
        .shell()
        .sidecar("go-engine")
        .map_err(|e| e.to_string())?;

    let output = sidecar
        .args(["list-ips"])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

#[tauri::command]
async fn open_file_dialog(app: tauri::AppHandle) -> Result<Option<String>, String> {
    use tauri_plugin_dialog::DialogExt;
    let path = app
        .dialog()
        .file()
        .blocking_pick_file();
    Ok(path.map(|p| p.to_string()))
}

#[tauri::command]
async fn open_dir_dialog(app: tauri::AppHandle) -> Result<Option<String>, String> {
    use tauri_plugin_dialog::DialogExt;
    let path = app
        .dialog()
        .file()
        .blocking_pick_folder();
    Ok(path.map(|p| p.to_string()))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(ServerState {
            child: Mutex::new(None),
            port: Mutex::new(9527),
        })
        .invoke_handler(tauri::generate_handler![
            start_server,
            stop_server,
            get_server_port,
            discover_peers,
            send_files,
            open_file_dialog,
            open_dir_dialog,
            list_ips,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
