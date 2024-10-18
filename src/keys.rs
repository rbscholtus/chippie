use egui::Key;

pub struct KeyMapper {
    pub key_map: [Key; 16],
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
    #[allow(dead_code)]
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
    pub fn new(key_map: Option<[Key; 16]>) -> Self {
        Self {
            key_map: key_map.unwrap_or(Self::COSMAC_ELF),
        }
    }
}
