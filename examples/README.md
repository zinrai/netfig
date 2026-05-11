# examples

Practical network diagrams drawn with netfig. Each directory holds a
`topology.yaml` plus the rendered `diagram.svg`. The YAML's `purpose`
field carries the audience and intent for that specific diagram —
read it together with the SVG to see what the example is trying to
teach.

The diagrams are intentionally close to setups encountered in real
operations work, not minimal feature demos.

## Which example should I read first?

| If you want to... | Start with |
| --- | --- |
| See the smallest YAML that produces a useful diagram | [`ha-firewall-pair`](ha-firewall-pair/) |
| See a canonical RFC 7938 Clos data centre fabric | [`datacenter-fabric`](datacenter-fabric/) |
| See line styles carry operational meaning (eBGP solid, iBGP dashed) | [`isp-backbone`](isp-backbone/) |
| See netfig's per-cell upper limit in a real setting | [`dense-cell`](dense-cell/) |
| See a deep multi-band stack, end-to-end | [`multi-tier-isp`](multi-tier-isp/) |

Render any of them with the same command:

```
netfig topology.yaml > diagram.svg
```

For the field-by-field YAML reference and the validation error
catalogue, see [`docs/USAGE.md`](../docs/USAGE.md).
