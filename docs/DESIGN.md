# Design

## What netfig is built for

I believe that network diagrams, like source code, are read and
maintained for far longer than the time required to create them.
netfig is built around that long-term use: it aims to keep
diagrams easier for later readers to understand and consistent
through repeated edits, with the trade-off that the writer's
up-front work increases somewhat.

My starting point was Go's design philosophy. In Go, I feel that
code written by different developers takes on an overall similar
appearance, and I wanted to take the same approach for network
diagrams in netfig. I do not see this as the only right answer —
just as one possible approach.

## The trade-off

netfig is built for one specific situation: a diagram being read
by someone who did not write it, weeks or years later, and the
maintenance burden of keeping such a diagram consistent across
edits. For that situation, netfig makes a deliberate trade-off —
the writer pays a cost so the reader does not.

The writer must declare a legend before drawing, choose named
bands, name locations, and declare what each non-solid line
means. The reader, in exchange, does not have to wonder what the
diagram's vocabulary is — every diagram produced through netfig
uses a vocabulary that was declared explicitly and validated
mechanically.

This is one trade-off, not the only possible one. Many other
tools serve different goals and make different trade-offs.

## What this costs the writer

- A legend must exist before nodes and links can be drawn. Every
  shape and line in the diagram has to come from it.
- Line variety is capped at three styles. Non-solid lines are
  only allowed when their meaning is declared.
- Vertical position is determined by named bands; horizontal
  position by named locations. The writer cannot drag a node to
  wherever looks nicest.
- Per-cell node count is capped (currently twelve). Beyond that,
  the writer has to split the band or the location.
- Anything the tool considers a violation is a fatal error. There
  is no permissive mode and no warning tier.

## What this gives the reader

- The same role is always drawn the same way, both within a
  diagram and (when conventions are shared) across diagrams.
- A non-solid line always carries an explicit meaning the reader
  can look up in the legend.
- Vertical position has a single, declared semantic axis.
- Horizontal position has a single, declared semantic axis.
- Nothing in the diagram is there because the tool quietly fell
  back to a default; everything is there because the input passed
  validation.

The intent is that the reader's effort to decode the diagram is
paid down once, inside the legend and the layout policy, rather
than re-done from scratch on every reading.

## What netfig owns

netfig is a mechanical enforcer of constraints, plus a renderer
sized to the constrained input.

The constraints fix every node's row (band) and column (location).
At that point there is little layout problem left to solve —
mostly coordinates to compute. netfig accordingly renders directly
to SVG: the validated layout becomes pixel coordinates by simple
arithmetic; each node becomes a `<rect>` or `<ellipse>` with a
centred label; each link becomes either a straight `<line>`
between the two centres or, when a straight line would cross a
non-endpoint node, an orthogonal `<polyline>` routed through the
gap between bands.

There is no external rendering engine in the pipeline. Converting
SVG to PNG, PDF, or other formats is left to standard SVG tools.

## Reducing hard-to-read, not adding beauty

netfig does not try to make diagrams beautiful. It tries to keep
diagrams from being hard to read in the specific ways the
underlying reference highlights: inconsistent symbols, unexplained
dashed lines, meaningless layouts, missing legends.

A netfig output is not automatically a good diagram. It is a
diagram that has been kept from a known set of bad states. The
remaining distance between "not bad" and "good" is up to the
writer and the domain.

## Where netfig does not fit

In my view, netfig is a poor fit for:

- **Sketching while thinking.** When the diagram exists to help
  the writer work through a design, the friction of declaring a
  legend before drawing is wasted effort.
- **One-off illustrations.** When a diagram will be used once and
  thrown away, the discipline does not pay back.
- **Presentation graphics.** When the goal is to make an
  impression on an audience rather than to convey reusable
  structural information, free-form drawing tools and a human
  designer are more appropriate.

Free-form drawing tools cover those situations well. netfig is
not competing with them; it occupies a different category, aimed
at diagrams that need to outlive the moment of their creation.

## Scope

What is implemented:

- Parse a YAML description of nodes, links, legend, and layout.
- Validate roles, link kinds, and shapes against the legend.
- Validate cell density (a `(band, location)` cell holds at most
  `maxNodesPerCell` nodes).
- Compute pixel coordinates for every node from its (band,
  location) cell.
- Render to SVG: rect/ellipse nodes with centred labels, straight
  lines for unobstructed links, orthogonal polylines for links
  that would otherwise cross a non-endpoint node.
- Fail with a non-zero exit on any input the tool cannot honour
  faithfully.

What is not implemented:

- Crossing minimisation between unrelated diagonal links. Lines
  that share no obstacle but overlap each other visually are not
  rerouted; the writer can adjust by re-ordering bands or
  locations.
- Converting SVG to PNG, PDF, or other formats. Standard SVG
  tools handle this.
- Icon support beyond `rect` and `ellipse`.
- Determining the diagram's purpose. The `purpose` YAML field is
  recorded as metadata only.

## Reference

The constraints netfig implements come from 萩原 学『ネットワーク
図の描き方入門 — 分かりやすさ・見やすさのルールを学ぶ』(Nikkei
BP, 2025). The book is one possible reference for diagram
readability; I adopted it as the constraint set for netfig
without arguing it is the only right one.

Online catalogue entry:
[ネットワーク図の描き方入門](https://bookplus.nikkei.com/atcl/catalog/25/11/21/02321/)
