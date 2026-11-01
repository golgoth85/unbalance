Esegui una seconda review indipendente di https://github.com/golgoth85/unbalance, release 2026.11.01.2.

Leggi /workspace/AGENTS.md e l'AGENTS.md del progetto. Usa un tuo branch/worktree. Non installare plugin e non avviare operazioni su dati reali; non modificare l'upstream e non aggirare il diniego plugins:create.

Confronta a40f0cc1d9128b39fcf069c0e1686f8d59106f14 con d2184fbcd9141c2c89e7691a98c53453c0862920. Leggi ROADMAP.md sul main corrente. Repository locale /workspace/unbalance-source; implementazione /workspace/unbalance-review-fixes.

La prima review ha dimostrato scelta errata della destinazione con copie parziali, sottostima con xattr grandi, difetti dell'installer e scroll della pagina Unraid; ha dedotto un rischio di rimozione sorgenti dopo rollback. Non assumere che le nuove correzioni siano sufficienti.

Controlla selezione del delta minimo capiente dopo confronto di tutte le destinazioni, fallback quando manca spazio, copie residue su altri dischi e blocco della cancellazione MOVE Scatter. Verifica stima dell'allocazione per voci mancanti e degli xattr differenti, modifica di soli xattr, errori e scritture concorrenti, Frsize/Bsize e opzioni quiet/stderr.

Controlla lo stato 6 dei fallimenti di verifica/trasferimenti parziali: deve preservare la sorgente e non essere offerto a Remove source dalle UI nuova e legacy dopo rollback. Controlla replay/validate, STOP, filtri, --chmod e copia senza -X. I nuovi GATHER MOVE verificano la sorgente prima di cancellarla, ma la loro pianificazione resta quella precedente. Lo storico vecchio non acquisisce retroattivamente verifiche. Verifica l'ammissione seriale dei comandi, il contesto STOP e lo stato atomico esposto da health.

Controlla installer per start fallito, salute fallita, swap fallito, reinstallazione identica, avvio NAS, retention dei backup/pacchetti e directory .page residue della prima release. Staging e codice precedente devono restare fuori emhttp/plugins. Configurazione e cronologia devono restare intatte. /api/health espone solo stato/versione, non dati protetti. La porta viene letta numericamente senza eseguire la configurazione.

Limite deliberato: una vecchia versione avviata con autenticazione e senza /api/health richiede arresto dal gestore dopo completamento operazioni. /api/auth/status non dimostra l'inattività. COPY ordinaria mantiene il quick-check dimensione/data: conflitti di contenuto richiedono --checksum o Validate. Differenze con filtri/--chmod/senza -X possono preservare le sorgenti e sono documentate nella UI.

Esegui prove indipendenti sotto /temp/unbalance/<tuo-task>, verificando mount e system.LOCATION, mai scritture su disk2. Test disponibili: test Go, tests/installer.py e tests/browser.cjs. Per il browser usa ~/.local/bin/ui-node e UI compilata in preview con API simulate, a 1440×900, 1280×850 e 390×844 touch. Controlla tastiera/focus, tap, console, scroll e assenza di azioni pericolose sui comandi non verificati.

Prove dell'autore: rsync 3.2.7 reale con copia 9/10 preesistente, xattr 60 KB e modifiche solo xattr; test Go con race detector e go vet; 12 casi installer; 27 screenshot browser con API simulate. La pagina impostazioni usa ora il markup reale, rimuovendo PHP e script dipendenti da Unraid; PHP e CSS Unraid reali non sono stati eseguiti. Artefatti in /temp/unbalance/review-fixes. ZFS/btrfs non provati.

Plugin non installato sul NAS: plugins:create negato, runtime atteso ancora 2026.09.11. Backup, rollback, salute e comportamento reali della nuova release sul NAS non verificati.

Restituisci prima problemi riproducibili per gravità, con file:riga, scenario, impatto e correzione. Separa prove, inferenze e limiti; segnala quali rilievi precedenti sono risolti e quali restano. Nessuna review a pagamento.
