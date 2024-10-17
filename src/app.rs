#![allow(dead_code)]
use crate::chip8;
use crate::roms_db;
use crate::PROGRAMS;
use egui::{menu, Key, TextureOptions, Vec2};
use once_cell::sync::Lazy;
use sha1::{Digest, Sha1};
use web_time::{Duration, Instant};

// Constants
const EMU_ASPECT_RATIO: f32 = 64_f32 / 32_f32;
static FRAME_DURATION: Lazy<Duration> = Lazy::new(|| Duration::from_secs_f64(1_f64 / 60_f64));

fn calculate_sha1(data: &Vec<u8>) -> String {
    // Create a Sha1 hasher instance
    let mut hasher = Sha1::new();

    // Feed the Vec<u8> data into the hasher
    hasher.update(data);

    // Retrieve the resulting hash as bytes and convert to a hexadecimal string
    let result = hasher.finalize();
    hex::encode(result)
}

struct KeyMapper {
    key_map: [Key; 16],
}

impl KeyMapper {
    // Constant key map for COSMAC ELF
    pub const COSMAC_ELF: [Key; 16] = [
        Key::X,
        Key::Num1,
        Key::Num2,
        Key::Num3,
        Key::Q,
        Key::W,
        Key::E,
        Key::A,
        Key::S,
        Key::D,
        Key::Z,
        Key::C,
        Key::Num4,
        Key::R,
        Key::F,
        Key::V,
    ];

    // Constant key map for DREAM 6800
    pub const DREAM_6800: [Key; 16] = [
        Key::Num1,
        Key::Num2,
        Key::Num3,
        Key::Num4,
        Key::Q,
        Key::W,
        Key::E,
        Key::R,
        Key::A,
        Key::S,
        Key::D,
        Key::F,
        Key::Z,
        Key::X,
        Key::C,
        Key::V,
    ];

    // Create a new KeyMapper with a custom or default key map
    fn new(key_map: Option<[Key; 16]>) -> Self {
        Self {
            key_map: key_map.unwrap_or(Self::COSMAC_ELF),
        }
    }
}

pub struct TemplateApp {
    paused: bool,
    ticks_per_frame: u32,
    updates: u32,
    begin_updates_time: Instant,
    frames: u32,
    begin_time: Instant,
    next_update: Instant,
    image_texture: Option<egui::TextureHandle>,
    chip8: chip8::CPU,
    keys: KeyMapper,
}

impl Default for TemplateApp {
    fn default() -> Self {
        Self {
            paused: true,
            ticks_per_frame: 10,
            updates: 0,
            begin_updates_time: Instant::now(),
            frames: 0,
            begin_time: Instant::now(),
            next_update: Instant::now() + *FRAME_DURATION,
            image_texture: None,
            chip8: chip8::CPU::new(),
            keys: KeyMapper::new(None),
        }
    }
}

impl TemplateApp {
    /// Called once before the first frame.
    pub fn new(_: &eframe::CreationContext<'_>) -> Self {
        Default::default()
    }
}

impl eframe::App for TemplateApp {
    /// Called each time the UI needs repainting, which may be many times per second.
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        ctx.input(|x| {
            self.proc_input(ctx, x);
        });

        self.update_emu_state();

        egui::TopBottomPanel::top("menu").show(ctx, |ui| {
            self.show_menu(ctx, ui);
        });

        egui::TopBottomPanel::bottom("stats").show(ctx, |ui| {
            self.show_stats_bar(ctx, ui);
        });

        egui::CentralPanel::default()
            .frame(egui::Frame::none().inner_margin(0.0))
            .show(ctx, |ui| {
                self.show_emu(ctx, ui);
                ctx.request_repaint_after(self.next_update - Instant::now());
            });
    }
}

impl TemplateApp {
    fn proc_input(&mut self, _ctx: &egui::Context, x: &egui::InputState) {
        if x.key_pressed(egui::Key::Space) {
            self.paused = !self.paused;
        }
        for i in 0..self.chip8.inp.len() {
            self.chip8.inp[i] = x.key_down(self.keys.key_map[i]);
        }
    }

    fn update_emu_state(&mut self) {
        // doing an update(s)
        while self.next_update < Instant::now() {
            if !self.paused {
                for _ in 0..self.ticks_per_frame {
                    self.chip8.tick();
                }
                self.updates += 1;
            } else {
                self.begin_updates_time += *FRAME_DURATION;
            }

            self.next_update += *FRAME_DURATION;
        }
    }

    fn show_menu(&mut self, _ctx: &egui::Context, ui: &mut egui::Ui) {
        menu::bar(ui, |ui| {
            ui.menu_button("ROM", |ui| {
                // Collect the filenames into a vector and sort them
                let mut filenames: Vec<&str> = roms_db::ROMS.keys().cloned().collect();
                filenames.sort();

                for filename in filenames {
                    if ui.button(filename).clicked() {
                        // Get the corresponding data from the HashMap
                        if let Some(data) = roms_db::ROMS.get(filename) {
                            self.paused = true;
                            self.chip8 = chip8::cpu::CPU::new();
                            self.chip8.bus.load_rom(data);
                            let hash = calculate_sha1(data);
                            let id = (*roms_db::HASHES)[&hash];
                            let info = (*PROGRAMS).get(id as usize);
                            println!("{:?}", info);
                        }
                        ui.close_menu();
                    }
                }
            });

            // Create a pause/run toggle button
            let pr_text = if self.paused { "Paused" } else { "Running" };
            if ui.button(pr_text).highlight().clicked() {
                self.paused = !self.paused;
            }

            ui.add_space(10_f32);
            ui.separator();

            ui.label("Speed:");
            ui.add(egui::Slider::new(&mut self.ticks_per_frame, 1..=256).text("ticks/sec"));
        });
    }

    fn show_stats_bar(&self, _ctx: &egui::Context, ui: &mut egui::Ui) {
        ui.horizontal(|ui| {
            ui.label(format!(
                "Updates: {} ({:.1} ups)",
                self.updates,
                self.updates as f32 / self.begin_updates_time.elapsed().as_secs_f32()
            ));
            ui.label(format!(
                "Frames: {} ({:.1} fps)",
                self.frames,
                self.frames as f32 / self.begin_time.elapsed().as_secs_f32()
            ));
        });
    }

    fn show_emu(&mut self, ctx: &egui::Context, ui: &mut egui::Ui) {
        // Calculate the available aspect ratio
        let avail_asp_ratio = ui.available_width() / ui.available_height();

        if avail_asp_ratio > EMU_ASPECT_RATIO {
            // Layout horizontally, add spacer to the left/right
            let image_size = Vec2::new(
                EMU_ASPECT_RATIO * ui.available_height(),
                ui.available_height(),
            );
            let spacer = (ui.available_width() - image_size.x) / 2.0;

            ui.horizontal(|ui| {
                ui.add_space(spacer);
                self.show_emu_image(ctx, ui, image_size);
            });
        } else {
            // Layout vertically, add spacer to the top/bottom
            let image_size = Vec2::new(
                ui.available_width(),
                ui.available_width() / EMU_ASPECT_RATIO,
            );
            let spacer = (ui.available_height() - image_size.y) / 2.0;

            ui.vertical(|ui| {
                ui.add_space(spacer);
                self.show_emu_image(ctx, ui, image_size);
            });
        }
    }

    fn show_emu_image(&mut self, ctx: &egui::Context, ui: &mut egui::Ui, image_size: Vec2) {
        // Load new or update the existing framebuffer texture
        let image_texture = self.image_texture.get_or_insert_with(|| {
            ctx.load_texture("gpu", self.chip8.bus.gpu, TextureOptions::NEAREST)
        });

        // Update the framebuffer texture
        image_texture.set(self.chip8.bus.gpu, TextureOptions::NEAREST);

        // Draw the texture in the UI
        ui.image((
            image_texture.id(),
            image_size, /* ui.available_size() */
        ));

        self.frames += 1;
    }
}
