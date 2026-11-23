#!/usr/bin/env python3
"""Fetch the unmodified published extension bundles at immutable commits."""
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

sources = {
    'bilibili': ('monkeyWie/gopeed-extension-bilibili', 'b91db1ea52e975d8de9823f01a7af5f9ea83792f'),
    'youtube': ('monkeyWie/gopeed-extension-youtube', 'eeb78d168e053e9a2aad1dbaa309fc11f6992f7c'),
    'twitter': ('monkeyWie/gopeed-extension-twitter', 'a963428bbfdbee9510bf224b00b384e37514ce77'),
}
root = Path(__file__).resolve().parent / 'testdata' / 'extensions'
root.mkdir(parents=True, exist_ok=True)
expected = json.loads((root.parent.parent / 'results/sources.json').read_text())
manifest = {}
for name, (repo, commit) in sources.items():
    url = f'https://raw.githubusercontent.com/{repo}/{commit}/dist/index.js'
    with tempfile.TemporaryDirectory(prefix='gopeed-extension-') as checkout:
        subprocess.run(['git', 'init', '-q', checkout], check=True)
        subprocess.run(['git', '-C', checkout, 'fetch', '-q', '--depth=1', f'https://github.com/{repo}.git', commit], check=True)
        data = subprocess.check_output(['git', '-C', checkout, 'show', 'FETCH_HEAD:dist/index.js'])
    digest = hashlib.sha256(data).hexdigest()
    if digest != expected[name]['sha256']:
        raise RuntimeError(f'Unexpected bundle hash for {name}: {digest}')
    (root / f'{name}.js').write_bytes(data)
    manifest[name] = dict(repo=repo, commit=commit, url=url, bytes=len(data), sha256=hashlib.sha256(data).hexdigest())
    print(name, len(data), manifest[name]['sha256'])
(root / 'sources.json').write_text(json.dumps(manifest, indent=2) + '\n')
