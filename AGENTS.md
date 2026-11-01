# Unbalanced — fork NASdelPala

Fork personale di `jbrodriguez/unbalance`. Branch applicativo: `main`; sviluppo in branch `agent/*` e worktree dedicato. Non modificare né pubblicare sull'upstream. GitHub Actions disabilitato; verifiche locali.

## Funzionamento e comandi

Plugin Unraid in Go con interfaccia React/Tailwind incorporata nel binario. `go test ./...`, `go vet ./...`; in `ui`: `npm ci`, `npm run build`. Go richiesto da `go.mod`. Per il browser usare `~/.local/bin/ui-node`, mai servizi o dati reali per le prove.

Fixture, toolchain, cache Go/npm, binari di prova e screenshot in `/temp/unbalance/<task>` dopo verifica mount e `system.LOCATION`, mai su disk2. Dati sintetici; pulizia al termine. Le dipendenze del frontend restano nel normale `node_modules` ignorato da Git.

## Runtime e distribuzione

NAS `192.168.1.55`, UI porta 7090. Plugin installato in `/usr/local/emhttp/plugins/unbalanced`; pacchetti e configurazione persistente in `/boot/config/plugins/unbalanced`. Non leggere o copiare configurazioni riservate. Il plugin non è un container: aggiornamento tramite pacchetto `.tgz` e manifest `.plg` del fork. Prima di installare verificare assenza di operazioni Unbalanced e rsync concorrenti pertinenti; conservare binario e template precedenti per rollback. Non interrompere trasferimenti reali.

## Zone delicate

Pianificazione spazio, argomenti rsync, ticket di piano, convalida percorsi e cancellazione sorgenti MOVE/GATHER. Stima fallita: nessun piano parziale utilizzabile. Stima nulla: cartella comunque inclusa, anche per MOVE; prima di cancellare una sorgente pianificata con deduplicazione verificare integralmente la destinazione con checksum senza filtri che possano nascondere file. Test su sorgente e destinazione sintetiche, inclusi errori, cancellazione pianificazione, duplicati, conflitti, più destinazioni e riserva spazio.

Interfaccia e pagina impostazioni Unraid: browser reale alle tre dimensioni prescritte da `/workspace/AGENTS.md`, touch/mobile, focus e tastiera, console e scroll. Il prompt di review indipendente deve chiarire commit, prove effettive e limiti di deploy.
