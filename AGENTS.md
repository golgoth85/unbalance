# Unbalanced — fork NASdelPala

Fork personale di `jbrodriguez/unbalance`. Branch applicativo: `main`; sviluppo in branch `agent/*` e worktree dedicato. Non modificare né pubblicare sull'upstream. GitHub Actions disabilitato; verifiche locali.

## Funzionamento e comandi

Plugin Unraid in Go con interfaccia React/Tailwind incorporata nel binario. Prima generare la UI: in `ui`, `npm ci --include=dev` e `npm run build`; poi dalla radice `go test ./...` e `go vet ./...`. Il frontend deve esistere prima dei controlli Go perché viene incorporato. Go richiesto da `go.mod`. Per il browser usare `~/.local/bin/ui-node`, mai servizi o dati reali per le prove.

Fixture, toolchain, cache Go/npm, binari di prova e screenshot in `/temp/unbalance/<task>` dopo verifica mount e `system.LOCATION`, mai su disk2. Dati sintetici; pulizia al termine. Le dipendenze del frontend restano nel normale `node_modules` ignorato da Git.

## Runtime e distribuzione

NAS `192.168.1.55`, UI sulla porta configurata (7090 predefinita); endpoint pubblico minimo `/api/health` con stato/versione. Plugin installato in `/usr/local/emhttp/plugins/unbalanced`; pacchetti e configurazione persistente in `/boot/config/plugins/unbalanced`. Non leggere o copiare configurazioni riservate. Il plugin non è un container: aggiornamento tramite pacchetto `.tgz` e manifest `.plg` del fork. Prima di installare verificare assenza di operazioni Unbalanced e rsync concorrenti pertinenti; conservare binario e template precedenti per rollback. Non interrompere trasferimenti reali.

## Zone delicate

Pianificazione spazio, argomenti rsync, ticket di piano, convalida percorsi e cancellazione sorgenti MOVE/GATHER. Stima fallita: nessun piano parziale utilizzabile. Stima nulla: cartella comunque inclusa, anche per MOVE; prima di cancellare una sorgente di nuovi piani Scatter/Gather verificare integralmente la destinazione con checksum senza filtri che possano nascondere file. Test su sorgente e destinazione sintetiche, inclusi errori, cancellazione pianificazione, duplicati, conflitti, più destinazioni e riserva spazio.

Interfaccia e pagina impostazioni Unraid: browser reale alle tre dimensioni prescritte da `/workspace/AGENTS.md`, touch/mobile, focus e tastiera, console e scroll. Il prompt di review indipendente deve chiarire commit, prove effettive e limiti di deploy.

Per aggiornare una versione legacy con autenticazione attiva e senza `/api/health`, concludere le operazioni e arrestare il plugin dal suo gestore: il solo `/api/auth/status` non prova l'inattività. Non disattivare l'autenticazione né leggere password/hash per aggirare il controllo. L'installer legge solo il campo PORT, non esegue la configurazione.

GATHER: controllare i conflitti fra tutte le sorgenti e con la destinazione prima di qualsiasi copia/rimozione, e ricontrollare la destinazione prima di ogni comando. I confronti devono preservare esattamente gli argomenti dei filtri. Inizializzare STOP/contesto durante l'ammissione sincrona, prima del worker. Staging installer in `/usr/local/lib/unbalanced-runtime`, fuori dalla radice web. Il backend legacy dopo rollback non acquisisce protezioni nuove: non usare API di rimozione sui comandi non verificati.

Xattr GATHER: proteggere file, cartelle e genitori impliciti trasferiti da -R, sia contro altre sorgenti sia contro la destinazione. Nessuna esclusione automatica per attributi volatili. Mtime/permessi/proprietario dei duplicati identici e delle cartelle padre implicite già presenti seguono l'ultima sorgente rsync, anche se i genitori contengono altri dati: documentare l'effetto sull'accesso. STOP nel preflight va registrato come interrotto, non come verifica fallita.

PORT: assente usa 7090 anche in scripts/start; valori invalidi vanno fermati prima del killall. Installer rifiuta CR nella riga PORT e accetta export PORT=numero; non esegue la configurazione.

La politica GATHER deve essere visibile per default prima del MOVE. Nel parser rsync, --no-OPTION è uno switch; se un token è il valore di un'altra opzione, preservarlo prima di interpretare eventuali switch.
Installer PORT: formato Unraid, non Bash arbitrario; rifiutare riferimenti testuali a PORT fuori dalle assegnazioni ammesse invece di indovinare la porta, ignorando commenti e letterali innocui di altre variabili. Il controllo è statico, non interpreta Bash arbitrario od offuscato. Mantenere i messaggi di rifiuto prima di qualunque effetto.

Argomenti rsync: vietare sia --remove-source-files sia l’alias storico --remove-sent-files, comprese le forme con =. Il rifiuto deve valere per impostazioni, pianificazione e worker di COPY/MOVE/GATHER/Validate/replay, prima di qualunque trasferimento.

Installazione manuale della .9: `tools/install-nas-2026.11.01.9.sh` va eseguito come root sul NAS, non nel container. Wrapper fissato a release/revisione/checksum; usa il gestore plugin ufficiale. Backup manuali in `/boot/config/plugins/unbalanced/manual-install-*`, esclusi configurazioni/segreti. Preserva lo stato iniziale del servizio e non interrompe attività per forzare un rollback. Verifica isolata con PHP reale: `python3 tests/nas_install.py`, `NAS_INSTALL_TEST_ROOT` sotto /temp/unbalance dopo i controlli prescritti; `TEST_PHP_BIN` se PHP non è in PATH.
