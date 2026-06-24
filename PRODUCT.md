# Product

## Register

product

## Users

StashFlow is used by a technically comfortable individual operator who manages proxy subscription files for Stash and Quantumult X. The user is usually trying to repair or normalize a subscription quickly, often because QX rejects an upstream response before local tooling can process it.

## Product Purpose

StashFlow turns fragile subscription maintenance into a repeatable workflow. The CLI repairs local files; the Cloudflare edge panel will privately store one upstream subscription and expose target-specific QX and Stash subscription URLs that fetch, clean, and return usable configuration on demand.

## Brand Personality

Quiet, precise, private. The interface should feel like an operational control surface: clear status, obvious next actions, no marketing framing, no decorative storytelling.

## Anti-references

Avoid public SaaS landing-page tropes, oversized hero copy, decorative gradients, nested cards, crypto-style dark dashboards, and playful illustrations. Avoid any UI that exposes subscription tokens unnecessarily or makes copy/paste workflows ambiguous.

## Design Principles

1. Make the next operational action obvious.
2. Treat subscription URLs and tokens as sensitive by default.
3. Prefer durable automation over repeated manual conversion.
4. Keep target-specific behavior visible: QX and Stash outputs should be unmistakable.
5. Surface upstream errors plainly instead of hiding them behind generic failure states.

## Accessibility & Inclusion

Target WCAG AA contrast for text and controls. The UI should be keyboard usable, readable on mobile, and should not rely on motion to communicate state. Error text must be specific enough to act on.
