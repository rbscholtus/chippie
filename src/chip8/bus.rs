use super::gpu::Gpu;
use std::fmt;
use std::fs::File;
use std::io::{self, Read};

const MEMORY_SIZE: usize = 0x1000; // 4Kb

#[rustfmt::skip]
const FONT_BYTES: [u8; 80] = [
    0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
    0x20, 0x60, 0x20, 0x20, 0x70, // 1
    0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
    0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
    0x90, 0x90, 0xF0, 0x10, 0x10, // 4
    0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
    0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
    0xF0, 0x10, 0x20, 0x40, 0x40, // 7
    0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
    0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
    0xF0, 0x90, 0xF0, 0x90, 0x90, // A
    0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
    0xF0, 0x80, 0x80, 0x80, 0xF0, // C
    0xE0, 0x90, 0x90, 0x90, 0xE0, // D
    0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
    0xF0, 0x80, 0xF0, 0x80, 0x80, // F
];

#[allow(dead_code)]
fn read_binary_file(path: &str) -> io::Result<Vec<u8>> {
    let mut file = File::open(path)?;
    let mut buffer = Vec::new();
    file.read_to_end(&mut buffer)?;
    Ok(buffer)
}

pub struct Bus {
    pub memory: [u8; MEMORY_SIZE],
    pub gpu: Gpu,
}

impl Bus {
    pub fn new() -> Self {
        let mut new_bus = Bus {
            memory: [0; MEMORY_SIZE],
            gpu: Gpu::new(),
        };
        new_bus.load_font();

        new_bus
    }

    pub fn read_byte(&self, address: u16) -> u8 {
        self.memory[(address as usize) % MEMORY_SIZE]
    }

    pub fn save_byte(&mut self, address: u16, data: u8) {
        self.memory[(address as usize) % MEMORY_SIZE] = data;
    }

    pub fn load_font(&mut self) {
        // it’s become popular to put it at 050–09F
        // instruction Fx29 relies on this base address
        self.memory[0x50..0xa0].copy_from_slice(&FONT_BYTES);
    }

    pub fn load_rom(&mut self, source: &[u8]) {
        let to_idx = 0x200 + source.len();
        self.memory[0x200..to_idx].copy_from_slice(source);
    }

    pub fn display(&mut self, x_coord: u8, y_coord: u8, address: u16) -> bool {
        let sprite_data = self.read_byte(address);
        self.gpu.draw_sprite_line(x_coord, y_coord, sprite_data)
    }
}
