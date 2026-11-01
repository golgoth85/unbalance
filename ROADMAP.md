# Unbalanced — roadmap

## 2026-11-01 — correzione ripresa copie e interfaccia mobile

- Fork personale creato; upstream invariato, GitHub Actions disabilitato.
- Scatter: confronto rsync per ogni destinazione, stima spazio conservativa, inclusione duplicati e gestione degli errori senza piano utilizzabile.
- Preflight aggiornato prima della copia; verifica checksum integrale prima di eliminare sorgenti MOVE dei nuovi piani, anche per rimozione da cronologia. STOP cancella i confronti.
- UI adattabile: selezione, validazione, trasferimento, Gather, cronologia, impostazioni, log; pagina impostazioni Unraid adattata tramite CSS. Selettori accessibili da tastiera/touch; nessun NaN per cartelle senza dati da trasferire.
- Verifiche locali: suite Go con race detector, go vet, test rsync reale su fixture sintetiche, build TypeScript/Vite. Browser Chromium su API simulate a 1440×900, 1280×850, 390×844 touch: 24 screenshot; nessun errore JS o scroll orizzontale. Pagina Unraid: prova del CSS con campi sintetici, non esecuzione PHP sul NAS.
- Installer provato in sandbox: rsync attivo, Unbalanced occupato, aggiornamento con servizio avviato/spento, rollback per salute fallita. Configurazioni preservate.
- Pubblicazione pacchetto e installazione sul NAS: da completare dopo commit e merge; non interrompere copie reali. Prompt di review indipendente da consegnare con revisione esatta e limiti effettivi.
