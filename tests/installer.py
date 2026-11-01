"""Esegue solo il blocco di installazione in un filesystem sintetico isolato."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]

class InstallerTests(unittest.TestCase):
    def run_case(self, busy=False, running=True, state='neutral', healthy=True):
        with tempfile.TemporaryDirectory(prefix='installer-', dir='/temp/unbalance/resume-mobile') as tmp:
            root = Path(tmp)
            plugins = root / 'plugins'
            boot = root / 'boot'
            code = plugins / 'unbalanced'
            persistent = boot / 'unbalanced'
            (code / 'scripts').mkdir(parents=True)
            persistent.mkdir(parents=True)
            (boot / 'unbalanced.plg').write_text('manifest sintetico')
            for name in ['cfg', 'env']:
                (persistent / f'unbalanced.{name}').write_text('CONFIGURAZIONE SINTETICA DA PRESERVARE')
            (code / 'VERSION').write_text('previous')
            for name in ['start','stop']:
                p = code / 'scripts' / name
                p.write_text('#!/bin/bash\nexit 0\n'); p.chmod(0o755)
            package = root / 'package' / 'unbalanced'
            shutil.copytree(code, package)
            (package / 'VERSION').write_text('2026.11.01.1')
            (package / 'unbalanced').write_text('#!/bin/bash\nexit 0\n')
            (package / 'unbalanced').chmod(0o755)
            bundle = persistent / 'unbalanced-2026.11.01.1.tgz'
            with tarfile.open(bundle,'w:gz') as archive:
                archive.add(package, arcname='unbalanced')
            stubs = root / 'stubs'; stubs.mkdir()
            commands = {
                'pgrep': f'#!/bin/bash\nexit {0 if busy else 1}\n',
                'pidof': f'#!/bin/bash\nexit {0 if running else 1}\n',
                'sleep': '#!/bin/bash\nexit 0\n',
                'curl': '#!/bin/bash\ncase "$*" in\n *api/state*) echo \'{"status":' + ('0' if state=='neutral' else '3') + ',"unraid":null}\';;\n *) echo \'{"version":"' + ('2026.11.01.1' if healthy else 'previous') + '"}\';;\nesac\n',
            }
            for name, text in commands.items():
                p = stubs / name; p.write_text(text); p.chmod(0o755)
            text = (ROOT / 'meta/template/unbalanced.plg').read_text()
            text = text.split('<![CDATA[\nset -eu\n',1)[1].split(']]>',1)[0]
            text = 'set -eu\n' + text
            text = text.replace('/usr/local/emhttp/plugins',str(plugins)).replace('/boot/config/plugins',str(boot))
            text = text.replace('##version##','2026.11.01.1').replace('##sha256##',hashlib.sha256(bundle.read_bytes()).hexdigest())
            script=root/'install.sh'; script.write_text(text)
            env=dict(os.environ,PATH=str(stubs)+':/usr/bin:/bin')
            result=subprocess.run(['bash',str(script)],env=env,capture_output=True,text=True)
            for name in ['cfg','env']:
                self.assertEqual((persistent/f'unbalanced.{name}').read_text(),'CONFIGURAZIONE SINTETICA DA PRESERVARE')
            if busy or state!='neutral':
                self.assertNotEqual(result.returncode,0,result.stdout)
                self.assertEqual((code/'VERSION').read_text(),'previous')
                self.assertFalse(list(persistent.glob('rollback-*')))
            elif not healthy and running:
                self.assertNotEqual(result.returncode,0,result.stdout)
                self.assertEqual((code/'VERSION').read_text(),'previous')
            else:
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                self.assertEqual((code/'VERSION').read_text(),'2026.11.01.1')
                backups=list(persistent.glob('rollback-*/code.tgz'))
                self.assertEqual(len(backups),1)
                with tarfile.open(backups[0]) as archive:
                    self.assertFalse(any(p.endswith(('.env','.cfg')) for p in archive.getnames()))
    def test_rsync_busy(self): self.run_case(busy=True)
    def test_unbalanced_busy(self): self.run_case(state='busy')
    def test_update_running(self): self.run_case()
    def test_update_stopped(self): self.run_case(running=False)
    def test_failed_health_rolls_back(self): self.run_case(healthy=False)

if __name__=='__main__': unittest.main()
