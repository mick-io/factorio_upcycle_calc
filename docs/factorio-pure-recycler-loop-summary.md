# Factorio Pure Recycler Loop: Problem Summary and Project Relevance

Source: https://dfamonteiro.com/posts/factorio-pure-recycler-loop/

## Problem Being Solved
The post models a **pure recycler loop** where:

1. Items go into a recycler.
2. Non-target qualities are fed back into the recycler input.
3. Target quality items (usually legendary) are removed from the loop.

The goal is to compute:

1. Expected production efficiency of the kept quality.
2. How many input items are needed per kept item.
3. Internal recycle flow load needed to sustain the loop.

## Math Model (Core Idea)
The post represents quality flow as a vector and each recycler pass as a matrix transform.

1. Let `f` be the input quality vector (for one belt of normal items: `[1,0,0,0,0]`).
2. Let `R(q)` be the recycler transition matrix for quality chance `q` and recycle output ratio `0.25`.
3. Per-pass state is recursive:
   `s_x = s_(x-1) * R(q)`, with `s_1 = f`.
4. Total steady-state flow is the infinite sum:
   `S = s_1 + s_2 + s_3 + ...`
5. Numerically, this can be computed by iterating until the next pass is negligible.

Equivalent closed form (when stable): `S = f * (I - R)^(-1)`.

## Key Results Highlighted in the Post
For a loop that keeps only legendary output:

1. At `q = 10%`, legendary efficiency is about `0.00693%` (`0.0000693` per 1 normal input), which is about `14,430` normal items per legendary.
2. At `q = 24.8%` (high-end module setup), legendary efficiency is about `0.03667%`, or about `2,726.91` normal items per legendary.
3. The non-legendary entries of `S` represent internal loop throughput; their sum gives loop belt load required to fully consume the input.

## How This Relates to This Project
This app estimates machine/recycler counts and quality output under Factorio quality mechanics. The post directly supports that:

1. It provides a rigorous way to compute **expected quality output rates** from recycler-only loops.
2. It gives a direct path to estimate **recycler support volume** (internal recycle load), which maps to our required recycler count.
3. It reinforces that results should be expectation-based and then converted to machine counts by rounding up.

## Practical Mapping to Current App Fields
In this app:

1. `Recycler Quality` + module selections determine effective quality chance and craft-speed modifiers.
2. `Recycler Input Per Cycle` and `Recycle Cycle Time` convert throughput into per-machine capacity.
3. Required recyclers can be computed as:
   `ceil(recycle_load_per_second / recycler_capacity_per_second)`.

Where:

1. `recycle_load_per_second` can come from the loop steady-state math above.
2. `recycler_capacity_per_second = (effective_craft_speed * base_recycle_input_per_cycle) / base_recycle_time_seconds`.

## Why This Matters for Next Iteration
If we add a dedicated "pure recycler loop" mode, we can compute:

1. Expected output by quality tier.
2. Legendary yield efficiency.
3. Internal recycle burden.
4. Required recycler count from capacity constraints.

That would make recycler sizing in this app mathematically aligned with the model from the post.
