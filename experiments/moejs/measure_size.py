#!/usr/bin/env python3
"""Build stripped interpreter probes plus real CLI baseline/coexistence."""
import gzip
import json
import os
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parent
os.chdir(root)
output = root / 'bin'
output.mkdir(exist_ok=True)
rows = []
for target in [('darwin','arm64'),('linux','amd64'),('windows','amd64'),('android','arm64')]:
    goos, arch = target
    env = {**os.environ, 'GOOS':goos, 'GOARCH':arch, 'CGO_ENABLED':'0'}
    for engine in ['goja','moejs']:
        path = output / f'{goos}-{arch}-{engine}'
        cmd = ['go','build','-trimpath','-pgo=off','-ldflags=-s -w','-o',str(path)]
        if engine == 'moejs':
            cmd += ['-tags=moejs']
        subprocess.run(cmd+['./cmd/engine-size'], env=env, check=True)
        data=path.read_bytes()
        rows.append(dict(scope='interpreter-only',target=f'{goos}/{arch}',engine=engine,bytes=len(data),gzip_bytes=len(gzip.compress(data,mtime=0))))
        print(rows[-1], flush=True)
# No fake replacement: this is the actual CLI, with and without linking the
# candidate runtime. The overlay does not change the root module or source.
with tempfile.TemporaryDirectory(prefix='gopeed-moejs-size-') as tmp:
    tmp=Path(tmp)
    extra=tmp/'link.go'
    extra.write_text('''package main
import (
 "github.com/MoeForge/moejs"
 "github.com/MoeForge/moejs/eventloop"
 "github.com/MoeForge/moejs/webapi"
)
func init() {
 rt:=moejs.NewRuntime(moejs.Options{MutableIntrinsics:true})
 l,e:=eventloop.New(rt,eventloop.Options{});if e!=nil{panic(e)}
 api,e:=webapi.Install(l,webapi.Options{});if e!=nil{panic(e)}
 s,e:=moejs.CompileScript("probe","1+1");if e!=nil{panic(e)}
 _,e=rt.RunScript(s);if e!=nil{panic(e)}
 api.Close();l.Terminate()
}
''')
    overlay=tmp/'overlay.json'
    overlay.write_text(json.dumps({'Replace':{str(root.parent.parent/'cmd/gopeed/moejs_eval_link.go'):str(extra)}}))
    for goos,arch in [('darwin','arm64'),('linux','amd64'),('windows','amd64')]:
        env={**os.environ,'GOOS':goos,'GOARCH':arch,'CGO_ENABLED':'0'}
        for engine in ['goja-production','goja-plus-moejs-webapi']:
            path=output/f'cli-{goos}-{arch}-{engine}'
            cmd=['go','build','-mod=mod','-trimpath','-pgo=off','-tags=nosqlite','-ldflags=-s -w','-o',str(path)]
            if engine.endswith('webapi'):
                cmd += ['-overlay',str(overlay)]
            subprocess.run(cmd+['github.com/GopeedLab/gopeed/cmd/gopeed'],env=env,check=True)
            data=path.read_bytes()
            rows.append(dict(scope='real-cli',target=f'{goos}/{arch}',engine=engine,bytes=len(data),gzip_bytes=len(gzip.compress(data,mtime=0))))
            print(rows[-1],flush=True)
(root/'results/size.json').write_text(json.dumps(rows,indent=2)+'\n')
# CLI builds may add sums/indirect requirements in this isolated module.
subprocess.run(['go','mod','tidy'],check=True)
