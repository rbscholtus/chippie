#![warn(clippy::all, rust_2018_idioms)]

mod app;
pub use app::TemplateApp;

mod chip8;

mod roms_db;
pub use roms_db::{get_data, HASHES, PROGRAMS, ROMS};
