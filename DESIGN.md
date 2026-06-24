# Design

## System

StashFlow Cloud is a private product UI. Design serves the task: authenticate, save one upstream subscription, copy target-specific subscription URLs, and understand upstream health.

## Color

Use a restrained operational palette in OKLCH:

```css
:root {
  --bg: oklch(0.985 0 0);
  --surface: oklch(0.955 0.004 242);
  --surface-strong: oklch(0.925 0.010 242);
  --ink: oklch(0.185 0.018 248);
  --muted: oklch(0.455 0.022 248);
  --primary: oklch(0.515 0.140 242);
  --primary-strong: oklch(0.430 0.135 242);
  --accent: oklch(0.620 0.145 168);
  --danger: oklch(0.550 0.170 28);
  --warning: oklch(0.640 0.150 72);
  --success: oklch(0.570 0.140 154);
  --line: oklch(0.875 0.008 242);
  --focus: oklch(0.690 0.130 242);
}
```

Primary blue is used for active controls and selected output links. Green accent is reserved for healthy state and copy success. Error and warning colors are functional only.

## Typography

Use `Inter`, `ui-sans-serif`, `system-ui`, `-apple-system`, `BlinkMacSystemFont`, and `Segoe UI`. Product type is fixed-scale, not viewport-fluid:

- Page title: 28px / 1.15, 700
- Section heading: 16px / 1.3, 700
- Body: 14px / 1.55, 450
- Label: 12px / 1.2, 650
- Code and URLs: `ui-monospace`, `SFMono-Regular`, `Menlo`, monospace

## Layout

Use a top bar, a constrained main workspace, and two primary panels on desktop. Collapse to one column on mobile. Controls should remain stable when status text changes.

## Components

Buttons use one vocabulary: primary filled, secondary bordered, danger text. Inputs have visible labels, focus rings, and inline error text. URL fields use monospace and copy buttons. Empty state should guide the user to add a subscription, not describe the product.

## Motion

Use 150-200ms transitions for hover, focus, and toast feedback. Respect `prefers-reduced-motion: reduce`.
