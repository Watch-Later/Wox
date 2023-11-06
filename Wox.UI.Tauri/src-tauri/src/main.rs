// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[macro_use]
extern crate log;
extern crate simplelog;

use simplelog::*;
use std::env;
use std::fs::File;
use std::path::PathBuf;
use std::thread::spawn;
use sysinfo::{Pid, System, SystemExt};
use tauri::Manager;

mod websocket;

#[tauri::command]
fn get_server_port() -> String {
    let args: Vec<String> = env::args().collect();
    // use default port 34987 if args[1] is not provided
    let port = if args.len() > 1 {
        args[1].parse::<u16>().unwrap_or(34987)
    } else {
        34987
    };
    port.to_string()
}

#[tauri::command]
fn log_ui(msg: String) {
    info!("UI: {}", msg)
}

fn init_log_file() {
    let config = ConfigBuilder::new()
        .set_time_offset_to_local().unwrap()
        .set_time_format_custom(format_description!("[year]-[month]-[day] [hour]:[minute]:[second].[subsecond digits:3]"))
        .build();

    if let Some(home_dir) = dirs::home_dir() {
        let mut base_path = PathBuf::new();
        base_path.push(home_dir);
        base_path.push(".wox");
        base_path.push("log");
        base_path.push("ui.log");
        CombinedLogger::init(
            vec![
                WriteLogger::new(LevelFilter::Info, config, File::create(base_path).unwrap()),
            ]
        ).unwrap();
    } else {
        println!("Can not find user main home path");
    }
}

fn check_process(pid: i32) -> bool {
    let mut system = System::new_all();
    system.refresh_processes();
    system.process(Pid::from(pid as usize)).is_some()
}

fn check_wox_alive() {
    let args: Vec<String> = env::args().collect();
    if args.len() > 2 {
        let wox_pid = args[2].parse::<i32>().unwrap();

        loop {
            if !check_process(wox_pid) {
                info!("wox process is not alive, exit ui process");
                std::process::exit(0);
            } else {
                info!("wox process is alive");
                std::thread::sleep(std::time::Duration::from_secs(3));
            }
        }
    }
}

fn main() {
    init_log_file();
    spawn(move || {
        check_wox_alive();
    });

    #[cfg(target_os = "macos")]
    {
        use tauri_nspanel::cocoa::appkit::{NSMainMenuWindowLevel, NSWindowCollectionBehavior};
        use tauri_nspanel::WindowExt;
        tauri::Builder::default()
            .plugin(tauri_nspanel::init())
            .setup(|app| {
                let window = app.get_window("main").unwrap();
                // hide the dock icon
                app.set_activation_policy(tauri::ActivationPolicy::Accessory);

                let panel = window.to_panel().unwrap();
                // Set panel above the main menu window level
                panel.set_level(NSMainMenuWindowLevel + 1);
                // Ensure that the panel can display over the top of fullscreen apps
                panel.set_collection_behaviour(NSWindowCollectionBehavior::NSWindowCollectionBehaviorTransient
                    | NSWindowCollectionBehavior::NSWindowCollectionBehaviorMoveToActiveSpace
                );

                spawn(move || {
                    websocket::conn(window);
                });

                Ok(())
            })
            .invoke_handler(tauri::generate_handler![get_server_port, log_ui])
            .run(tauri::generate_context!()).expect("error while running tauri application");
    }

    #[cfg(not(target_os = "macos"))]
    {
        tauri::Builder::default()
            .setup(|app| {
                let window = app.get_window("main").unwrap();
                spawn(move || {
                    websocket::conn(window);
                });

                Ok(())
            })
            .invoke_handler(tauri::generate_handler![get_server_port, log_ui])
            .run(tauri::generate_context!()).expect("error while running tauri application");
    }
}