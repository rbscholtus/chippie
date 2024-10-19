#![allow(dead_code)]
use crate::chip8;
use crate::{N, NN, NNN, X, Y};

use std::fmt;

pub struct CPU {
    pub bus: chip8::Bus,
    pub keys_down: [bool; 16],
    pub pc: u16,
    i: u16,
    stack: Vec<u16>,
    delay_timer: u8,
    pub sound_timer: u8,
    v: [u8; 16],
    key_pressed: Option<usize>,
}

impl CPU {
    pub fn new() -> Self {
        let mut cpu = CPU {
            bus: chip8::Bus::new(),
            keys_down: [false; 16],
            pc: 0x200,
            i: 0,
            stack: Vec::new(),
            delay_timer: 0,
            sound_timer: 0,
            v: [0_u8; 16],
            key_pressed: None,
        };
        cpu.bus.save_byte(0x200, 0x12);
        // cpu.bus.save_byte(0x201, 0x1200);
        cpu
    }
}

impl fmt::Debug for CPU {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // Start the struct formatting
        let mut debug_struct = f.debug_struct("CPU");

        // Format `pc` and `i` with 4 hexadecimal characters without 0x prefix
        debug_struct.field("pc", &format_args!("{:04X}", self.pc));
        let opcode: u16 =
            ((self.bus.read_byte(self.pc) as u16) << 8) | (self.bus.read_byte(self.pc + 1) as u16);
        debug_struct.field("op", &format_args!("{:04X}", opcode));

        debug_struct.field("i", &format_args!("{:04X}", self.i));
        let idata: u16 =
            ((self.bus.read_byte(self.i) as u16) << 8) | (self.bus.read_byte(self.i + 1) as u16);
        debug_struct.field("data", &format_args!("{:04X}", idata));

        // Format `v` array with 2 hexadecimal characters per element without 0x prefix
        let v_str: String = self
            .v
            .iter()
            .map(|&x| format!("{:02X}", x))
            .collect::<Vec<_>>()
            .join(" ");
        debug_struct.field("v", &v_str);

        // Format `inp` with a single digit number for each element
        let inp_str: String = self
            .keys_down
            .iter()
            .map(|&b| if b { 'X' } else { '-' })
            .collect();
        debug_struct.field("inp", &inp_str);

        // Format `delay_timer` and `sound_timer` with 2 hexadecimal digits without 0x prefix
        // debug_struct.field("delay_timer", &format_args!("{:02X}", self.delay_timer))
        //              .field("sound_timer", &format_args!("{:02X}", self.sound_timer));

        // Finish formatting, ignoring `bus` and `stack`
        debug_struct.finish()
    }
}

pub fn fmt_opcode(opcode: u16) -> String {
    match opcode & 0xf000 {
        0x0000 => match opcode & 0xfff {
            0x0e0 => format!("00E0 Clear"),
            0x0ee => format!("00EE Return"),
            _ => "0___ Not implemented".to_string(),
        },
        0x1000 => format!("1NNN Jump to {}", NNN!(opcode)),
        0x2000 => format!("2NNN Call sub at {}", NNN!(opcode)),
        0x3000 => format!("3XNN Skip if V{} equals {}", X!(opcode), NN!(opcode)),
        0x4000 => format!("4XNN Skip if V{} not equals {}", X!(opcode), NN!(opcode)),
        0x5000 => format!("5XY0 Skip if V{} equals V{}", X!(opcode), Y!(opcode)),
        0x6000 => {
            format!("6XNN Set V{} to {}", X!(opcode), NN!(opcode))
        }
        0x7000 => {
            format!("7XNN Add {} to V{}", NN!(opcode), X!(opcode))
        }
        0x8000 => match opcode & 0xf {
            0x0 => {
                format!("8XY0 Set V{} to V{}", X!(opcode), Y!(opcode))
            }
            0x1 => format!(
                "8XY1 Set V{} to V{} OR V{}",
                X!(opcode),
                X!(opcode),
                Y!(opcode)
            ),
            0x2 => format!(
                "8XY2 Set V{} to V{} AND V{}",
                X!(opcode),
                X!(opcode),
                Y!(opcode)
            ),
            0x3 => format!(
                "8XY3 Set V{} to V{} XOR V{}",
                X!(opcode),
                X!(opcode),
                Y!(opcode)
            ),
            0x4 => format!("8XY4 Add V{} to V{} with carry", Y!(opcode), X!(opcode)),
            0x5 => format!("8XY5 Sub V{} from V{} with carry", Y!(opcode), X!(opcode)),
            0x6 => format!(
                "8XY6 Set V{} to V{}>>1 with carry (VIP impl)",
                X!(opcode),
                Y!(opcode)
            ),
            0x7 => format!(
                "8XY7 Set V{} to V{}-V{} with carry",
                X!(opcode),
                Y!(opcode),
                X!(opcode)
            ),
            0xe => format!(
                "8XYE Set V{} to V{}<<1 with carry (VIP impl)",
                X!(opcode),
                Y!(opcode)
            ),
            _ => "8XY_ Invalid".to_string(),
        },
        0x9000 => format!("9XY0 Skip if V{} not equals V{}", X!(opcode), Y!(opcode)),
        0xa000 => format!("ANNN Set I to {}", NNN!(opcode)),
        0xb000 => format!("BNNN Jump to {} + V0", NNN!(opcode)),
        0xc000 => format!("CNNN Not implemented yet"),
        0xd000 => format!(
            "DXYN Display {} rows at V{},V{} with carry",
            N!(opcode),
            X!(opcode),
            Y!(opcode)
        ),
        0xe000 => match opcode & 0xff {
            0x9e => format!("EX9E Skip if inp[V{}]", X!(opcode)),
            0xa1 => format!("EXA1 Skip if not inp[V{}]", X!(opcode)),
            _ => "E___ Invalid".to_string(),
        },
        0xf000 => match opcode & 0xff {
            0x07 => {
                format!("FX07 Set V{} to delay timer", X!(opcode))
            }
            0x0a => format!("FX0A Set V{} to first inp, or decr PC", X!(opcode)),
            0x15 => {
                format!("FX15 Set delay timer to V{}", X!(opcode))
            }
            0x18 => {
                format!("FX18 Set sound timer to V{}", X!(opcode))
            }
            0x1e => format!("FX1E Add V{} to I", X!(opcode)),
            0x29 => format!("FX29 Set I to addr of font char in V{}", X!(opcode)),
            0x33 => format!("FX33 Store BCD of V{} in M[I], incr I", X!(opcode)),
            0x55 => {
                format!("FX55 Store V0..V{} in M[I], incr I (VIP impl)", X!(opcode))
            }
            0x65 => format!("FX65 Load V0..V{} from M[I], incr I (VIP impl)", X!(opcode)),
            _ => "FX__ Invalid".to_string(),
        },
        _ => "____ Invalid".to_string(),
    }
}

impl CPU {
    pub fn get_op(&self) -> u16 {
        ((self.bus.read_byte(self.pc) as u16) << 8) | (self.bus.read_byte(self.pc + 1) as u16)
    }

    pub fn ticks(&mut self, ticks: u16) {
        for _ in 0..ticks {
            self.tick();
        }
        self.decr_timers();
    }

    pub fn decr_timers(&mut self) {
        if self.delay_timer > 0 {
            self.delay_timer -= 1;
        }
        if self.sound_timer > 0 {
            self.sound_timer -= 1;
        }
    }

    fn tick(&mut self) {
        let opcode: u16 = self.get_op();
        self.pc += 2;

        match opcode & 0xf000 {
            0x0000 => {
                match opcode & 0xfff {
                    0x0e0 => {
                        // clear screen 00E0
                        self.bus.gpu.clear();
                    }
                    0x0ee => {
                        // Returning from a subroutine 00EE
                        self.pc = self.stack.pop().unwrap();
                    }
                    _ => {
                        // Execute machine language routine 0NNN
                        // don't implement
                    }
                }
            }
            0x1000 => {
                // jump to NNN
                self.pc = NNN!(opcode);
            }
            0x2000 => {
                // 2NNN - calls the subroutine at memory location NNN
                self.stack.push(self.pc);
                self.pc = NNN!(opcode);
            }
            0x3000 => {
                // 3XNN - skip one instruction if the value in VX is equal to NN
                if self.v[X!(opcode)] == NN!(opcode) {
                    self.pc += 2;
                }
            }
            0x4000 => {
                // 4XNN - skip one instruction if the value in VX is not equal to NN
                if self.v[X!(opcode)] != NN!(opcode) {
                    self.pc += 2;
                }
            }
            0x5000 => {
                // 5XY0 - skips if the values in VX and VY are equal
                if self.v[X!(opcode)] == self.v[Y!(opcode)] {
                    self.pc += 2;
                }
            }
            0x6000 => {
                // 6XNN - set register VX
                self.v[X!(opcode)] = NN!(opcode);
            }
            0x7000 => {
                // 7XNN - add value to register VX
                self.v[X!(opcode)] = self.v[X!(opcode)].wrapping_add(NN!(opcode));
            }
            0x8000 => {
                let x = X!(opcode);
                let y = Y!(opcode);
                match opcode & 0x000f {
                    0x0 => {
                        // VX is set to the value of VY 8XY0
                        self.v[x] = self.v[y];
                    }
                    0x1 => {
                        // VX is set to the bitwise OR of VX and VY
                        self.v[x] |= self.v[y];
                        // COSMAC VIP behavior undefined
                        // Timendus' test suite requires consistency
                        self.v[0xf] = 0;
                    }
                    0x2 => {
                        // VX is set to the bitwise AND of VX and VY
                        self.v[x] &= self.v[y];
                        // COSMAC VIP behavior undefined
                        // Timendus' test suite requires consistency
                        self.v[0xf] = 0;
                    }
                    0x3 => {
                        // VX is set to the bitwise XOR of VX and VY
                        self.v[x] ^= self.v[y];
                        // COSMAC VIP behavior undefined
                        // Timendus' test suite requires consistency
                        self.v[0xf] = 0;
                    }
                    0x4 => {
                        // 8XY4 - Add VY to VX with carry
                        let (result, carry) = self.v[x].overflowing_add(self.v[y]);
                        self.v[x] = result;
                        self.v[0xF] = carry as u8;
                    }
                    0x5 => {
                        // 8XY5 - set VX to the result of VX - VY
                        let flag = match self.v[x] >= self.v[y] {
                            true => 1,
                            false => 0,
                        };
                        self.v[X!(opcode)] = self.v[X!(opcode)].wrapping_sub(self.v[Y!(opcode)]);
                        self.v[0xf] = flag;
                    }
                    0x6 => {
                        // 8XY6 - Shift right with carry
                        // original COSMAC VIP=true
                        self.v[x] = self.v[y];
                        let flag = self.v[x] & 0x1;
                        self.v[x] >>= 1;
                        self.v[0xf] = flag;
                    }
                    0x7 => {
                        // 8XY7 - set VX to the result of VY - VX
                        let flag = if self.v[y] >= self.v[x] { 1 } else { 0 };
                        self.v[x] = self.v[y].wrapping_sub(self.v[x]);
                        self.v[0xf] = flag;
                    }
                    0xe => {
                        // 8XYE - Shift left with carry
                        // original COSMAC VIP=true
                        self.v[x] = self.v[y];
                        let flag = self.v[x] >> 7;
                        self.v[x] <<= 1;
                        self.v[0xf] = flag;
                    }
                    _ => {
                        eprintln!(">>opcode {:04x} invalid<<", opcode);
                    }
                }
            }
            0x9000 => {
                // 9XY0 - skips if the values in VX and VY are not equal
                if self.v[X!(opcode)] != self.v[Y!(opcode)] {
                    self.pc += 2;
                }
            }
            0xa000 => {
                // ANNN - set index register I
                self.i = NNN!(opcode);
            }
            0xb000 => {
                // BNNN Jump to NNN plus V0
                // COSMAC VIP implementation crashes Timendus test
                // self.pc = NNN!(opcode) + self.v[0] as u16;
                self.pc = NNN!(opcode) + self.v[X!(opcode)] as u16;
            }
            0xc000 => {
                // CXNN - Random number AND NN
                // first op to use Rust macros
                let salt: u8 = rand::random();
                self.v[X!(opcode)] = salt & NN!(opcode);
            }
            0xd000 => {
                // DXYN - display/draw
                self.op_DXYN(X!(opcode), Y!(opcode), N!(opcode));
            }
            0xe000 => {
                match opcode & 0xff {
                    0x9e => {
                        // EX9E - Skip if key VX is pressed
                        let vx = self.v[X!(opcode)];
                        if self.keys_down[(vx % self.keys_down.len() as u8) as usize] {
                            self.pc += 2;
                        }
                    }
                    0xa1 => {
                        // EXA1 - Skip if key VX is not pressed
                        let vx = self.v[X!(opcode)];
                        if !self.keys_down[(vx % self.keys_down.len() as u8) as usize] {
                            self.pc += 2;
                        }
                    }
                    _ => {
                        eprintln!(">>opcode {:04x} invalid<<", opcode);
                    }
                }
            }
            0xf000 => {
                let x = ((opcode & 0x0f00) >> 8) as usize;
                match opcode & 0xff {
                    0x07 => {
                        // FX07 - sets VX to the current value of the delay timer
                        self.v[x] = self.delay_timer;
                    }
                    0x0a => {
                        // FX0A - Get key
                        self.op_FX0A(x);
                    }
                    0x15 => {
                        // FX15 - sets the delay timer to the value in VX
                        self.delay_timer = self.v[x];
                    }
                    0x18 => {
                        // FX18 - sets the sound timer to the value in VX
                        self.sound_timer = self.v[x];
                    }
                    0x1e => {
                        // FX1E - Add VX to index
                        let vx: u8 = self.v[((opcode & 0x0f00) >> 8) as usize];
                        self.i += vx as u16;
                    }
                    0x29 => {
                        // FX29 - Font character
                        self.i = 0x0050 + (self.v[x] & 0xf) as u16;
                    }
                    0x33 => {
                        // FX33 - Binary-coded decimal conversion
                        let vx: u8 = self.v[((opcode & 0x0f00) >> 8) as usize];
                        self.bus.save_byte(self.i, vx / 100);
                        self.bus.save_byte(self.i + 1, vx / 10 % 10);
                        self.bus.save_byte(self.i + 2, vx % 10);
                    }
                    0x55 => {
                        // FX55 - store registers to memory
                        // original COSMAC VIP
                        for n in 0..x + 1 {
                            self.bus.save_byte(self.i, self.v[n]);
                            self.i += 1;
                        }
                    }
                    0x65 => {
                        // FX65 - load registers from memory
                        // original COSMAC VIP
                        for n in 0..x + 1 {
                            self.v[n] = self.bus.read_byte(self.i);
                            self.i += 1;
                        }
                    }
                    _ => {
                        eprintln!(">>F opcode {:04x} invalid<<", opcode);
                    }
                }
            }
            _ => {
                panic!(">>opcode {:04x} invalid<<", opcode);
            }
        };
    }

    #[allow(non_snake_case)]
    // DXYN - display/draw
    fn op_DXYN(&mut self, x: usize, y: usize, n: u8) {
        let x_coord = self.v[x] & 0x3f;
        let mut y_coord = self.v[y] & 0x1f;
        self.v[0xf] = 0;
        for address in self.i..self.i + n as u16 {
            if self.bus.display(x_coord, y_coord, address) {
                self.v[0xf] = 1;
            }
            y_coord += 1;
        }
    }

    #[allow(non_snake_case)]
    /// Fx0A GETKEY
    fn op_FX0A(&mut self, x: usize) {
        self.pc -= 2;
        match self.key_pressed {
            None => {
                self.key_pressed = self
                    .keys_down
                    .iter()
                    .enumerate()
                    .find_map(|(index, &value)| value.then(|| index));
            }
            Some(key) => {
                if self.keys_down[key] {
                    self.sound_timer = 4;
                } else if self.sound_timer == 0 {
                    self.pc += 2;
                    self.v[x] = key as u8;
                    self.key_pressed = None;
                }
            }
        };
    }
}
