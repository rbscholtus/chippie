# TV artwork: image brief

The prompts behind Chippie's TV skins, and tips for making new ones. How the
art is cut up and used is in [DESIGN.md §12](DESIGN.md#12-tv-artwork).

Written for Gemini but generator-neutral: plain-language prompts, with the
aspect ratio set in the tool and things to avoid listed in the prompt.

**Cabinet: Walnut console** (`classic` skin; front view, flat-on, 4:3):

```
front-facing straight-on product photograph of a mid-1970s tabletop colour
television set, walnut wood-grain veneer cabinet with rounded corners, large
rounded-rectangle picture tube screen on the left two-thirds, the screen is
completely black and switched off with no reflection, brushed aluminium trim
around the screen, control panel on the right with a large rotary channel
selector dial with numbers 2 to 13, a smaller UHF fine-tuning dial, two small
knobs for volume and tint, a rectangular push-button power switch, vertical
fabric speaker grille below the controls, short tapered legs, plain neutral
studio background, soft even lighting, no logo, no brand name, orthographic,
symmetrical, high detail, photorealistic, 4:3 aspect ratio. Avoid: logos,
text, brand names, people, reflections on the screen, perspective, tilt
```

**Cabinet: Portable** (`portable` skin, made 2026-11-14 for more picture and
less cabinet; the same background and lighting phrases keep the skins
consistent):

```
front-facing straight-on product photograph of a compact mid-1970s portable
colour television set, cream and walnut plastic case with rounded corners and
a chrome carrying handle on top, the rounded-rectangle picture tube fills
almost the whole front with only a thin bezel around it, the screen is
completely black and switched off with no reflection, a narrow control strip
below the screen with one rotary channel selector dial without numbers, two
small knobs and a small rocker power switch, small speaker slots in the strip,
plain light grey studio background, soft even lighting, no logo, no brand
name, orthographic, symmetrical, high detail, photorealistic, 4:3 aspect
ratio. Avoid: logos, text, numbers, brand names, people, reflections on the
screen, perspective, tilt
```

**Cabinet: Slim walnut** (`slim` skin, made 2026-11-14 for the most picture:
no handle, feet or control strip, controls in a narrow column at the side):

```
front-facing straight-on product photograph of a mid-1970s colour television
set without legs, feet or handle, a slim walnut wood-grain case with rounded
corners, the rounded-rectangle picture tube fills the whole front from top
to bottom with only a very thin bezel above and below it, the screen is
completely black and switched off with no reflection, a narrow vertical
control column on the right side of the screen with one rotary channel
selector dial without numbers, two small knobs and a small rocker power
switch stacked from top to bottom, plain light grey studio background, soft
even lighting, no logo, no brand name, orthographic, symmetrical, high
detail, photorealistic, 4:3 aspect ratio. Avoid: logos, text, numbers, brand
names, people, handle, legs, feet, stand, control strip below the screen,
reflections on the screen, perspective, tilt
```

**Knobs and buttons** (one per generation, square; not used: the controls are
cut from the cabinets):

```
top-down orthographic photograph of a single 1970s television rotary knob,
chrome rim, black ribbed plastic grip, white indicator line, isolated on a
plain flat white background, soft even lighting, no shadow, high detail,
photorealistic, square 1:1. Avoid: text, logos, perspective
```

Variations: "large channel selector dial with numbers 2 to 13 printed around
the edge", "small ribbed volume knob", "rectangular chrome push-button power
switch, seen straight on".

**Tips:**

- **Generate several, pick the most front-on.** Generators rarely give a
  perfectly straight view. Small skew can be fixed with a perspective
  correction in any image editor.
- **Make the screen area exact.** Paint it pure black and note its rectangle
  and corner radius for `skin.json`. The shader covers it anyway.
- **Remove backgrounds** from knobs (any background-removal tool) and save as
  PNG with transparency. Crop each knob to a square centred on its spindle, so
  it rotates around the right point.
- **Paint out the generated knobs** on the cabinet (leave the dark recess), so
  the separate sprites sit in their place without doubling.
- **Lighting must match** between cabinet and knobs: same "soft even lighting"
  phrase in every prompt.
- **Record the generator, model and prompt** for every image in
  `assets/LICENSES.md`. Check that generator's terms before publishing; the
  terms decide whether and how the images can be used. For Gemini this is
  done: the images are believed to be in the public domain, and Google
  allows personal and commercial use (`assets/LICENSES.md`, "TV artwork").
