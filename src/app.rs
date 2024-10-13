use egui::{Color32, ColorImage};
use once_cell::sync::Lazy;
use rand::Rng;
use web_time::{Duration, Instant};

static FRAME_DURATION: Lazy<Duration> = Lazy::new(|| Duration::from_secs_f64(1_f64 / 60_f64));

/// We derive Deserialize/Serialize so we can persist app state on shutdown.
#[derive(serde::Deserialize, serde::Serialize)]
#[serde(default)] // if we add new fields, give them default values when deserializing old state
pub struct TemplateApp {
    #[serde(skip)]
    updates: u32,
    #[serde(skip)]
    frames: u32,
    #[serde(skip)]
    begin_time: Instant,
    #[serde(skip)]
    next_update: Instant,
    #[serde(skip)]
    image_size: [usize; 2],
    #[serde(skip)]
    color_image: ColorImage,
    #[serde(skip)]
    image_texture: Option<egui::TextureHandle>,
}

impl Default for TemplateApp {
    fn default() -> Self {
        Self {
            updates: 0,
            frames: 0,
            begin_time: Instant::now(),
            next_update: Instant::now() + *FRAME_DURATION,
            image_size: [640, 480],
            color_image: ColorImage::new([640, 480], Color32::from_black_alpha(255)),
            image_texture: None,
        }
    }
}

impl TemplateApp {
    /// Called once before the first frame.
    pub fn new(cc: &eframe::CreationContext<'_>) -> Self {
        // Load previous app state (if any).
        // Note that you must enable the `persistence` feature for this to work.
        if let Some(storage) = cc.storage {
            return eframe::get_value(storage, eframe::APP_KEY).unwrap_or_default();
        }

        Default::default()
    }
}

impl eframe::App for TemplateApp {
    /// Called by the frame work to save state before shutdown.
    fn save(&mut self, storage: &mut dyn eframe::Storage) {
        eframe::set_value(storage, eframe::APP_KEY, self);
    }

    /// Called each time the UI needs repainting, which may be many times per second.
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        egui::CentralPanel::default()
            .frame(egui::Frame::none().inner_margin(0.0))
            .show(ctx, |ui| {
                // doing an update(s)
                while self.next_update < Instant::now() {
                    self.color_image = create_checkerboard_image(self.image_size);
                    self.image_texture = Some(ctx.load_texture(
                        "checkerboard",
                        self.color_image.clone(),
                        Default::default(),
                    ));

                    self.next_update += *FRAME_DURATION;
                    self.updates += 1;
                }

                // draw the framebuffer
                if let Some(text) = &self.image_texture {
                    ui.image((text.id(), text.size_vec2()));
                }
                self.frames += 1;

                ctx.request_repaint_after(self.next_update - Instant::now());
            });

        egui::TopBottomPanel::bottom("bottom").show(ctx, |ui| {
            ui.horizontal(|ui| {
                ui.label(format!(
                    "Updates: {} ({:.1} ups)",
                    self.updates,
                    self.updates as f32 / self.begin_time.elapsed().as_secs_f32()
                ));
                ui.label(format!(
                    "Frames: {} ({:.1} fps)",
                    self.frames,
                    self.frames as f32 / self.begin_time.elapsed().as_secs_f32()
                ));
            });
        });
    }
}

fn create_random_color() -> [u8; 3] {
    let mut rng = rand::thread_rng();
    [
        rng.gen_range(0..256) as u8, // Random red value
        rng.gen_range(0..256) as u8, // Random green value
        rng.gen_range(0..256) as u8, // Random blue value
    ]
}

fn create_checkerboard_image(size: [usize; 2]) -> ColorImage {
    let (width, height) = (size[0], size[1]);
    let mut rgb = Vec::with_capacity(width * height * 3); // 3 bytes per pixel (RGB)

    // Generate two random colors
    let color1 = create_random_color();
    let color2 = create_random_color();

    for y in 0..height {
        for x in 0..width {
            // Alternate colors based on the position
            if (x / 5 + y / 5) % 2 == 0 {
                rgb.push(color1[0]); // R
                rgb.push(color1[1]); // G
                rgb.push(color1[2]); // B
            } else {
                rgb.push(color2[0]); // R
                rgb.push(color2[1]); // G
                rgb.push(color2[2]); // B
            }
        }
    }

    ColorImage::from_rgb(size, &rgb)
}
