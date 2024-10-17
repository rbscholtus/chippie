#![allow(unused_macros)]
#![allow(unused_imports)]

/// The `bus` module contains the logic and structures for managing the system bus,
/// which coordinates data flow between the CPU, GPU, and other system components.
pub mod bus;

/// The `cpu` module contains the logic for the Central Processing Unit (CPU),
/// responsible for executing instructions.
pub mod cpu;

/// The `gpu` module contains the logic for the Graphics Processing Unit (GPU),
/// which handles rendering and graphical operations.
pub mod gpu;

// Re-exporting common components for easier access.
pub use bus::Bus;
pub use cpu::CPU;
pub use gpu::GPU;

/// Extracts the least significant nibble (lowest 4 bits) from the given opcode.
#[macro_export]
macro_rules! N {
    ($opcode:expr) => {
        ($opcode & 0x000f) as u8
    };
}

/// Extracts the least significant byte (lowest 8 bits) from the given opcode.
#[macro_export]
macro_rules! NN {
    ($opcode:expr) => {
        ($opcode & 0x00ff) as u8
    };
}

/// Extracts the least significant 12 bits from the given opcode.
#[macro_export]
macro_rules! NNN {
    ($opcode:expr) => {
        ($opcode & 0x0fff) as u16
    };
}

/// Extracts the X register index from the given opcode (bits 8-11).
#[macro_export]
macro_rules! X {
    ($opcode:expr) => {
        (($opcode & 0x0f00) >> 8) as usize
    };
}

/// Extracts the Y register index from the given opcode (bits 4-7).
#[macro_export]
macro_rules! Y {
    ($opcode:expr) => {
        (($opcode & 0x00f0) >> 4) as usize
    };
}
