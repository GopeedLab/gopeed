#!/usr/bin/env python3
"""Summarize saved Go benchmark samples; times are medians, not best runs."""
import json
from pathlib import Path
import re
import statistics

root = Path(__file__).resolve().parent
samples = {}
for line in (root/'results/bench.txt').read_text().splitlines():
    match = re.match(r'(Benchmark\S+)\s+\d+\s+([\d.]+) ns/op\s+([\d.]+) B/op\s+([\d.]+) allocs/op',line)
    if match:
        key, *values = match.groups()
        samples.setdefault(key, []).append(list(map(float,values)))
summary = {}
for key, rows in samples.items():
    summary[key] = dict(samples=len(rows),ns_median=statistics.median(r[0] for r in rows),ns_min=min(r[0] for r in rows),ns_max=max(r[0] for r in rows),bytes_median=statistics.median(r[1] for r in rows),allocs_median=statistics.median(r[2] for r in rows))
(root/'results/bench-summary.json').write_text(json.dumps(summary,indent=2)+'\n')
for key, row in summary.items():
    print(f'{key}: {row["ns_median"]/1e6:.3f} ms/op; {row["bytes_median"]/1024:.1f} KiB/op; samples={row["samples"]}')
