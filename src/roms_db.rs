use once_cell::sync::Lazy;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// Embed the binary data (e.g., a .ch8 file) into the program
pub static ROMS: Lazy<HashMap<&'static str, Vec<u8>>> = Lazy::new(|| {
    let mut programs: HashMap<&'static str, Vec<u8>> = HashMap::new();

    // Insert each file into the HashMap with the filename as the key
    programs.insert(
        "1-chip8-logo.ch8",
        include_bytes!("../roms/tests/1-chip8-logo.ch8").to_vec(),
    );
    programs.insert(
        "2-ibm-logo.ch8",
        include_bytes!("../roms/tests/2-ibm-logo.ch8").to_vec(),
    );
    programs.insert(
        "3-corax+.ch8",
        include_bytes!("../roms/tests/3-corax+.ch8").to_vec(),
    );
    programs.insert(
        "4-flags.ch8",
        include_bytes!("../roms/tests/4-flags.ch8").to_vec(),
    );
    programs.insert(
        "5-quirks.ch8",
        include_bytes!("../roms/tests/5-quirks.ch8").to_vec(),
    );
    programs.insert(
        "6-keypad.ch8",
        include_bytes!("../roms/tests/6-keypad.ch8").to_vec(),
    );
    programs.insert(
        "7-beep.ch8",
        include_bytes!("../roms/tests/7-beep.ch8").to_vec(),
    );

    programs
});

// Embed the binary data (e.g., a .ch8 file) into the program
pub static ROMS2: Lazy<HashMap<&'static str, Vec<u8>>> = Lazy::new(|| {
    let mut programs: HashMap<&'static str, Vec<u8>> = HashMap::new();

    // Insert each file into the HashMap with the filename as the key
    programs.insert(
        "petdog.ch8",
        include_bytes!("../roms/games/petdog.ch8").to_vec(),
    );
    programs.insert(
        "pumpkindressup.ch8",
        include_bytes!("../roms/games/pumpkindressup.ch8").to_vec(),
    );
    programs.insert(
        "octoachip8story.ch8",
        include_bytes!("../roms/games/octoachip8story.ch8").to_vec(),
    );
    programs.insert("RPS.ch8", include_bytes!("../roms/games/RPS.ch8").to_vec());
    programs.insert(
        "outlaw.ch8",
        include_bytes!("../roms/games/outlaw.ch8").to_vec(),
    );
    programs.insert(
        "caveexplorer.ch8",
        include_bytes!("../roms/games/caveexplorer.ch8").to_vec(),
    );
    programs.insert(
        "chipwar.ch8",
        include_bytes!("../roms/games/chipwar.ch8").to_vec(),
    );
    programs.insert(
        "slipperyslope.ch8",
        include_bytes!("../roms/games/slipperyslope.ch8").to_vec(),
    );
    programs.insert(
        "fuse.ch8",
        include_bytes!("../roms/games/fuse.ch8").to_vec(),
    );

    programs
});

pub static HASHES: Lazy<HashMap<String, u32>> =
    Lazy::new(|| load_embedded_sha1_hashes().unwrap_or_default());

pub static PROGRAMS: Lazy<Vec<Program>> =
    Lazy::new(|| load_embedded_programs().unwrap_or_default());

// Function to load the embedded JSON and parse it into a HashMap<String, u32>
fn load_embedded_sha1_hashes() -> Result<HashMap<String, u32>, Box<dyn std::error::Error>> {
    // Use the include_str! macro to embed the JSON file into the binary
    let json_data = include_str!("../data/sha1-hashes.json");

    // Parse the JSON into a HashMap<String, u32>
    let hash_map: HashMap<String, u32> = serde_json::from_str(json_data)?;

    Ok(hash_map)
}

/// Struct for colors used in the ROM
#[derive(Debug, Serialize, Deserialize)]
pub struct Colors {
    pixels: Vec<String>,
    buzzer: Option<String>,
    silence: Option<String>,
}

// Struct for optional keys mapping
#[derive(Debug, Serialize, Deserialize)]
pub struct Keys {
    up: Option<u8>,
    down: Option<u8>,
    left: Option<u8>,
    right: Option<u8>,
    a: Option<u8>,
    b: Option<u8>,
    #[serde(rename = "player2Up")]
    player2_up: Option<u8>,
    #[serde(rename = "player2Down")]
    player2_down: Option<u8>,
}

// // Struct for quirky platform behavior (optional)
// #[derive(Debug, Serialize, Deserialize)]
// pub struct QuirkyPlatform {
//     shift: Option<bool>,
//     #[serde(rename = "memoryLeaveIUnchanged")]
//     memory_leave_i_unchanged: Option<bool>,
// }

// Struct representing details of a ROM
#[derive(Debug, Serialize, Deserialize)]
pub struct Rom {
    file: String,
    platforms: Vec<String>,
    tickrate: Option<u16>,       // Optional tickrate
    colors: Option<Colors>,      // Optional colors
    keys: Option<Keys>,          // Optional keys
    description: Option<String>, // Optional description
    #[serde(rename = "embeddedTitle")]
    embedded_title: Option<String>, // Optional embedded title
    #[serde(rename = "touchInputMode")]
    touch_input_mode: Option<String>, // Optional touch input mode
    #[serde(rename = "fontStyle")]
    font_style: Option<String>, // Optional font style
    // #[serde(rename = "quirkyPlatforms")]
    // quirky_platforms: Option<HashMap<String, QuirkyPlatform>>, // Optional quirky platforms
    release: Option<String>, // Optional release year/date
    #[serde(rename = "screenRotation")]
    screen_rotation: Option<u16>, // Optional screen rotation angle
}

impl Rom {
    pub fn get_file(&self) -> &str {
        &self.file
    }

    pub fn get_platforms(&self) -> String {
        if self.platforms.is_empty() {
            String::new()
        } else {
            self.platforms.join(", ")
        }
    }

    pub fn get_tickrate(&self) -> Option<u16> {
        self.tickrate
    }

    pub fn get_colors(&self) -> Option<String> {
        self.colors.as_ref().map(|_| "Colors exist".to_string())
    }

    pub fn get_keys(&self) -> Option<String> {
        // Replace with actual handling if Keys needs to return a string
        self.keys.as_ref().map(|_| "Keys exist".to_string())
    }

    pub fn get_description(&self) -> Option<&str> {
        self.description.as_deref()
    }

    pub fn get_embedded_title(&self) -> Option<&str> {
        self.embedded_title.as_deref()
    }

    pub fn get_touch_input_mode(&self) -> Option<&str> {
        self.touch_input_mode.as_deref()
    }

    pub fn get_font_style(&self) -> Option<&str> {
        self.font_style.as_deref()
    }

    pub fn get_release(&self) -> Option<&str> {
        self.release.as_deref()
    }

    pub fn get_screen_rotation(&self) -> Option<String> {
        self.screen_rotation.map(|rotation| rotation.to_string())
    }
}

// Struct representing the origin of the program
#[derive(Debug, Serialize, Deserialize)]
pub struct Origin {
    pub r#type: String, // Escape the keyword `type` using `r#`
    pub reference: String,
}

impl From<&Origin> for String {
    fn from(value: &Origin) -> Self {
        format!("{}: {}", value.r#type, value.reference)
    }
}

// Main struct for each program entry
#[derive(Debug, Serialize, Deserialize)]
pub struct Program {
    title: String,
    description: Option<String>,
    release: Option<String>,
    authors: Option<Vec<String>>,
    images: Option<Vec<String>>,
    pub roms: HashMap<String, Rom>,
    origin: Option<Origin>,
    urls: Option<Vec<String>>,
    copyright: Option<String>,
}

impl Program {
    pub fn get_title(&self) -> &str {
        &self.title
    }

    pub fn get_description(&self) -> &str {
        self.description.as_deref().unwrap_or("")
    }

    pub fn get_release(&self) -> &str {
        self.release.as_deref().unwrap_or("")
    }

    pub fn get_authors(&self) -> String {
        match &self.authors {
            Some(authors) => authors.join(", "),
            None => String::new(),
        }
    }

    #[allow(dead_code)]
    pub fn get_images(&self) -> Option<String> {
        self.images.as_ref().map(|images| images.join(", "))
    }

    pub fn get_origin(&self) -> Option<String> {
        self.origin.as_ref().map(|origin| origin.into())
    }

    pub fn get_urls(&self) -> Option<Vec<String>> {
        self.urls.clone()
    }

    pub fn get_copyright(&self) -> Option<&str> {
        self.copyright.as_deref()
    }
}

// Function to load and parse the embedded programs.json file
fn load_embedded_programs() -> Result<Vec<Program>, Box<dyn std::error::Error>> {
    // Use the include_str! macro to embed the JSON file into the binary
    let json_data = include_str!("../data/programs.json");

    // Parse the JSON into a Vec<Program>
    let programs: Vec<Program> = serde_json::from_str(json_data)?;

    Ok(programs)
}

#[allow(dead_code)]
pub fn get_data() {
    match load_embedded_sha1_hashes() {
        Ok(hash_map) => {
            for (sha1, id) in hash_map {
                println!("SHA1: {}, ID: {}", sha1, id);
            }
        }
        Err(e) => eprintln!("Error reading embedded SHA1_hashes: {}", e),
    }

    match load_embedded_programs() {
        Ok(programs) => {
            for program in programs {
                println!("Title: {}", program.title);
                if let Some(authors) = &program.authors {
                    println!("Authors: {:?}", authors);
                }
                if let Some(release) = &program.release {
                    println!("Release: {}", release);
                }
                if let Some(description) = &program.description {
                    println!("Description: {}", description);
                }
                for (sha1, rom) in &program.roms {
                    println!(
                        "ROM SHA1: {}, File: {}, Platforms: {:?}",
                        sha1, rom.file, rom.platforms
                    );
                    if let Some(description) = &rom.description {
                        println!("ROM Description: {}", description);
                    }
                    if let Some(embedded_title) = &rom.embedded_title {
                        println!("ROM Embedded Title: {}", embedded_title);
                    }
                    if let Some(keys) = &rom.keys {
                        println!("Keys: {:?}", keys);
                    }
                }
                println!();
            }
        }
        Err(e) => eprintln!("Error loading embedded programs: {}", e),
    }
}
