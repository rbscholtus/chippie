#![allow(dead_code)]

use egui::{Color32, ColorImage, ImageData};
use once_cell::sync::Lazy;
use std::sync::Arc;

/// the screen is 64 pixels wide x 32 pixels high
#[derive(Debug, Clone, Copy)]
pub struct GPU {
    pub buffer: [u64; 32],
    pub color_on: Color32,
    pub color_off: Color32,
    // dirty_lines: [bool; 32],
    // pub dirty: bool,
}

impl GPU {
    pub fn new() -> Self {
        GPU {
            buffer: [0; 32],
            color_on: Color32::WHITE,
            color_off: Color32::BLACK,
            // dirty_lines: [true; 32],
            // dirty: true,
        }
    }

    pub fn clear(&mut self) {
        for i in 0..32 {
            self.buffer[i] = 0;
            // self.dirty_lines[i] = true;
            // self.dirty = true;
        }
    }

    pub fn draw_sprite(&mut self, mut x: u8, mut y: u8, sprite_data: u8) -> bool {
        // wrap x and y around
        x %= 64;
        y %= 32;

        // any data in the sprite will flip a bit, mark the line as dirty
        // if sprite_data > 0 {
        //     self.dirty_lines[y as usize] = true;
        //     self.dirty = true;
        // }

        // shift the sprite to the right x coordinate, wrap around if needed
        let mask = match x <= 56 {
            true => (sprite_data as u64) << (56 - x),
            false => (sprite_data as u64) >> (x - 56) | (sprite_data as u64) << (120 - x),
        };
        let cleared_any = self.buffer[y as usize] & mask > 0;
        self.buffer[y as usize] ^= mask;
        // println!("{:064b} {:?}", self.buffer[y as usize], cleared_any);

        // return true if any bit was flipped back to 0
        cleared_any
    }

    pub fn draw_sprite_line(&mut self, x: u8, y: u8, sprite_data: u8) -> bool {
        if y > 31 {
            return false;
        }
        // if sprite_data > 0 {
        //     self.dirty_lines[y as usize] = true;
        //     self.dirty = true;
        // }

        // shift the sprite to the desired x coordinate
        let mask = if x <= 56 {
            (sprite_data as u64) << (56 - x)
        } else {
            (sprite_data as u64) >> (x - 56)
        };
        let collision = self.buffer[y as usize] & mask > 0;
        self.buffer[y as usize] ^= mask;

        // return true if any bit was flipped back to 0
        collision
    }
}

impl Into<ImageData> for GPU {
    fn into(self) -> ImageData {
        let mut pixel_data: Vec<Color32> = Vec::with_capacity(64 * 32);
        for y in 0..32 {
            let mut mask = 1 << 63;
            for _ in 0..64 {
                pixel_data.push(if self.buffer[y] & mask > 0 {
                    self.color_on
                } else {
                    self.color_off
                });
                mask >>= 1;
            }
        }

        let color_image = ColorImage {
            size: [64, 32],
            pixels: pixel_data,
        };

        // for i in 0..self.dirty_lines.len() {
        //     self.dirty_lines[i] = false;
        // }
        // self.dirty = false;

        ImageData::Color(Arc::new(color_image))
    }
}
