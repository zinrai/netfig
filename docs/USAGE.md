# Usage

Field-by-field reference, command-line flags, and the catalogue of
validation errors. For full topologies see [`examples/`](../examples/).

## Command line

```
netfig FILE
```

Reads a YAML topology description from FILE and writes SVG to
stdout. On any rule violation the run exits non-zero with a single
error line on stderr; nothing is written to stdout.

Example:

```
netfig topology.yaml > diagram.svg
```

## YAML structure

The file has six top-level keys: `purpose`, `legend`, `layout`,
`nodes`, `links`, `groups`. The `groups` key is optional; the other
five are the same as before. See the example YAMLs under
[`examples/`](../examples/) for full topologies; the field semantics
below cover every key.

### `purpose`

Metadata only — not used by the tool. Two free-form strings:

- `audience` — who is meant to read the diagram.
- `intent` — what they should learn from it.

### `legend.symbols`

A map from role name to shape. Every node's `role` must appear
here; otherwise validation fails.

```yaml
legend:
  symbols:
    firewall: { shape: rect }
    host:     { shape: ellipse }
```

Supported shapes: `rect`, `ellipse`.

### `legend.line_kinds`

Named line styles for links.

```yaml
legend:
  line_kinds:
    fiber:   { style: solid,  width: 2 }
    planned: { style: dashed, meaning: "future capacity" }
```

- `style` — `solid`, `dashed`, or `dotted`.
- `width` — integer; values ≥ 2 emit a heavier stroke.
- `meaning` — required for any non-solid style; the reader needs to
  know what the deviation conveys.

The legend may declare at most three distinct styles total.

### `legend.patterns`

Named visual conventions for groups of related links. A pattern
applies to two or more links that share endpoints and share the
pattern name. The rendered output for such a group is a set of
parallel runs; the pattern's `meaning` tells the reader what that
parallel-run convention represents in this diagram (a redundant
pair, an ECMP bundle, an active/standby relationship, and so on).

```yaml
legend:
  patterns:
    redundant_pair:
      meaning: "Primary and secondary on the same endpoints. Both are operationally active; secondary carries higher cost."
```

- `meaning` — required for every pattern; without it the visual
  convention is undeclared.

Patterns are referenced by links via the `pattern` field; see the
`links` section for usage.

### `layout.bands`

Ordered list of horizontal bands, top-to-bottom. The first band is
drawn at the top of the diagram, encoding an upstream-to-downstream
axis.

```yaml
layout:
  bands:
    - name: external
      roles: [firewall]
    - name: core
      roles: [l3-switch]
```

A given role may appear in only one band.

### `layout.locations`

Map from location name to column index (a string-encoded integer,
left-to-right starting at `"0"`).

```yaml
layout:
  locations:
    osaka: "0"
    tokyo: "1"
```

The horizontal axis carries whatever semantic the YAML author
chose (typically geographical). The tool does not enforce a
specific meaning.

### `nodes`

```yaml
nodes:
  - id: tk1-fw01
    role: firewall
    location: tokyo
    label: tk1-fw01    # optional, defaults to id
```

The optional metadata fields `vendor`, `model`, `ip`, `vlan` are
accepted on a node and ignored by the renderer.

### `links`

```yaml
links:
  - from: tk1-fw01
    to:   tk1-core01
    label: uplink                   # optional
    kind:  fiber                    # optional; must reference legend.line_kinds
    pattern: redundant_pair         # optional; must reference legend.patterns
```

A link's `kind`, if set, must reference an entry in
`legend.line_kinds`. Empty `kind` means "default solid line".

A link's `pattern`, if set, must reference an entry in
`legend.patterns`. Setting `pattern` declares that this link is part
of a group with other links sharing the same endpoints, the same
`kind`, and the same pattern name. See "Sibling links" below.

**Sibling links.** Two or more links between the same pair of
endpoints (in either direction) that also share the same `kind`
form a sibling group. Every member of such a group must declare the
same `pattern`; netfig refuses to render a same-kind duplicate
without a declared pattern, because the writer's intent is
ambiguous — the two links could be a redundant pair or two
unrelated same-kind relationships that happen to share endpoints.
The legend's pattern declaration tells the reader what the parallel
runs mean; the reader does not have to guess.

Two links between the same endpoints with *different* kinds are not
treated as siblings. Their kinds already declare distinct visual
meanings (one is solid, the other dashed, and so on), so they are
not parallel runs of the same thing — they are two separate lines
representing two separate relationships, and both render on their
default lanes. A common real-world case is an intra-site physical
link drawn alongside an iBGP session that rides over it.

```yaml
legend:
  patterns:
    redundant_pair:
      meaning: "Primary and secondary on the same endpoints. Both are operationally active; secondary carries higher cost."

links:
  - { from: core1, to: core2, kind: ebgp, label: primary,            pattern: redundant_pair }
  - { from: core1, to: core2, kind: ebgp, label: "secondary cost30", pattern: redundant_pair }
```

The two links render as parallel ebgp runs. Each link's `label` is
rendered on its own run. The column layout reserves extra horizontal
space in the gap between the two endpoints' columns so the parallel
runs and their labels do not crowd the neighbouring columns.

### `groups`

Optional. Draws a visual cluster boundary around a rectangular region
of the `(band, location)` grid. The intended use is "same site",
"same failure domain", "same administrative scope": a region the
reader's eye should pick out as a unit.

```yaml
groups:
  - name: site-a
    locations: [site-a]              # required
    bands: [core, route-reflector]   # optional; defaults to all bands
    label: "site-a (AS 64601)"       # optional; defaults to name
```

`locations` lists the columns the group covers; `bands` optionally
narrows the rows. The rendered rectangle is the bounding box of the
listed cells, drawn as a filled rounded rectangle without an outline
and emitted under every node and link so it does not interfere with
them.

## Validation

netfig fails the run on every rule violation rather than warning.
There is no "permissive" mode: if the tool cannot produce a diagram
faithful to the input, or if the input would produce a diagram
known to be hard to read, the run exits non-zero and nothing is
written to stdout.

Input integrity:

- duplicate node id
- empty role
- node uses an undefined location
- link references an unknown node
- self-link

Legend:

- role used by a node is not declared in `legend.symbols`
- shape is empty or unsupported (only `rect`, `ellipse` allowed)
- line kind has empty or unsupported style (only `solid`, `dashed`, `dotted`)
- non-solid line kind has no `meaning`
- legend declares more than three distinct line styles
- link uses a `kind` not in `legend.line_kinds`
- pattern has no `meaning`
- link uses a `pattern` not in `legend.patterns`

Patterns:

- two links share endpoints and kind but neither declares a pattern
- two links share endpoints and kind and one declares a pattern but
  the other does not
- two links share endpoints and kind but declare different patterns

Layout:

- role appears in more than one band
- a `(band, location)` cell holds more than `maxNodesPerCell` (12)
  nodes — split the band or the location to keep the cell readable

Groups:

- group has empty `name`
- group has empty `locations`
- group references an unknown location
- group references an unknown band


