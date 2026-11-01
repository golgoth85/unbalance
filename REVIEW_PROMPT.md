Esegui una review indipendente del fork https://github.com/golgoth85/unbalance.

Leggi /workspace/AGENTS.md e l'AGENTS.md del progetto. Lavora in un tuo branch/worktree; non modificare l'upstream e non installare aggiornamenti o avviare copie/cancellazioni sul NAS durante questa review.

Confronta la base cedfedf5e344e930c361391f2267ad6d5991326a con la revisione di codice a40f0cc1d9128b39fcf069c0e1686f8d59106f14, pubblicata nella release 2026.11.01.1. Leggi anche ROADMAP.md sul main corrente per l'esito operativo. Repository locale: /workspace/unbalance-source; implementazione: /workspace/unbalance-resume-mobile.

Problema: Scatter calcolava lo spazio sulle cartelle sorgenti intere, anche se quasi tutte già copiate, escludendo dal piano cartelle ancora mancanti. Il fork stima il delta con rsync per destinazione, riserva spazio prudenziale per temporanei/metadati, include cartelle con delta nullo e ricontrolla prima dell'esecuzione. MOVE dei nuovi piani verifica la destinazione con checksum senza filtri prima della cancellazione. STOP deve interrompere anche i confronti; opzioni rsync cambiate invalidano il piano. L'interfaccia web e il template impostazioni Unraid sono adattati al mobile. La pianificazione backend GATHER mantiene il comportamento precedente; i vecchi record di cronologia non diventano retroattivamente verificati.

Non assumere corretta l'implementazione. Cerca soprattutto sottostime dello spazio, perdita/omissione di file, cancellazioni premature, conflitti di contenuto con dimensione/data uguali, filtri e skip flag, errori rsync, symlink, nomi particolari, cancellazione e corse concorrenti, ticket obsoleti, compatibilità cronologia/replay/validate e casi completamente già copiati. Controlla che nessun errore di rimozione venga mostrato come successo. Valuta anche installer, provenienza/hash degli artefatti, preservazione configurazione, controllo attività, backup e rollback.

Esegui verifiche indipendenti su dati sintetici esclusivamente sotto /temp/unbalance/<tuo-task>, dopo verifica mount e system.LOCATION, mai su disk2. Sono disponibili tests/browser.cjs, tests/installer.py e i test Go. Per il browser usa ~/.local/bin/ui-node, la UI compilata in preview e API simulate: controlla 1440×900, 1280×850 e 390×844 con isMobile/hasTouch, interazioni, tastiera/focus, console e scroll. Non considerare gli screenshot da soli prova della correttezza del backend.

Prove dell'autore: suite Go con race detector, go vet, rsync reale su fixture sintetiche, build TypeScript/Vite, 24 screenshot Chromium con API simulate e 5 casi installer in sandbox. Artefatti: /temp/unbalance/resume-mobile/. La pagina Unraid è stata provata come CSS/campi sintetici, senza eseguire PHP sul NAS. I contatori indicano dati stimati elaborati, non byte fisici scritti.

Limite operativo: plugin NON installato sul NAS. UnraidClaw ha rifiutato anche il dry run per permesso mancante plugins:create; runtime ancora 2026.09.11. La release è pubblicata ma salute, backup e rollback reali della nuova versione sul NAS non sono verificati. Non aggirare il diniego.

Restituisci prima i rilievi ordinati per gravità, ciascuno con file:riga, scenario riproducibile, impatto e correzione suggerita. Separa problemi dimostrati, ipotesi da verificare e limiti dei test; se non trovi problemi, dichiaralo insieme ai rischi residui. Non avviare review a pagamento.
