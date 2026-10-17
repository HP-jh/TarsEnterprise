// TarsEnterprise Rust Adapter — Environment Detection & Tool Listing
use std::process::Command;
use std::collections::HashMap;

#[derive(Debug)]
pub struct EnvInfo {
    pub lang: String,
    pub version: String,
    pub tools: Vec<String>,
}

pub fn detect() -> EnvInfo {
    let version = Command::new("rustc")
        .arg("--version")
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .unwrap_or_else(|_| "unknown".to_string());

    let mut tools = vec![];
    for tool in &["cargo", "rustfmt", "clippy"] {
        if Command::new(tool).arg("--version").output().is_ok() {
            tools.push(tool.to_string());
        }
    }

    EnvInfo {
        lang: "rust".to_string(),
        version,
        tools,
    }
}

fn main() {
    let info = detect();
    println!("{}", serde_json::to_string_pretty(&info).unwrap_or_default());
}
