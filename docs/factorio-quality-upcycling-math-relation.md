# Factorio Quality Upcycling Math: Relevance to This Project

Source: https://wiki.factorio.com/Tutorial:Quality_upcycling_math

## What the Tutorial Solves
The wiki tutorial answers two practical optimization questions for quality upcycling:

1. What quality/productivity module mix is best per crafting tier.
2. How many crafting machines and recyclers are needed per tier to sustain the loop and maximize legendary output.

It does this with expected-value math, not simulation.

## Core Math Ideas in the Tutorial
The model is discrete and matrix-based:

1. **Quality jump probabilities**: when an item upgrades, jump sizes follow 90% (+1 tier), 9% (+2), 0.9% (+3), 0.1% (+4), capped at legendary.
2. **Assembler transform**: a quality/productivity matrix maps input-tier materials to output-tier products with per-tier module effects.
3. **Recycler transform**: a second matrix models recycling, including:
   - recycler anti-productivity (-75% effective item return),
   - quality-module-only recycler setup,
   - excluding legendary items from recycle input.
4. **Combined loop model**: assembler and recycler transforms are composed into one loop matrix to project quality distribution and throughput over repeated cycles.

The tutorial then derives:

1. Best module ratios by machine type and module quality tier.
2. Machine-count ratios per quality tier (plus recycler requirements).

## How This Relates to This App
This app is solving the same planning problem at UI level:

1. Users choose upcycling target quality and machine/recycler setups.
2. The app computes machine allocation, recycler needs, and output/recycle rates.
3. Results are converted to deployable machine counts with rounding up.

The tutorial provides a stronger mathematical foundation for those outputs:

1. It validates using expected-value transitions between quality tiers.
2. It supports computing tier-specific machine ratios from loop equilibrium.
3. It directly informs recycler demand from recycle-flow load.

## Practical Mapping to Current Fields
Current app inputs/outputs already align with tutorial variables:

1. **Machine stats and modules** map to assembler transform parameters.
2. **Recycler quality/modules + recycle cycle time** map to recycler transform capacity.
3. **Machines dedicated per quality + required recyclers** map to the derived loop ratios and throughput constraints.
4. **Target modules/hour and recycled/hour** map to steady-state flow outputs of the combined loop.

## Current Gap vs. Tutorial
The tutorial optimizes module mix **per quality tier machine group** and **per machine type**.  
The current app still uses a simplified single-configuration flow (then scales/allocates), so it does not yet fully reproduce the tutorial tables.

## Recommended Next Step
To match the tutorial more closely, add a server-side matrix pipeline:

1. Build assembler matrix `A` from per-tier module choices and machine stats.
2. Build recycler matrix `R` from recycler quality setup and anti-productivity.
3. Combine into loop matrix `L` (with legendary carry-over handling).
4. Solve for steady-state tier flows.
5. Convert flows to:
   - machine counts per tier,
   - recycler counts from recycle load/capacity,
   - hourly target output and recycled throughput.

This would make the app mathematically consistent with the official tutorial’s optimization method.
