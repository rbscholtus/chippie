#![allow(dead_code)]
use crate::chip8;
use crate::roms_db;
use crate::PROGRAMS;
use egui::{menu, Key, Pos2, TextureOptions, Vec2};
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

// fn optvec_to_string(authors: &Option<Vec<String>>) -> String {
//     match authors {
//         Some(vec) => vec.join(", "),
//         None => String::new(), // or you can return "No authors" or similar
//     }
// }

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

pub struct TemplateApp<'a> {
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
    program_info: Option<&'a roms_db::Program>,
    show_popup: bool,
    start_clicked: bool,
}

impl Default for TemplateApp<'_> {
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
            program_info: None,
            show_popup: false,
            start_clicked: false,
        }
    }
}

impl TemplateApp<'_> {
    /// Called once before the first frame.
    pub fn new(_: &eframe::CreationContext<'_>) -> Self {
        Default::default()
    }
}

impl eframe::App for TemplateApp<'_> {
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

        // Show the popup window when `show_popup` is true
        if self.show_popup {
            if let Some(rom) = self.program_info {
                self.show_rom_popup(ctx, rom);
            }
            if !self.show_popup || self.start_clicked {
                self.paused = false;
                self.show_popup = false;
                self.start_clicked = false;
            }
        }
    }
}

impl TemplateApp<'_> {
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
                            self.program_info = (*PROGRAMS).get(id as usize);
                            self.show_popup = true;
                            self.start_clicked = false;
                        }
                        ui.close_menu();
                    }
                }

                // ui.separator();
            });

            ui.menu_button("Color", |ui| {
                if ui.button("From ROM (if any)").clicked() {
                    todo!()
                }
                if ui.button("B/W").clicked() {
                    todo!()
                }
                if ui.button("Timendus").clicked() {
                    todo!()
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

    fn show_rom_popup(&mut self, ctx: &egui::Context, rom: &roms_db::Program) {
        let avl_rect = ctx.screen_rect();
        let pos_rect = Pos2::new(avl_rect.width() * 0.15, avl_rect.height() * 0.1);
        let size_vec = Vec2::new(avl_rect.width() * 0.7, avl_rect.height() * 0.8);

        egui::Window::new(rom.get_title())
            .fixed_pos(pos_rect)
            .fixed_size(size_vec)
            .resizable(false)
            .collapsible(false)
            .title_bar(false)
            .open(&mut self.show_popup)
            .show(ctx, |ui| {
                ui.with_layout(egui::Layout::top_down(egui::Align::Center), |ui| {
                    // header
                    ui.label(egui::RichText::new(rom.get_title()).heading().strong());
                    // ui.heading(rom.get_title());

                    // basic program metadata
                    egui::Grid::new("my_grid").num_columns(2).show(ui, |ui| {
                        ui.label("Title:");
                        ui.label(rom.get_title());
                        ui.end_row();
                        ui.label("Description:");
                        ui.add(egui::Label::new(rom.get_description()).wrap());
                        ui.end_row();
                        ui.label("Released:");
                        ui.label(rom.get_release());
                        ui.end_row();
                        ui.label("Author(s):");
                        ui.label(rom.get_authors());
                        ui.end_row();
                        if let Some(copy) = rom.get_copyright() {
                            ui.label("Copyright:");
                            ui.label(copy);
                            ui.end_row();
                        }
                        if let Some(origin) = rom.get_origin() {
                            ui.label("Origin:");
                            ui.label(origin);
                            ui.end_row();
                        }
                        if let Some(urls) = rom.get_urls() {
                            for i in 0..urls.len() {
                                ui.label(if i == 0 { "URL:" } else { "" });
                                ui.hyperlink(&urls[i]);
                                ui.end_row();
                            }
                        }
                    });

                    // Display ROM files
                    for (romhash, romfile) in &rom.roms {
                        ui.add_space(10.0);
                        ui.add(egui::Label::new(
                            egui::RichText::new(format!("ROM ({})", romhash)).strong(),
                        ));
                        // ui.heading(format!("ROM ({})", romhash));
                        egui::Grid::new(format!("rom{}", romhash))
                            .num_columns(2)
                            .show(ui, |ui| {
                                ui.label("File:");
                                ui.label(romfile.get_file());
                                ui.end_row();
                                ui.label("Platforms:");
                                ui.label(romfile.get_platforms());
                                ui.end_row();
                                if let Some(tickrate) = romfile.get_tickrate() {
                                    ui.label("Tickrate:");
                                    ui.label(format!("{}", tickrate));
                                    ui.end_row();
                                }
                                if let Some(colors) = romfile.get_colors() {
                                    ui.label("Colors:");
                                    ui.label(colors);
                                    ui.end_row();
                                }
                                if let Some(keys) = romfile.get_keys() {
                                    ui.label("Keys:");
                                    ui.label(keys);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_description() {
                                    ui.label("Description:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_embedded_title() {
                                    ui.label("Embedded title:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_release() {
                                    ui.label("Released:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_touch_input_mode() {
                                    ui.label("Touch input mode:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_font_style() {
                                    ui.label("Font style:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                                if let Some(desc) = romfile.get_screen_rotation() {
                                    ui.label("Screen rotation:");
                                    ui.label(desc);
                                    ui.end_row();
                                }
                            });
                    }

                    ui.add_space(10.0);
                    if ui.button("Start!").clicked() {
                        self.start_clicked = true;
                    };
                });
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
