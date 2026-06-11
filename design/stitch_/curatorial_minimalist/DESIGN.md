---
name: Curatorial Minimalist
colors:
  surface: '#fdf8f8'
  surface-dim: '#ddd9d8'
  surface-bright: '#fdf8f8'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f7f3f2'
  surface-container: '#f1edec'
  surface-container-high: '#ebe7e6'
  surface-container-highest: '#e5e2e1'
  on-surface: '#1c1b1b'
  on-surface-variant: '#444748'
  inverse-surface: '#313030'
  inverse-on-surface: '#f4f0ef'
  outline: '#747878'
  outline-variant: '#c4c7c7'
  surface-tint: '#5f5e5e'
  primary: '#000000'
  on-primary: '#ffffff'
  primary-container: '#1c1b1b'
  on-primary-container: '#858383'
  inverse-primary: '#c9c6c5'
  secondary: '#5e5e65'
  on-secondary: '#ffffff'
  secondary-container: '#e3e1ea'
  on-secondary-container: '#64646b'
  tertiary: '#000000'
  on-tertiary: '#ffffff'
  tertiary-container: '#1c1b1b'
  on-tertiary-container: '#858383'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#e5e2e1'
  primary-fixed-dim: '#c9c6c5'
  on-primary-fixed: '#1c1b1b'
  on-primary-fixed-variant: '#474646'
  secondary-fixed: '#e3e1ea'
  secondary-fixed-dim: '#c7c5ce'
  on-secondary-fixed: '#1b1b21'
  on-secondary-fixed-variant: '#46464d'
  tertiary-fixed: '#e5e2e1'
  tertiary-fixed-dim: '#c8c6c5'
  on-tertiary-fixed: '#1c1b1b'
  on-tertiary-fixed-variant: '#474646'
  background: '#fdf8f8'
  on-background: '#1c1b1b'
  surface-variant: '#e5e2e1'
typography:
  display-lg:
    fontFamily: Noto Serif
    fontSize: 48px
    fontWeight: '400'
    lineHeight: '1.1'
    letterSpacing: -0.02em
  display-lg-mobile:
    fontFamily: Noto Serif
    fontSize: 32px
    fontWeight: '400'
    lineHeight: '1.2'
  headline-md:
    fontFamily: Noto Serif
    fontSize: 32px
    fontWeight: '400'
    lineHeight: '1.2'
  headline-sm:
    fontFamily: Noto Serif
    fontSize: 24px
    fontWeight: '400'
    lineHeight: '1.3'
  body-lg:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '400'
    lineHeight: '1.6'
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: '1.6'
  label-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '500'
    lineHeight: '1.4'
    letterSpacing: 0.01em
  label-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: '1.2'
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  unit: 4px
  container-max: 1120px
  gutter: 24px
  margin-mobile: 16px
  margin-desktop: 48px
  stack-sm: 8px
  stack-md: 16px
  stack-lg: 32px
  section-gap: 80px
---

## Brand & Style

This design system is built upon a philosophy of **Curatorial Minimalism**. It prioritizes clarity, breathing room, and a humanistic touch for participatory cultural events. The visual direction is "type-driven," where information hierarchy is established through meticulous typographic scale rather than heavy ornamentation.

The aesthetic evokes the feeling of a modern art gallery: clean white surfaces, precise hairline boundaries, and a sophisticated interplay between functional UI and expressive editorial accents. The tone is intellectual yet accessible, shifting the focus from the interface to the cultural content and human mediation it facilitates.

## Colors

The palette is intentionally restrained to maintain a high-contrast, "ink-on-paper" feel. 

- **Primary Text (#0A0A0A):** Used for maximum legibility in body text and primary headers.
- **Secondary Text (#3F3F46):** Softens meta-information and labels to reduce visual noise.
- **Background (#FFFFFF):** Pure white serves as the primary canvas to maximize whitespace.
- **Surface (#FAFAF7):** A subtle, warm off-white used for cards and section containment to provide soft differentiation without breaking the minimalist flow.
- **Hairline Borders (#ECECEC):** The primary tool for structural separation, replacing shadows entirely.

## Typography

The typography system uses a pairing of **Noto Serif** for curatorial expression and **Inter** for functional clarity.

- **Noto Serif:** Reserved for headlines, quotes, and event titles. It provides a literary, academic, and humanistic quality to the interface.
- **Inter:** Used for all UI elements, navigation, inputs, and long-form body text. Its neutral, systematic nature ensures that the "utility" of the platform remains unobtrusive.
- **Language Support:** All scales are optimized for Russian Cyrillic, ensuring appropriate line-heights to prevent descender clashing in dense text blocks.

## Layout & Spacing

The layout philosophy follows a **Fixed Grid** model for desktop to maintain a composed, editorial feel, while transitioning to a fluid model for mobile.

- **Whitespace:** Use generous `section-gap` values to separate different types of content (e.g., separating the event description from the registration module).
- **Alignment:** Content is generally center-aligned in the container for a focused, "invitation-style" layout.
- **Rhythm:** Spacing follows a 4px base unit. Vertical rhythm is critical; maintain consistent `stack-lg` spacing between primary content blocks to avoid visual clutter.

## Elevation & Depth

This design system rejects physical shadows in favor of **Tonal Layers** and **Hairline Outlines**.

- **Depth through Contrast:** Depth is created by placing `#FAFAF7` (Surface) elements against the `#FFFFFF` (Background).
- **Hairline Borders:** Use 1px borders in `#ECECEC` for all structural elements (cards, input fields, dividers).
- **Interaction:** On hover, a border can transition to a slightly darker `#D4D4D8` or the surface can shift color slightly, but no "lifting" shadow should be applied. This maintains the flat, paper-like aesthetic characteristic of high-end cultural institutions.

## Shapes

The shape language combines geometric precision with extreme softness for primary touchpoints.

- **Cards:** Use a generous `16px` (2xl) radius to soften the presentation of event images and containers.
- **Interactive Elements:** Buttons and tags utilize a **Pill-shape** (full radius) to make them feel distinct from the structural grid and more "participatory" and friendly.
- **Inputs:** Maintain the standard system roundedness (8px) to keep them feeling functional and grounded.

## Components

### Buttons
- **Primary:** Black background (#111111), white text, pill-shaped. No border.
- **Secondary:** White background, black text, pill-shaped, 1px hairline border (#ECECEC).
- **Ghost:** No background, black text, underline on hover.

### Cards
- Background: `#FAFAF7`.
- Border: 1px `#ECECEC`.
- Radius: `16px`.
- Padding: `24px` for content. Images should be top-aligned with no radius on the bottom corners if they bleed into the card, or fully inset with a `12px` radius.

### Input Fields
- Background: `#FFFFFF`.
- Border: 1px `#ECECEC` (Focus: `#0A0A0A`).
- Radius: `8px`.
- Text: Inter 16px.

### Chips & Tags
- Pill-shaped, small uppercase labels (`label-sm`).
- Light gray background (#F4F4F5) or just a hairline border.

### Lists
- Clean, unstyled lists with 1px dividers between items. 
- Use Noto Serif for list titles and Inter for metadata to maintain the curatorial hierarchy.