use crate::{chip8, keys, roms_db};
use egui::{
    menu, Color32, ColorImage, Context, ImageData, Pos2, RichText, TextureOptions, Vec2, Vec2b,
};
use once_cell::sync::Lazy;
use sha1::{Digest, Sha1};
use std::sync::Arc;
use web_time::{Duration, Instant};

// Constants
const EMU_ASPECT_RATIO: f32 = 64_f32 / 32_f32;
static FRAME_DURATION: Lazy<Duration> = Lazy::new(|| Duration::from_secs_f64(1_f64 / 60_f64));

fn calculate_sha1(data: &[u8]) -> String {
    let mut hasher = Sha1::new();
    hasher.update(data);
    hex::encode(hasher.finalize())
}

pub struct TemplateApp<'a> {
    paused: bool,
    ticks_per_frame: u16,
    updates: u32,
    begin_updates_time: Instant,
    frames: u32,
    begin_time: Instant,
    next_update: Instant,
    color_on: Color32,
    color_off: Color32,
    image_texture: Option<egui::TextureHandle>,
    chip8: chip8::Cpu,
    keys: keys::KeyMapper,
    hash: Option<String>,
    program_info: Option<&'a roms_db::Program>,
    rom_info: Option<&'a roms_db::Rom>,
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
            color_on: Color32::WHITE,
            color_off: Color32::BLACK,
            image_texture: None,
            chip8: chip8::Cpu::new(),
            keys: keys::KeyMapper::new(None),
            hash: None,
            program_info: None,
            rom_info: None,
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
    fn update(&mut self, ctx: &Context, _frame: &mut eframe::Frame) {
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
            if let Some(program) = self.program_info {
                self.show_rom_popup(ctx, program);
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
    fn proc_input(&mut self, _ctx: &Context, x: &egui::InputState) {
        // SPACE runs/pauses the emu
        if x.key_released(egui::Key::Space) {
            self.paused = !self.paused;
        }
        // register keys down
        for i in 0..self.chip8.keys_down.len() {
            self.chip8.keys_down[i] = x.key_down(self.keys.key_map[i]);
        }
    }

    fn update_emu_state(&mut self) {
        // doing an update(s)
        let now = Instant::now();
        while self.next_update < now {
            if !self.paused {
                self.chip8.ticks(self.ticks_per_frame);
                self.updates += 1;
            } else {
                self.begin_updates_time += *FRAME_DURATION;
            }

            self.next_update += *FRAME_DURATION;
        }
    }

    fn load_roms_menu(
        &mut self,
        ui: &mut egui::Ui,
        roms: &std::collections::HashMap<&str, Vec<u8>>,
    ) {
        // Collect the filenames into a vector and sort them
        let mut filenames: Vec<_> = roms.keys().collect();
        filenames.sort();

        // Display each item in the menu
        for &filename in filenames {
            if ui.button(filename).clicked() {
                self.hash = None;
                self.program_info = None;
                self.rom_info = None;

                // load ROM data
                let bindata = roms.get(filename).unwrap();
                let hash = calculate_sha1(bindata);

                // get program and rom info, and set tickrate
                if let Some(id) = roms_db::HASHES.get(&hash) {
                    self.program_info = roms_db::PROGRAMS.get(*id as usize);
                    self.rom_info = self
                        .program_info
                        .and_then(|pr_info| pr_info.roms.get(&hash));
                    if let Some(ticks) = self.rom_info.and_then(|rinfo| rinfo.get_tickrate()) {
                        self.ticks_per_frame = ticks;
                    }
                    self.hash = Some(hash);

                    // show popup next frame
                    self.show_popup = true;
                    self.start_clicked = false;
                    self.paused = true;
                } else {
                    self.paused = false;
                }

                // create new emu with the same colors
                self.chip8 = chip8::cpu::Cpu::new();
                self.chip8.bus.load_rom(bindata);
                ui.close_menu();
            }
        }
    }

    fn show_menu(&mut self, _ctx: &Context, ui: &mut egui::Ui) {
        menu::bar(ui, |ui| {
            ui.menu_button("Programs", |ui| {
                ui.menu_button("Timendus tests", |ui| {
                    self.load_roms_menu(ui, &roms_db::ROMS)
                });
                ui.menu_button("Games", |ui| self.load_roms_menu(ui, &roms_db::ROMS2));
            });

            ui.menu_button("Color", |ui| {
                /* if ui.butto n("From ROM (if any)").clicked() {
                    if let Some(info) = self.program_info {
                        if let Some(ticks) = info.roms[&hash].colors() {
                            self.ticks_per_frame = ticks;
                        }
                    }
                    if let Some(colors) = self.program_info. {}
                    // todo!()
                } */
                if ui.button("B/W").clicked() {
                    self.color_on = Color32::WHITE;
                    self.color_off = Color32::BLACK;
                    ui.close_menu();
                }
                if ui.button("Orange").clicked() {
                    self.color_on = Color32::from_rgb(0xFF, 0xAA, 0);
                    self.color_off = Color32::BLACK;
                    ui.close_menu();
                }
                if ui.button("Timendus").clicked() {
                    self.color_on = Color32::from_rgb(0xFF, 0xCC, 0x01);
                    self.color_off = Color32::from_rgb(0x99, 0x66, 0x01);
                    ui.close_menu();
                }
            });

            // Create a pause/run toggle button
            ui.separator();

            let (t, fg, bg) = if self.paused {
                ("Paused", Color32::BLACK, Color32::DARK_GRAY)
            } else {
                ("Running", Color32::BLACK, Color32::LIGHT_GREEN)
            };
            if ui
                .button(RichText::new(t).color(fg).background_color(bg))
                .clicked()
            {
                self.paused = !self.paused;
            }

            // Show sound or not
            ui.separator();

            if self.chip8.sound_timer > 0 {
                ui.label(
                    RichText::new("BEEP")
                        .strong()
                        .color(Color32::BLACK)
                        .background_color(Color32::LIGHT_RED),
                );
            } else {
                ui.label("BEEP");
            }

            // Show emu speed slider
            ui.separator();
            ui.label("Tickrate (speed):");
            ui.add(egui::Slider::new(&mut self.ticks_per_frame, 1..=256).text("ticks/frame"));
        });
    }

    fn show_rom_popup(&mut self, ctx: &Context, program: &roms_db::Program) {
        let avl_rect = ctx.screen_rect();
        let pos_rect = Pos2::new(avl_rect.width() * 0.15, avl_rect.height() * 0.1);
        let size_vec = Vec2::new(avl_rect.width() * 0.7, avl_rect.height() * 0.8);

        egui::Window::new(program.get_title())
            .fixed_pos(pos_rect)
            .fixed_size(size_vec)
            .resizable(false)
            .collapsible(false)
            .title_bar(false)
            .scroll(Vec2b::new(true, true))
            .open(&mut self.show_popup)
            .show(ctx, |ui| {
                ui.with_layout(egui::Layout::top_down(egui::Align::Center), |ui| {
                    // header
                    ui.label(egui::RichText::new(program.get_title()).heading().strong());

                    // basic program metadata
                    egui::Grid::new("my_grid").num_columns(2).show(ui, |ui| {
                        ui.label("Description:");
                        ui.add(egui::Label::new(program.get_description()).wrap());
                        ui.end_row();
                        ui.label("Released:");
                        ui.label(program.get_release());
                        ui.end_row();
                        ui.label("Author(s):");
                        ui.label(program.get_authors());
                        ui.end_row();
                        if let Some(copy) = program.get_copyright() {
                            ui.label("Copyright:");
                            ui.label(copy);
                            ui.end_row();
                        }
                        if let Some(origin) = program.get_origin() {
                            ui.label("Origin:");
                            ui.label(origin);
                            ui.end_row();
                        }
                        if let Some(urls) = program.get_urls() {
                            for (i, url) in urls.iter().enumerate() {
                                ui.label(if i == 0 { "URL:" } else { "" });
                                ui.hyperlink(url);
                                ui.end_row();
                            }
                        }
                    });

                    // Display ROM files
                    for (romhash, romfile) in &program.roms {
                        ui.add_space(10.0);
                        let text = if romhash == self.hash.as_deref().unwrap() {
                            egui::RichText::new(format!("Currently loaded ROM ({})", romhash))
                                .strong()
                        } else {
                            egui::RichText::new(format!("Other ROM ({})", romhash))
                        };
                        ui.add(egui::Label::new(text));

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

    fn show_stats_bar(&self, _ctx: &Context, ui: &mut egui::Ui) {
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

    fn show_emu(&mut self, ctx: &Context, ui: &mut egui::Ui) {
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

    fn show_emu_image(&mut self, ctx: &Context, ui: &mut egui::Ui, image_size: Vec2) {
        // Load new or update the existing framebuffer texture
        let image_texture = self.image_texture.get_or_insert_with(|| {
            self.chip8.bus.gpu.has_changed = false;
            ctx.load_texture(
                "gpu",
                gpu_to_image_data(&self.chip8.bus.gpu.buffer, self.color_on, self.color_off),
                TextureOptions::NEAREST,
            )
        });

        // Update the framebuffer texture
        if self.chip8.bus.gpu.has_changed {
            self.chip8.bus.gpu.has_changed = false;
            image_texture.set(
                gpu_to_image_data(&self.chip8.bus.gpu.buffer, self.color_on, self.color_off),
                TextureOptions::NEAREST,
            );
        }

        // Draw the texture in the UI
        ui.image((image_texture.id(), image_size));

        self.frames += 1;
    }
}

fn gpu_to_image_data(buffer: &[u64; 32], color_on: Color32, color_off: Color32) -> ImageData {
    let mut pixel_data: Vec<Color32> = Vec::with_capacity(64 * 32);
    for buff_line in buffer.iter() {
        let mut mask = 1 << 63;
        for _ in 0..64 {
            pixel_data.push(if buff_line & mask > 0 {
                color_on
            } else {
                color_off
            });
            mask >>= 1;
        }
    }

    let color_image = ColorImage {
        size: [64, 32],
        pixels: pixel_data,
    };

    ImageData::Color(Arc::new(color_image))
}
