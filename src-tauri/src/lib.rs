// Tauri后端命令定义和Go引擎进程管理
use std::sync::Mutex;
use tauri::Emitter;
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;
use tauri::Manager;
use tauri_plugin_store::StoreExt;
use tauri::menu::{Menu, MenuItem};
use tauri::tray::{TrayIconBuilder, MouseButton, MouseButtonState, TrayIconEvent};

// 服务器状态，存储Go引擎进程信息
struct ServerState {
    child: Mutex<Option<CommandChild>>,  // Go引擎子进程
    port: Mutex<u16>,                     // 服务器端口
}

// start_server 启动Go引擎文件接收服务器
#[tauri::command]
async fn start_server(
    state: tauri::State<'_, ServerState>,
    app: tauri::AppHandle,
    port: u16,
    dir: String,
    ip: Option<String>,
) -> Result<String, String> {
    // 停止已运行的服务器
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

    // 等待服务器就绪信号（5秒超时）
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

    // 继续监听剩余事件
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

// stop_server 停止Go引擎服务器
#[tauri::command]
async fn stop_server(state: tauri::State<'_, ServerState>) -> Result<String, String> {
    if let Some(child) = state.child.lock().unwrap().take() {
        child.kill().map_err(|e| e.to_string())?;
        Ok("Server stopped".to_string())
    } else {
        Err("No server running".to_string())
    }
}

// get_server_port 获取当前服务器端口
#[tauri::command]
async fn get_server_port(state: tauri::State<'_, ServerState>) -> Result<u16, String> {
    Ok(*state.port.lock().unwrap())
}

// discover_peers 发现局域网内的其他xShare设备
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

// send_files 发送文件到指定peer
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

    // 监听传输事件并转发给前端
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

// list_ips 列出本机所有可用IP地址
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

// open_file_dialog 打开文件选择对话框
#[tauri::command]
async fn open_file_dialog(app: tauri::AppHandle) -> Result<Option<String>, String> {
    use tauri_plugin_dialog::DialogExt;
    let path = app
        .dialog()
        .file()
        .blocking_pick_file();
    Ok(path.map(|p| p.to_string()))
}

// open_dir_dialog 打开目录选择对话框
#[tauri::command]
async fn open_dir_dialog(app: tauri::AppHandle, dir: Option<String>) -> Result<Option<String>, String> {
    use tauri_plugin_dialog::DialogExt;
    let mut dialog = app.dialog().file();
    if let Some(d) = dir {
        let path = std::path::Path::new(&d);
        let abs_path = if path.is_absolute() {
            d
        } else {
            std::env::current_dir()
                .map(|cwd| cwd.join(path).to_string_lossy().to_string())
                .unwrap_or(d)
        };
        dialog = dialog.set_directory(abs_path);
    }
    let path = dialog.blocking_pick_folder();
    Ok(path.map(|p| p.to_string()))
}

// load_settings 加载应用设置（保存目录及历史记录）
#[tauri::command]
async fn load_settings(app: tauri::AppHandle) -> Result<serde_json::Value, String> {
    let store = app.store("settings.json").map_err(|e| e.to_string())?;
    
    let save_dir = store.get("saveDir")
        .and_then(|v| v.as_str().map(String::from))
        .unwrap_or_else(|| "./received".to_string());
    
    let history: Vec<serde_json::Value> = store.get("saveDirHistory")
        .and_then(|v| serde_json::from_value(v.clone()).ok())
        .unwrap_or_else(Vec::new);
    
    Ok(serde_json::json!({
        "saveDir": save_dir,
        "saveDirHistory": history
    }))
}

// save_settings 保存应用设置（保存目录及历史记录）
#[tauri::command]
async fn save_settings(
    app: tauri::AppHandle,
    save_dir: String,
    history: Vec<serde_json::Value>,
) -> Result<(), String> {
    let store = app.store("settings.json").map_err(|e| e.to_string())?;
    store.set("saveDir", serde_json::Value::String(save_dir));
    store.set("saveDirHistory", serde_json::Value::Array(history));
    store.save().map_err(|e| e.to_string())?;
    Ok(())
}

// run 初始化并运行Tauri应用
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.show();
                let _ = window.set_focus();
            }
        }))
        .manage(ServerState {
            child: Mutex::new(None),
            port: Mutex::new(9527),
        })
        .setup(|app| {
            // 创建托盘菜单项
            let show_i = MenuItem::with_id(app, "show", "显示主界面", true, None::<&str>)?;
            let about_i = MenuItem::with_id(app, "about", "关于", true, None::<&str>)?;
            let quit_i = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&show_i, &about_i, &quit_i])?;

            // 创建系统托盘图标
            let _tray = TrayIconBuilder::with_id("main")
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "show" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    "about" => {
                        // 通过事件通知前端显示关于对话框
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                            let _ = window.emit("show-about", ());
                        }
                    }
                    "quit" => {
                        // 移除托盘图标
                        app.remove_tray_by_id("main");
                        // 关闭所有窗口
                        for (_, window) in app.webview_windows() {
                            let _ = window.close();
                        }
                        // 执行清理并退出
                        app.cleanup_before_exit();
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
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                })
                .build(app)?;

            // 拦截窗口关闭和最小化事件，隐藏到托盘
            if let Some(window) = app.get_webview_window("main") {
                let window_clone = window.clone();
                window.on_window_event(move |event| {
                    if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                        api.prevent_close();
                        let _ = window_clone.hide();
                    }
                });
            }

            Ok(())
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
            load_settings,
            save_settings,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
