"""Prova del wrapper NAS con PHP reale e soli servizi/file sintetici."""
import hashlib
import ctypes
import signal
import time
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
TEST_ROOT = Path(os.environ.get('NAS_INSTALL_TEST_ROOT', '/temp/unbalance/nas-install-script'))
VERSION = '2026.11.01.9'
REVISION = 'cf6321e3b775b73c1bd106c2f41ee90bb015e43f'

class NasInstallTests(unittest.TestCase):
 def run_case(self, mode='ok', running=True, busy=False, related=False, unrelated=False, terminated=None, mixed_live=False):
  related=related or bool(terminated)
  php=os.environ.get('TEST_PHP_BIN') or shutil.which('php')
  if not php:self.skipTest('PHP reale richiesto')
  with tempfile.TemporaryDirectory(prefix='wrapper-',dir=TEST_ROOT) as tmp:
   root=Path(tmp);code=root/'emhttp/plugins/unbalanced';persistent=root/'boot/plugins/unbalanced'
   boot=root/'boot/plugins/unbalanced.plg';reg=root/'log/plugins/unbalanced.plg';stubs=root/'stubs'
   code.mkdir(parents=True);persistent.mkdir(parents=True);reg.parent.mkdir(parents=True);stubs.mkdir()
   (root/'unraid-version').write_text('SINTETICO')
   cfg=b'PORT="9876"\n';reserved=b'CONFIGURAZIONE SINTETICA NON COPIARE'
   (persistent/'unbalanced.cfg').write_bytes(cfg);(persistent/'unbalanced.env').write_bytes(reserved)
   (persistent/'unbalanced.hist').write_bytes(b'STORICO SINTETICO')
   boot.write_text('manifest precedente');reg.symlink_to(boot)
   (code/'VERSION').write_text('2026.09.11\n');(code/'unbalanced').write_text('codice precedente')
   (code/'private.cfg').write_bytes(reserved);(code/'private.env').write_bytes(reserved)
   (code/'scripts').mkdir()
   for action in ['start','stop']:
    body=f'#!/bin/bash\necho {action} >> "{root}/actions"\n'
    body+=(f'touch "{root}/running"\n' if action=='start' else f'unlink "{root}/running"\n')
    script=code/'scripts'/action;script.write_text(body);script.chmod(0o755)
   package=root/'package/unbalanced';shutil.copytree(code,package)
   (package/'VERSION').write_text(VERSION+'\n');(package/'RELEASE_REVISION').write_text(REVISION+'\n')
   (package/'unbalanced').write_text('codice nuovo')
   fixture=root/'fixture';fixture.mkdir();(fixture/'unbalanced.plg').write_text('manifest nuovo verificato')
   bundle=fixture/f'unbalanced-{VERSION}.tgz'
   with tarfile.open(bundle,'w:gz') as archive:archive.add(package,arcname='unbalanced')
   if running:(root/'running').touch()
   def stub(name, body, python=False):
    p=stubs/name;p.write_text(('#!'+sys.executable+'\n' if python else '#!/bin/bash\n')+body);p.chmod(0o755)
   stub('id','echo 0\n')
   stub('sleep','exit 0\n')
   stub('pidof',f'[ -f "{root}/running" ] && {{ echo 42; exit 0; }}\nexit 1\n')
   processes='echo 987654; '+('echo 987655; ' if mixed_live else '')+'exit 0'
   if terminated:processes=f'[ "$(cat "{code}/VERSION")" = "2026.09.11" ] || exit 1; '+processes
   stub('pgrep',processes+'\n' if related or unrelated else 'exit 1\n')
   stub('ps','case "$*" in *stat=*) '+('case "$*" in *987655*) echo S;; *) echo '+terminated+';; esac' if terminated else 'echo S')+';; *) '+('case "$*" in *987655*) echo 1;; *) echo 42;; esac' if related else 'echo 1')+';; esac\n')
   (stubs/'php').symlink_to(php)
   common='import os,sys,json,shutil\nfrom pathlib import Path\nr=Path(os.environ["NAS_FIXTURE_ROOT"])\nmode=os.environ["NAS_FIXTURE_MODE"]\n'
   stub('curl',common+'''
a=sys.argv[1:];url=next(x for x in a if x.startswith('http'))
if '-o' in a:
 name=url.rsplit('/',1)[1];p=Path(a[a.index('-o')+1]);shutil.copyfile(r/'fixture'/name,p)
 with (r/'calls').open('a') as f:f.write('download\\n')
 if mode=='bad_checksum' and name.endswith('.tgz'):p.write_bytes(p.read_bytes()+b'bad')
 if mode=='changed_state' and name.endswith('.tgz'):(r/'running').unlink()
 sys.exit(0)
if not (r/'running').exists():sys.exit(22)
v=(r/'emhttp/plugins/unbalanced/VERSION').read_text().strip()
if mode=='auth' or mode=='unreachable' and v=='2026.11.01.9':sys.exit(22)
if url.endswith('/api/health'):
 if v!='2026.11.01.9':sys.exit(22)
 print(json.dumps({'version':v,'status':3 if mode=='busy_after' else 0}));sys.exit(0)
if url.endswith('/api/state'):
 print(json.dumps({'status':3 if os.environ.get('NAS_FIXTURE_BUSY')=='1' or mode=='busy_after' and v=='2026.11.01.9' else 0}));sys.exit(0)
sys.exit(22)
''',True)
   stub('plugin',common+'''
with (r/'calls').open('a') as f:f.write('plugin\\n')
code=r/'emhttp/plugins/unbalanced';boot=r/'boot/plugins/unbalanced.plg';reg=r/'log/plugins/unbalanced.plg'
if mode in ['fail_before','fail_before_dead']:
 boot.write_text('manifest fallito')
 if mode=='fail_before_dead':(r/'running').unlink()
 sys.exit(1)
shutil.rmtree(code);shutil.copytree(r/'package/unbalanced',code)
boot.write_bytes(Path(sys.argv[2]).read_bytes());reg.unlink();reg.symlink_to(boot)
if mode=='fail_after':sys.exit(1)
if mode=='wrong_revision':(code/'RELEASE_REVISION').write_text('errata')
if mode=='wrong_manifest':boot.write_text('manifest inatteso')
if mode=='dead_service' and (r/'running').exists():(r/'running').unlink()
''',True)
   text=(ROOT/'tools/install-nas-2026.11.01.9.sh').read_text()
   for original,replacement in [('/usr/local/emhttp',str(root/'emhttp')),('/boot/config/plugins',str(root/'boot/plugins')),('/var/log/plugins',str(root/'log/plugins')),('/usr/local/lib/unbalanced-runtime',str(root/'runtime')),('/usr/local/sbin/plugin',str(stubs/'plugin')),('/etc/unraid-version',str(root/'unraid-version')),('/var/lock/unbalanced-manual-install.lock',str(root/'install.lock'))]:text=text.replace(original,replacement)
   text=text.replace('a98ac1e77bdaa62adc8de91730226d621b1b34b6143fba68707e67a7049d1d38',hashlib.sha256((fixture/'unbalanced.plg').read_bytes()).hexdigest()).replace('50e5e03d682f13317e05b8e0215143055437c748216d33123df17a77ace80a5e',hashlib.sha256(bundle.read_bytes()).hexdigest())
   script=root/'install.sh';script.write_text(text)
   env=dict(os.environ,PATH=str(stubs)+':/usr/bin:/bin',NAS_FIXTURE_ROOT=str(root),NAS_FIXTURE_MODE=mode,NAS_FIXTURE_BUSY='1' if busy else '0')
   result=subprocess.run(['bash',str(script)],env=env,capture_output=True,text=True)
   output=result.stdout+result.stderr
   self.assertEqual((persistent/'unbalanced.cfg').read_bytes(),cfg)
   self.assertEqual((persistent/'unbalanced.env').read_bytes(),reserved)
   self.assertEqual((persistent/'unbalanced.hist').read_bytes(),b'STORICO SINTETICO')
   self.assertNotIn(reserved.decode(),output)
   calls=(root/'calls').read_text().splitlines() if (root/'calls').exists() else []
   actions=(root/'actions').read_text().splitlines() if (root/'actions').exists() else []
   backups=list(persistent.glob('manual-install-*'))
   if busy or (related and not terminated) or mixed_live or mode in ['auth','bad_checksum','changed_state']:
    self.assertNotEqual(result.returncode,0,output);self.assertNotIn('plugin',calls)
    self.assertEqual((code/'VERSION').read_text().strip(),'2026.09.11');self.assertEqual(actions,[])
    self.assertEqual(backups,[])
   elif mode in ['unreachable','busy_after']:
    self.assertNotEqual(result.returncode,0,output)
    self.assertEqual((code/'VERSION').read_text().strip(),VERSION)
    self.assertEqual(actions,[],'non interrompere un servizio non verificabile/occupato')
    self.assertIn('Rollback automatico sospeso',output)
   elif mode!='ok':
    self.assertNotEqual(result.returncode,0,output)
    self.assertEqual((code/'VERSION').read_text().strip(),'2026.09.11')
    self.assertEqual((code/'unbalanced').read_text(),'codice precedente')
    self.assertEqual(boot.read_text(),'manifest precedente');self.assertEqual(os.readlink(reg),str(boot))
    self.assertEqual((root/'running').exists(),running)
    if mode=='fail_before':self.assertEqual(actions,[],'evitare un secondo rollback del codice già ripristinato')
   else:
    self.assertEqual(result.returncode,0,output)
    self.assertEqual((code/'VERSION').read_text().strip(),VERSION)
    self.assertEqual((code/'RELEASE_REVISION').read_text().strip(),REVISION)
    self.assertEqual(boot.read_bytes(),(fixture/'unbalanced.plg').read_bytes())
    self.assertEqual((root/'running').exists(),running)
    self.assertEqual(actions,['stop','start'] if terminated else [])
    # Riesecuzione idempotente e lock rilasciato.
    second=subprocess.run(['bash',str(script)],env=env,capture_output=True,text=True,timeout=5)
    self.assertEqual(second.returncode,0,second.stdout+second.stderr)
    self.assertIn('già installata',second.stdout)
    self.assertEqual(list(persistent.glob('manual-install-*')),backups)
   for backup in backups:
    with tarfile.open(backup/'code.tgz') as archive:
     self.assertFalse(any(n.endswith(('.cfg','.env')) for n in archive.getnames()))

 def test_update_running(self):self.run_case()
 def test_update_stopped_preserves_state(self):self.run_case(running=False)
 def test_busy_refuses_without_effects(self):self.run_case(busy=True)
 def test_related_rsync_refuses_without_effects(self):self.run_case(related=True)
 def test_unrelated_rsync_is_allowed(self):self.run_case(unrelated=True)
 def test_zombie_rsync_is_not_an_active_transfer(self):self.run_case(terminated='Z')
 def test_dead_rsync_is_not_an_active_transfer(self):self.run_case(terminated='X')
 def test_zombie_does_not_override_busy_api(self):self.run_case(terminated='Z',busy=True)
 def test_zombie_and_live_unattributed_rsync_still_block(self):self.run_case(terminated='Z',mixed_live=True)
 def test_legacy_auth_refuses_without_stopping(self):self.run_case(mode='auth')
 def test_bad_checksum_refuses_before_backup(self):self.run_case(mode='bad_checksum')
 def test_service_state_changed_refuses(self):self.run_case(mode='changed_state')
 def test_failure_before_swap_keeps_code(self):self.run_case(mode='fail_before')
 def test_failure_after_swap_rolls_back(self):self.run_case(mode='fail_after')
 def test_stopped_failure_preserves_stopped_state(self):self.run_case(mode='fail_after',running=False)
 def test_old_code_restored_but_service_dead_is_restarted(self):self.run_case(mode='fail_before_dead')
 def test_wrong_revision_rolls_back(self):self.run_case(mode='wrong_revision')
 def test_wrong_manifest_rolls_back(self):self.run_case(mode='wrong_manifest')
 def test_dead_service_rolls_back(self):self.run_case(mode='dead_service')
 def test_unreachable_service_does_not_get_killed(self):self.run_case(mode='unreachable')
 def test_new_busy_operation_does_not_get_killed(self):self.run_case(mode='busy_after')

class RealProcessDetectionTests(unittest.TestCase):
 def test_real_zombie_is_ignored_but_sleeping_process_blocks(self):
  text=(ROOT/'tools/install-nas-2026.11.01.9.sh').read_text()
  function=text[text.index('assert_idle() {'):text.index('assert_unchanged() {')]
  for terminated in [False,True]:
   with self.subTest(terminated=terminated):
    pid=os.fork()
    if pid==0:
     libc=ctypes.CDLL(None)
     libc.prctl.argtypes=[ctypes.c_int,ctypes.c_char_p,ctypes.c_ulong,ctypes.c_ulong,ctypes.c_ulong]
     libc.prctl(15,b'rsync',0,0,0)
     if terminated:os._exit(0)
     while True:signal.pause()
    try:
     expected='Z' if terminated else 'S'
     for _ in range(200):
      stat=Path(f'/proc/{pid}/stat').read_text().split()[2]
      if stat==expected and Path(f'/proc/{pid}/comm').read_text().strip()=='rsync':break
      time.sleep(0.01)
     else:self.fail('stato processo di prova non raggiunto')
     script='set -euo pipefail\n'+function+f'\npidof() {{ echo {os.getpid()}; }}\npgrep() {{ echo {pid}; }}\nread_status() {{ echo 0; }}\nassert_idle\n[ "$legacy_cleanup" -eq 1 ]\n'
     result=subprocess.run(['bash'],input=script,text=True,capture_output=True)
     if terminated:self.assertEqual(result.returncode,0,result.stderr)
     else:
      self.assertNotEqual(result.returncode,0)
      self.assertIn('Rsync di Unbalanced vivo',result.stderr)
    finally:
     if not terminated:os.kill(pid,signal.SIGKILL)
     os.waitpid(pid,0)

if __name__=='__main__':unittest.main()
