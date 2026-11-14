"""Exercise project creation, editing and export through real MCP stdio."""
import argparse
import json
from pathlib import Path
import queue
import subprocess
import tempfile
import threading
import zipfile

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--binary', type=Path, default=root / 'build/bin/EZ软图-build.exe')
parser.add_argument('--skip-png', action='store_true')
args = parser.parse_args()
if not args.binary.is_file():
    raise SystemExit('Build EZ软图 first: ./scripts/build.ps1')
base = root / 'test-results/mcp'
base.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='stdio-', dir=base) as temporary:
    out = Path(temporary).resolve()
    assert out.parent == base.resolve()
    p = subprocess.Popen([str(args.binary.resolve()), '--mcp', '--workspace', str(out)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8', creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
    stdin, stdout = p.stdin, p.stdout
    assert stdin is not None and stdout is not None, 'MCP stdio pipes were not created'
    lines = queue.Queue()
    def reader():
        assert stdout is not None
        for line in stdout:
            lines.put(line)
    threading.Thread(target=reader, daemon=True).start()
    seq = 0
    def request(method, params):
        global seq
        assert stdin is not None
        seq += 1
        stdin.write(json.dumps({'jsonrpc':'2.0','id':seq,'method':method,'params':params}, ensure_ascii=False) + '\n')
        stdin.flush()
        while True:
            obj = json.loads(lines.get(timeout=90))
            if obj.get('id') == seq:
                assert 'error' not in obj, obj
                return obj['result']
    def call(name, params, fail=False):
        result = request('tools/call', {'name':name,'arguments':params})
        assert bool(result.get('isError')) == fail, result
        return result.get('structuredContent')
    try:
        init = request('initialize', {'protocolVersion':'2025-11-25','capabilities':{},'clientInfo':{'name':'ezruantu-project-smoke','version':'1'}})
        assert init['serverInfo']['name'] == 'ezruantu'
        assert 'export_project' in init.get('instructions', '')
        stdin.write(json.dumps({'jsonrpc':'2.0','method':'notifications/initialized'}) + '\n'); stdin.flush()
        names = {tool['name'] for tool in request('tools/list', {})['tools']}
        assert len(names) == 13
        assert {'inspect_project','create_folder','read_diagram','save_diagram','patch_diagram','export_project'} <= names
        assert call('inspect_project', {})['count'] == 0
        call('create_folder', {'name':'项目演示'})
        call('create_folder', {'parent':'项目演示','name':'数据库'})
        diagrams = json.loads((root / 'shared/library.json').read_text(encoding='utf-8'))
        for diagram in diagrams:
            folder = '项目演示/数据库' if diagram['type'] in ('er', 'table') else '项目演示'
            call('save_diagram', {'path':folder + '/' + diagram['type'] + '.json','diagram':diagram})
        assert call('inspect_project', {'path':'项目演示'})['count'] == 8
        doc = call('read_diagram', {'path':'项目演示/usecase.json'})
        edited = call('patch_diagram', {'path':doc['path'],'revision':doc['revision'],'operations':[
            {'op':'test','path':'/model/actors/0/id','value':'reader'},
            {'op':'replace','path':'/model/actors/0/name','value':'注册读者'},
            {'op':'add','path':'/model/actors/-','value':{'id':'guest','name':'访客'}}
        ]})
        assert edited['revision'] != doc['revision'] and Path(edited['backup_path']).is_file()
        call('patch_diagram', {'path':doc['path'],'revision':doc['revision'],'operations':[{'op':'replace','path':'/title','value':'旧版本覆盖'}]}, fail=True)
        assert call('read_diagram', {'path':doc['path']})['diagram']['model']['actors'][0]['name'] == '注册读者'
        exported = call('export_project', {'path':'项目演示','format':'json','outputs':['folder','zip']})
        assert exported['count'] == 8 and len(exported['paths']) == 2
        with zipfile.ZipFile(next(p for p in exported['paths'] if p.endswith('.zip'))) as archive:
            assert len(archive.namelist()) == 8 and '数据库/table.json' in archive.namelist()
            assert json.loads(archive.read('usecase.json'))['model']['actors'][0]['name'] == '注册读者'
        if not args.skip_png:
            exported = call('export_project', {'path':'项目演示','format':'png','outputs':['folder','zip']})
            assert exported['count'] == 8 and len(exported['paths']) == 2
            with zipfile.ZipFile(next(p for p in exported['paths'] if p.endswith('.zip'))) as archive:
                assert len(archive.namelist()) == 8
                for name in archive.namelist():
                    assert name.endswith('.png') and archive.read(name).startswith(bytes.fromhex('89504e470d0a1a0a'))
                proof = root / 'test-results/mcp-proof'
                proof.mkdir(parents=True, exist_ok=True)
                (proof / 'usecase.png').write_bytes(archive.read('usecase.png'))
                (proof / 'sequence.png').write_bytes(archive.read('sequence.png'))
        assert call('inspect_project', {})['count'] == 8
        print('PASS: real stdio handshake, 13 tools, eight-diagram project, atomic edits, conflict protection, hierarchical JSON exports' + (', eight PNGs + ZIP' if not args.skip_png else ''))
    finally:
        stdin.close()
        try:
            p.wait(timeout=5)
        except subprocess.TimeoutExpired:
            p.terminate(); p.wait(timeout=5)
