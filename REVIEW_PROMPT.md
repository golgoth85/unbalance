Esegui una terza review indipendente di https://github.com/golgoth85/unbalance, release 2026.11.01.3.

Leggi /workspace/AGENTS.md, l'AGENTS.md del progetto e ROADMAP.md sul main corrente. Usa un tuo worktree; non modificare upstream o altri task. Nessuna installazione, operazione su dati reali o aggiramento di plugins:create. Non avviare review a pagamento.

Confronta d2184fbcd9141c2c89e7691a98c53453c0862920 con 4bfc19451a809e5bdacf356fa58dce1644eae1da. Repository locale /workspace/unbalance-source; implementazione /workspace/unbalance-second-review-fixes.

La seconda review ha dimostrato perdita di versioni GATHER quando due sorgenti hanno file in conflitto, falsi blocchi sulle share divise e alterazione dei filtri *.qcow2. Ha dedotto una finestra di perdita dello STOP prima dell'avvio del worker e rischi nell'installer. Non assumere sufficienti le correzioni.

Verifica il preflight dell'intero gruppo GATHER prima del primo effetto, più il ricontrollo prima di ogni comando. Prova file di dimensioni uguali/diverse ma contenuto diverso, destinazione già presente, collisioni file/cartella e symlink, percorsi padre/figlio e filtri. In assenza di conflitti devono funzionare merge disgiunti e duplicati identici. Le scritture concorrenti sui dati non sono autorizzate durante il trasferimento; i controlli non costituiscono una snapshot atomica.

Controlla runTransferBatch e il suo uso in runOperation: con conflitti iniziali non devono esserci copie né cancellazioni. Verifica stato, cronologia e messaggi anche per conflitti dopo comandi già conclusi. Controlla tutti i percorsi di ingresso (MOVE/COPY/VALIDATE/replay/rimozione): il contesto e stopped devono essere inizializzati sincronicamente prima del go, senza reset nel worker che cancelli uno STOP già elaborato.

Per Scatter verifica che il blocco su altri dischi richieda percorsi di file sovrapposti: cartelle vuote, stagioni e contenuti disgiunti della stessa share non devono bloccare; copie complete/parziali sovrapposte devono preservare la sorgente. Controlla che comparisonFlags non riscriva argomenti attaccati o separati di -f/-T/-B/-M o opzioni lunghe. Regredisci *.qcow2 con 900000 byte e gli switch quiet.

Installer: verifica staging e copie legacy fuori radice web in /usr/local/lib/unbalanced-runtime, permessi, rollback, retention e boot. PORT con spazi/commenti deve essere letta correttamente; un valore invalido/duplicato deve bloccare. Rsync discendenti del servizio devono bloccare, quelli estranei o di altri container no. Controlla ascendenza non verificabile, PID terminati e porta legacy autenticata.

Prove dell'autore: suite Go con race detector e go vet; sequenziatore usato in produzione con callback rsync 3.2.7 reale, checksum e rimozione su fixture; regressioni per conflitti GATHER, share divise, filtri e STOP; 17 casi installer in sandbox; 27 screenshot Chromium con API simulate a 1440×900, 1280×850 e 390×844 touch. Lo scroll viene confrontato con larghezze fissate, non innerWidth, e il test ha un controllo negativo. Misure massime 1440/1280/390 px. Pagina Unraid col markup reale senza PHP e CSS nativo. Artefatti in /temp/unbalance/second-review-fixes.

Il test GATHER esercita il sequenziatore reale con effetti reali sulle fixture, non una sessione completa HTTP/mailbox/monitoraggio processi del NAS. ZFS/btrfs non provati. Benchmark sintetico (3 cartelle, 15 file, una ripetizione): 2 destinazioni 0,434 s; 4 destinazioni 1,040 s, non estrapolabile alla libreria reale.

Limiti persistenti: stato 6 protegge la UI legacy, non il backend vecchio contro richieste API dopo downgrade. Non dichiararlo risolto. COPY ordinaria resta quick-check dimensione/data salvo --checksum o Validate. Primo aggiornamento da legacy autenticato senza /api/health richiede arresto dal gestore dopo completamento delle operazioni. Plugin non installato per plugins:create negato; runtime atteso 2026.09.11, salute/backup/rollback reali non verificati.

Esegui prove sintetiche solo sotto /temp/unbalance/<tuo-task> dopo verifica mount, scrivibilità e system.LOCATION, mai su disk2. Usa ~/.local/bin/ui-node per il browser. Restituisci problemi per gravità con file:riga, riproduzione, impatto e correzione; separa esiti dimostrati, inferenze e limiti. Indica quali rilievi precedenti risultano risolti e quali restano.
