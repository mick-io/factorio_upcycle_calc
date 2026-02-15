# Factorio Module Mechanics

Last verified: 2026-02-15

## Overview

Factorio has four module families:

- Speed
- Productivity
- Efficiency
- Quality (Space Age / Quality feature)

Modules are inserted into machine module slots (and some effects can also come from beacons). Module effects stack additively.

## Global Stacking Rules

- Effects are additive, not multiplicative.
- Negative reductions are capped: machine properties like speed, energy consumption, and pollution cannot be reduced below 20% of base.
- Higher module item quality increases module upside values, while drawbacks generally stay fixed.

## Speed Modules

What they do:

- Increase crafting speed.
- Increase energy consumption.
- Reduce quality chance.

Normal-quality module values by tier:

| Tier | Speed | Energy Consumption | Quality Penalty |
| --- | --- | --- | --- |
| Speed 1 | +20% | +50% | -1% |
| Speed 2 | +30% | +60% | -1.5% |
| Speed 3 | +50% | +70% | -2.5% |

Notes:

- Higher module quality increases the speed bonus (for example, legendary speed modules are much stronger).
- Quality bonus cannot go below 0%.

## Productivity Modules

What they do:

- Add productivity (free extra output over time).
- Increase energy use and pollution.
- Reduce crafting speed.

Normal-quality module values by tier:

| Tier | Productivity | Energy Consumption | Speed Penalty | Pollution |
| --- | --- | --- | --- | --- |
| Prod 1 | +4% | +40% | -5% | +5% |
| Prod 2 | +6% | +60% | -10% | +7% |
| Prod 3 | +10% | +80% | -15% | +10% |

Notes:

- Productivity bonus scales with module quality.
- Productivity modules are recipe-restricted (mostly intermediate products, with explicit exceptions).
- Productivity modules cannot be inserted into beacons.

## Efficiency Modules

What they do:

- Reduce machine energy consumption (and therefore usually reduce pollution).

Normal-quality module values by tier:

| Tier | Energy Consumption |
| --- | --- |
| Eff 1 | -30% |
| Eff 2 | -40% |
| Eff 3 | -50% |

Notes:

- Energy reduction still follows the global floor (minimum 20% of base usage).
- Over-80% total efficiency can still be useful if other modules increase power use.

## Quality Modules

What they do:

- Increase chance of producing higher-quality outputs.
- Apply a speed penalty.

Normal-quality module values by tier:

| Tier | Quality Chance | Speed Penalty |
| --- | --- | --- |
| Quality 1 | +1% | -5% |
| Quality 2 | +2% | -5% |
| Quality 3 | +2.5% | -5% |

Notes:

- All quality module tiers share the same speed penalty.
- Quality chance scales with module quality.
- Speed-module quality penalties directly counteract quality-module gains.

## How Quality Rolls Work

For quality crafting:

- A machine uses the summed quality chance from its modules/effects.
- On each craft, the game makes a quality roll.
- On success, the result is upgraded one quality tier above the ingredient quality.

Quality tiers:

- Normal
- Uncommon
- Rare
- Epic
- Legendary

## Sources

- https://wiki.factorio.com/Module
- https://wiki.factorio.com/Speed_module
- https://wiki.factorio.com/Productivity_module
- https://wiki.factorio.com/Efficiency_module
- https://wiki.factorio.com/Quality_module
- https://wiki.factorio.com/Quality
