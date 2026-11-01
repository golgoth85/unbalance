"""Installer reale in filesystem sintetico, nessun processo o file del NAS."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
ROOT = Path(__file__).resolve().parents[1]
VERSION='2026.11.01.4'

class InstallerTests(unittest.TestCase):
 def run_case(self,busy=False,running=True,state='neutral',healthy=True,start_fails=False,port=7090,legacy_auth=False,boot=False,reinstall=False,legacy_folder=False,seed_backups=False,unrelated=False,port_text=None,invalid_port=False):
  with tempfile.TemporaryDirectory(prefix='installer-',dir='/temp/unbalance/second-review-fixes') as tmp:
   root=Path(tmp);emhttp=root/'emhttp';plugins=emhttp/'plugins';bootdir=root/'boot';code=plugins/'unbalanced';persistent=bootdir/'unbalanced'
   (code/'scripts').mkdir(parents=True);persistent.mkdir(parents=True)
   (persistent/'unbalanced.cfg').write_text(port_text if port_text is not None else f'PORT="{port}"\n')
   (persistent/'unbalanced.env').write_text('CONFIGURAZIONE SINTETICA DA PRESERVARE')
   (code/'VERSION').write_text('previous');(code/'RELEASE_REVISION').write_text('old')
   for name in ('start','stop'):
    p=code/'scripts'/name;p.write_text('#!/bin/bash\nexit 0\n');p.chmod(0o755)
   package=root/'package'/'unbalanced';shutil.copytree(code,package)
   (package/'VERSION').write_text(VERSION);(package/'RELEASE_REVISION').write_text('new')
   (package/'unbalanced').write_text('#!/bin/bash\nexit 0\n');(package/'unbalanced').chmod(0o755)
   if start_fails:(package/'scripts'/'start').write_text('#!/bin/bash\nexit 1\n')
   if legacy_folder:
    legacy=plugins/'unbalanced.previous-2026.11.01.1';shutil.copytree(code,legacy)
    (legacy/'unbalanced').write_text('#!/bin/bash\nexit 0\n');(legacy/'unbalanced').chmod(0o755)
   if seed_backups:
    for v in range(3):
     (persistent/f'backup-code-2026.01.0{v}.tgz').write_bytes(b'old generated backup')
     (persistent/f'unbalanced-2026.01.0{v}.tgz').write_bytes(b'old generated package')
    (persistent/'user-backup.tgz').write_bytes(b'keep')
   if boot:shutil.rmtree(code)
   bundle=persistent/f'unbalanced-{VERSION}.tgz'
   with tarfile.open(bundle,'w:gz') as a:a.add(package,arcname='unbalanced')
   stubs=root/'stubs';stubs.mkdir()
   status='0' if state=='neutral' else '3'
   health='exit 22' if legacy_auth else 'echo \'{"status":'+status+',"version":"'+(VERSION if healthy else 'previous')+'"}\''
   state_response='exit 22' if legacy_auth else 'echo \'{"status":'+status+',"unraid":null}\''
   curl=f"#!/bin/bash\ncase \"$*\" in *127.0.0.1:{port}/*) ;; *) exit 1;; esac\ncase \"$*\" in\n *api/health*) {health};;\n *api/state*) {state_response};;\n *) exit 1;;\nesac\n"
   commands={
    'pgrep':f'#!/bin/bash\n'+('echo 987654; exit 0\n' if busy else 'exit 1\n'),
    'pidof':'#!/bin/bash\n'+('echo 42; exit 0\n' if running else 'exit 1\n'),
    'ps':'#!/bin/bash\necho '+('1' if unrelated else '42')+'\n',
    'sleep':'#!/bin/bash\nexit 0\n',
    'curl':curl}
   for name,text in commands.items():p=stubs/name;p.write_text(text);p.chmod(0o755)
   text=(ROOT/'meta/template/unbalanced.plg').read_text().split('<![CDATA[\nset -eu\n',1)[1].split(']]>',1)[0]
   text=('set -eu\n'+text).replace('/usr/local/emhttp',str(emhttp)).replace('/usr/local/lib/unbalanced-runtime',str(root/'runtime')).replace('/boot/config/plugins',str(bootdir)).replace('##version##',VERSION).replace('##sha256##',hashlib.sha256(bundle.read_bytes()).hexdigest())
   script=root/'install.sh';script.write_text(text)
   env=dict(os.environ,PATH=str(stubs)+':/usr/bin:/bin')
   result=subprocess.run(['bash',str(script)],env=env,capture_output=True,text=True)
   self.assertEqual((persistent/'unbalanced.env').read_text(),'CONFIGURAZIONE SINTETICA DA PRESERVARE')
   self.assertEqual((persistent/'unbalanced.cfg').read_text(),port_text if port_text is not None else f'PORT="{port}"\n')
   self.assertFalse(list((root/'runtime').glob('update.*')))
   self.assertEqual([p.name for p in plugins.iterdir()],['unbalanced'])
   backups=list(persistent.glob('backup-code-*.tgz'))
   if (busy and running and not unrelated) or state!='neutral' or legacy_auth or invalid_port:
    self.assertNotEqual(result.returncode,0,result.stdout);self.assertEqual((code/'VERSION').read_text(),'previous');self.assertEqual(backups,[])
   elif start_fails or (not healthy and running):
    self.assertNotEqual(result.returncode,0,result.stdout);self.assertEqual((code/'VERSION').read_text(),'previous')
   else:
    self.assertEqual(result.returncode,0,result.stdout+result.stderr);self.assertEqual((code/'VERSION').read_text(),VERSION)
    self.assertEqual(len(backups),2 if seed_backups else (0 if boot else 1))
    if seed_backups:
     self.assertEqual((persistent/'user-backup.tgz').read_bytes(),b'keep')
     self.assertEqual(len(list(persistent.glob('unbalanced-*.tgz'))),2)
    if legacy_folder:self.assertEqual(len(list((root/'runtime').glob('legacy.*/previous/VERSION'))),1)
    if reinstall:
     second=subprocess.run(['bash',str(script)],env=env,capture_output=True,text=True)
     self.assertEqual(second.returncode,0,second.stdout+second.stderr)
     self.assertEqual(list(persistent.glob('backup-code-*.tgz')),backups)
   for path in backups:
    if seed_backups and path.name!='backup-code-previous.tgz':continue
    with tarfile.open(path) as a:self.assertFalse(any(p.endswith(('.env','.cfg')) for p in a.getnames()))
 def test_rsync_busy(self):self.run_case(busy=True)
 def test_unbalanced_busy(self):self.run_case(state='busy')
 def test_update_running(self):self.run_case()
 def test_update_stopped(self):self.run_case(running=False)
 def test_failed_health_rolls_back(self):self.run_case(healthy=False)
 def test_start_failure_rolls_back(self):self.run_case(start_fails=True)
 def test_custom_port_authenticated_health(self):self.run_case(port=9876)
 def test_authenticated_legacy_refuses_without_stopping(self):self.run_case(legacy_auth=True)
 def test_boot_does_not_backup(self):self.run_case(running=False,boot=True)
 def test_same_revision_reinstall(self):self.run_case(reinstall=True)
 def test_moves_legacy_page_outside_plugins(self):self.run_case(legacy_folder=True)
 def test_generated_backup_and_package_retention(self):self.run_case(seed_backups=True)
 def test_port_spaces_and_comment(self):self.run_case(port=9876,port_text='  PORT = "9876" # commento sintetico\n')
 def test_unrelated_rsync_does_not_block(self):self.run_case(busy=True,unrelated=True)
 def test_rsync_during_boot_without_service(self):self.run_case(busy=True,running=False,boot=True)
 def test_invalid_port_does_not_fallback(self):self.run_case(port=9876,port_text='PORT="invalid"\n',invalid_port=True)
 def test_absent_port_uses_explicit_default(self):self.run_case(port_text='# PORT assente\n')
if __name__=='__main__':unittest.main()
