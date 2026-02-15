# Factorio Quality + Productivity Notes

This file summarizes the mechanics we need to model upcycling machine ratios.

## Quality

- Item quality tiers are `Normal`, `Uncommon`, `Rare`, `Epic`, `Legendary`.
- Tier stat multipliers are `1.0x`, `1.3x`, `1.6x`, `1.9x`, `2.5x`.
- Quality chance (`Q`) comes from module effects and penalties.
- `Q` is additive across modules/effects; speed modules apply negative quality.
- Quality chance is floored at `0` (cannot go negative).
- Fluids do not have quality.

### Quality roll behavior

- On craft start, the game rolls for a quality upgrade.
- If the first roll succeeds, there are further rolls at 10x lower chance for each additional tier jump.
- For input quality `Normal` with all tiers unlocked, expected output share by tier is:
  - `Normal = 1 - Q`
  - `Uncommon = 0.9Q`
  - `Rare = 0.09Q`
  - `Epic = 0.009Q`
  - `Legendary = 0.001Q`

If higher tiers are not unlocked yet, probability mass above the cap is folded into the max unlocked tier.

## Productivity

- Productivity bonuses are additive from:
  - Productivity modules
  - Building intrinsic productivity
  - Research productivity bonuses
- Recipe productivity is capped at `+300%` for recipes.
- Mining and research productivity are not recipe productivity and are not capped by that `+300%` recipe cap.
- Productivity duplicates the output when the productivity bar fills.
- The bonus output has the same quality as the crafted output that filled the bar.
- Catalyst recipes only gain productivity on their net output, not returned catalysts.

## Module stats to keep in mind

- Quality modules (Normal quality values):
  - Q1 `+1%`, Q2 `+2%`, Q3 `+2.5%`
- Productivity modules (Normal quality values):
  - P1 `+4%` productivity, `-5%` speed
  - P2 `+6%` productivity, `-10%` speed
  - P3 `+10%` productivity, `-15%` speed
- Speed modules apply negative quality chance:
  - S1 `-1%`, S2 `-1.5%`, S3 `-2.5%` quality

Higher module quality increases positive module effects.

## Upcycling implications for this app

- Recycler recovery behaves like `25%` output (equivalent to `-75%` productivity compared to full recovery).
- Upcycling flow should model:
  - Craft stage quality chance (`Q_craft`)
  - Recycler stage quality chance (`Q_recycle`)
  - Craft stage productivity (`P_craft`)
  - Recipe output amount and recycler recovery amount
- For machine ratio planning, use expected values first; simulation can be added later for variance/rounding effects.

## Sources

- https://wiki.factorio.com/Quality
- https://wiki.factorio.com/Productivity
- https://wiki.factorio.com/Module
- https://wiki.factorio.com/Speed_module
- https://wiki.factorio.com/Productivity_module
- https://wiki.factorio.com/Friday_Facts_#375_-_Quality
