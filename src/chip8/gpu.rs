/// the screen is 64 pixels wide x 32 pixels high
#[derive(Debug, Clone, Copy)]
pub struct Gpu {
    pub buffer: [u64; 32],
    pub has_changed: bool,
}

impl Gpu {
    pub fn new() -> Self {
        Gpu {
            buffer: [0; 32],
            has_changed: true,
        }
    }

    pub fn clear(&mut self) {
        for i in 0..32 {
            self.buffer[i] = 0;
        }
    }

    pub fn draw_sprite_line(&mut self, x: u8, y: u8, sprite_data: u8) -> bool {
        if y > 31 {
            return false;
        }

        self.has_changed = true;

        // shift the sprite to the desired x coordinate
        let mask = if x <= 56 {
            (sprite_data as u64) << (56 - x)
        } else {
            (sprite_data as u64) >> (x - 56)
        };
        let cleared_any = self.buffer[y as usize] & mask > 0;
        self.buffer[y as usize] ^= mask;

        // return true if any bit was flipped back to 0
        cleared_any
    }
}
