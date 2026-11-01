# Unbalanced — roadmap

## 2026-11-01 — correzione ripresa copie e interfaccia mobile

- Fork personale creato; upstream invariato, GitHub Actions disabilitato.
- Scatter: confronto rsync per ogni destinazione, stima spazio conservativa, inclusione duplicati e gestione degli errori senza piano utilizzabile.
- Preflight aggiornato prima della copia; verifica checksum integrale prima di eliminare sorgenti MOVE dei nuovi piani, anche per rimozione da cronologia. STOP cancella i confronti.
- UI adattabile: selezione, validazione, trasferimento, Gather, cronologia, impostazioni, log; pagina impostazioni Unraid adattata tramite CSS. Selettori accessibili da tastiera/touch; nessun NaN per cartelle senza dati da trasferire.
- Verifiche locali: suite Go con race detector, go vet, test rsync reale su fixture sintetiche, build TypeScript/Vite. Browser Chromium su API simulate a 1440×900, 1280×850, 390×844 touch: 24 screenshot; nessun errore JS o scroll orizzontale. Pagina Unraid: prova del CSS con campi sintetici, non esecuzione PHP sul NAS.
- Installer provato in sandbox: rsync attivo, Unbalanced occupato, aggiornamento con servizio avviato/spento, rollback per salute fallita. Configurazioni preservate.
- Codice pubblicato e unito in `main`: `a40f0cc1d9128b39fcf069c0e1686f8d59106f14` (base upstream `cedfedf5e344e930c361391f2267ad6d5991326a`). Release `2026.11.01.1` pubblicata con `.tgz`, `.plg`, `SHA256SUMS`: digest degli asset GitHub uguali ai file locali. Binario Linux amd64 statico, manifest XML e shell validi.
- Installazione reale non eseguita: l'anteprima `unraid_plugin_install` è stata rifiutata con `FORBIDDEN: Permission denied: plugins:create`. Runtime NAS ancora `2026.09.11`; nessuna modifica a servizio, configurazioni, cronologia o dati reali. Non aggirare il diniego con altri canali. Nessun backup NAS creato, perché l'installer non è stato eseguito.
- Passo operativo residuo: installare il manifest della release dal gestore plugin Unraid o con uno strumento autorizzato quando non ci sono rsync/operazioni Unbalanced attivi; poi controllare versione HTTP, stato e pagina impostazioni reale. L'installer produce il backup codice sul NAS e sospende lo scambio se trova attività.
- Prompt consegnato in `REVIEW_PROMPT.md`; richiede review indipendente del diff, senza operazioni su dati reali. Screenshot e artefatti richiesti conservati in `/temp/unbalance/resume-mobile/`; fixture temporanee di test eliminate automaticamente. Toolchain e cache locali eliminate al termine dopo le verifiche.
