#!/usr/bin/env python3
"""Measure equal extension work in fresh OS processes, never CPU percentage."""
import argparse
import json
import os
from pathlib import Path
import statistics
import subprocess
import tempfile

root = Path(__file__).resolve().parent
os.chdir(root)
parser = argparse.ArgumentParser()
parser.add_argument('--repeats', type=int, default=5)
options = parser.add_mutually_exclusive_group()
options.add_argument('--smoke', action='store_true')
options.add_argument('--memory-only', action='store_true', help='replace retained samples, keeping saved CPU samples')
options.add_argument('--summarize-only', action='store_true')
args = parser.parse_args()
if args.repeats < 1:
    parser.error('--repeats must be positive')
all_workloads = [('cold',20),('warm',200),('retained',1),('retained',8)]
workloads = [('cold',2),('warm',2),('retained',2)] if args.smoke else all_workloads
filename = 'resources-smoke.json' if args.smoke else 'resources.json'
rows = []
if args.summarize_only:
    rows = json.loads((root/'results'/filename).read_text())
else:
    (root/'bin').mkdir(exist_ok=True)
    binary = root/'bin/resource-worker.test'
    subprocess.run(['go','test','-pgo=off','-c','-o',str(binary)],check=True)
    if args.memory_only:
        rows = [r for r in json.loads((root/'results'/filename).read_text()) if r['mode'] != 'retained']
        workloads = [('retained',1),('retained',8)]
    with tempfile.TemporaryDirectory(prefix='gopeed-resource-') as temp:
        for extension in ['bilibili','youtube','twitter']:
            for mode,count in workloads:
                for repeat in range(args.repeats):
                    engines = ['goja','moejs'] if repeat % 2 == 0 else ['moejs','goja']
                    for engine in engines:
                        output = Path(temp)/'result.json'
                        env = {**os.environ,'GOMAXPROCS':'1','GOGC':'100','GOMEMLIMIT':'off',
                               'GOPEED_RESOURCE_WORKER':'1','RESOURCE_ENGINE':engine,
                               'RESOURCE_EXTENSION':extension,'RESOURCE_MODE':mode,
                               'RESOURCE_COUNT':str(count),'RESOURCE_OUTPUT':str(output)}
                        run = subprocess.run([str(binary),'-test.run=^TestResourceWorker$','-test.timeout=90s'],
                                             env=env,text=True,capture_output=True,timeout=100)
                        if run.returncode:
                            raise RuntimeError(run.stdout+run.stderr)
                        row = json.loads(output.read_text())
                        row['repeat'] = repeat
                        rows.append(row)
                        hashes = {r['output_sha256'] for r in rows if r['extension']==extension}
                        if len(hashes)!=1:
                            raise RuntimeError(f'Output mismatch for {extension}: {hashes}')
                        cpu = (row['user_cpu_ns']+row['system_cpu_ns'])/count/1e6
                        print(f'{extension}/{mode}/{count}/{engine} repeat={repeat+1}: '
                              f'CPU={cpu:.3f} ms/op, peakRSS={row["after_work"]["process_peak_rss_bytes"]/2**20:.1f} MiB',flush=True)
                        (root/'results'/filename).write_text(json.dumps(rows,indent=2)+'\n')

summary = []
for extension in ['bilibili','youtube','twitter']:
    for mode,count in (all_workloads if args.memory_only else workloads):
        for engine in ['goja','moejs']:
            group = [r for r in rows if (r['extension'],r['mode'],r['count'],r['engine']) == (extension,mode,count,engine)]
            if not group:
                raise RuntimeError(f'Missing samples for {extension}/{mode}/{count}/{engine}')
            def med(fn): return statistics.median(fn(r) for r in group)
            cpu = [(r['user_cpu_ns']+r['system_cpu_ns'])/count for r in group]
            summary.append(dict(extension=extension,mode=mode,count=count,engine=engine,samples=len(group),
                                cpu_ns_per_op=statistics.median(cpu),cpu_ns_min=min(cpu),cpu_ns_max=max(cpu),
                                user_cpu_ns_per_op=med(lambda r:r['user_cpu_ns']/count),
                                system_cpu_ns_per_op=med(lambda r:r['system_cpu_ns']/count),
                                wall_ns_per_op=med(lambda r:r['wall_ns']/count),
                                allocated_bytes_per_op=med(lambda r:r['allocated_bytes']/count),
                                gc_cycles=med(lambda r:r['gc_cycles']),
                                peak_rss_bytes=med(lambda r:r['after_work']['process_peak_rss_bytes']),
                                peak_rss_min_bytes=min(r['after_work']['process_peak_rss_bytes'] for r in group),
                                peak_rss_max_bytes=max(r['after_work']['process_peak_rss_bytes'] for r in group),
                                rss_after_work_bytes=med(lambda r:r['after_work']['rss_bytes']),
                                live_heap_bytes=med(lambda r:r['after_gc']['heap_alloc_bytes']),
                                live_heap_delta_bytes=med(lambda r:r['after_gc']['heap_alloc_bytes']-r['baseline']['heap_alloc_bytes']),
                                retained_rss_bytes=med(lambda r:r['after_os_release']['rss_bytes']),
                                retained_rss_delta_bytes=med(lambda r:r['after_os_release']['rss_bytes']-r['baseline']['rss_bytes']),
                                after_close_heap_delta_bytes=med(lambda r:r['after_close_os_release']['heap_alloc_bytes']-r['baseline']['heap_alloc_bytes']),
                                after_close_rss_bytes=med(lambda r:r['after_close_os_release']['rss_bytes'])))
name = 'resources-smoke-summary.json' if args.smoke else 'resources-summary.json'
(root/'results'/name).write_text(json.dumps(summary,indent=2)+'\n')
