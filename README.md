# Fork NASdelPala

Fork personale di `jbrodriguez/unbalance`, mantenuto su `golgoth85/unbalance`.
La pianificazione Scatter confronta sorgente e destinazione con rsync e riserva
solo lo spazio richiesto dai file mancanti/modificati, più un margine. I file
modificati vengono stimati per intero, per coprire i temporanei di rsync.
Le cartelle già complete restano nel piano. Ogni cartella viene confrontata su tutte le destinazioni e assegnata a quella capiente con meno dati da trasferire. La stima comprende allocazione della sorgente per voci mancanti e xattr differenti con -X. La stima non è un conteggio dei byte
fisicamente scritti; il dashboard indica dati previsti elaborati o simulati.

MOVE dei nuovi piani Scatter e Gather verifica la destinazione con checksum, senza filtri utente,
prima di cancellare gli originali. Errori e differenze preservano la sorgente;
STOP interrompe anche i confronti e la verifica. I piani devono essere rigenerati
se cambiano le opzioni rsync. Non scrivere sui file coinvolti durante il trasferimento.
L'interfaccia web e la pagina impostazioni Unraid si adattano a telefono e tablet.

I pacchetti del fork si trovano nelle [release](https://github.com/golgoth85/unbalance/releases).
L'installer aggiorna un'installazione esistente sulla porta configurata, preserva
configurazione e cronologia, verifica l'inattività, salva il codice precedente e
ripristina il codice se non supera il controllo HTTP. Non installare mentre
rsync o Unbalanced stanno lavorando. Non cambiano dati utente o schema.
Le sorgenti Scatter non vengono cancellate se gli stessi percorsi di file esistono su altri dischi; cartelle vuote e contenuti disgiunti delle share sono ammessi. Con filtri, --chmod o copia senza -X, differenze di metadati possono preservare le sorgenti: non sono autorizzazioni a rimuoverle manualmente. Lo stato di verifica fallita non è offerto a Remove source dalla vecchia UI dopo un rollback.
Il controllo COPY ordinario usa dimensione/data; per confrontare il contenuto usare Validate checksum. Il rollback verso una versione precedente non aggiunge verifiche ai suoi comandi storici.
Per il primo aggiornamento da una release legacy con autenticazione attiva occorre arrestare il plugin dal suo gestore dopo aver concluso le operazioni: senza endpoint di salute pubblico non è possibile accertarne l'inattività senza credenziali. Gli aggiornamenti successivi usano /api/health.
I backup di codice del nuovo installer sono limitati agli ultimi due; conserva separatamente eventuali rollback da mantenere più a lungo. Le directory rollback-* delle vecchie release non vengono eliminate automaticamente. Reinstallazioni identiche e avvii del NAS non producono backup aggiuntivi.
I vecchi record di cronologia mantengono il comportamento della versione che li ha generati.

Per sviluppo e verifiche consultare `AGENTS.md`; lo stato è in `ROADMAP.md`.

---

# unbalanced

_tl;dr_ **unbalanced** is an [Unraid](https://unraid.net/) plugin to transfer files/folders between disks in your array.

## Support Fund

If you wish to do so, read how to [support the developer](https://jbrio.net/unbalanced).

## [Changelog](https://github.com/jbrodriguez/unbalance/releases)

## Screenshot

![Home](meta/images/scatter-select.png)

## Introduction

unbalanced helps you manage space in your Unraid array, via two operating modes:

- **Scatter**<br/> Transfer data out of a disk, into one or more disks

- **Gather**<br/> Consolidate data from a user share into a single disk

It's versatile and can serve multiple purposes, based on your needs.

Some of the use cases are:

- Empty a disk, in order to change filesystems (read
  [kizer's example](https://forums.unraid.net/topic/43651-plug-in-unbalance/page/2/#comment-431428))
- Move all seasons of a tv show into a single disk
- Move a specific folder from a disk to another disk
- Split your movies/tv shows/games folder from a disk to other disks

You'll likely come up with other scenarios as you play around with it.

## Core Features

- **Makes sure to fill the target disk(s) as much as possible, without running out of space**<br> If it can't transfer
  some files/folders, it will inform you in the console and via notifications (if you set them up in the settings),
  before any actual transfer takes place.

- **Operates in the background**<br> You can close your browser while the transfer operation is ongoing. It will keep
  transferring files on the server and show you the current progress as soon as you reopen the browser.

- **Transfer operations work at the disk level (not at the user share level)**<br> This avoids file/folder clobbering.

> [!IMPORTANT]
> IMPORTANT: I suggest giving unbalanced exclusive access to disks (disable mover and/or any dockers that write to disks), so that free space calculation are not affected. If you're only reading data (streaming, etc.), it shouldn't be issue, although the operations may run slower.

## SCATTER Instructions

Scatter will transfer data from a source disk into one or more target disks, according to your selection, by filling the target disks, sorted by free space available, as much as possible.

> [!NOTE]
> Scatter doesn't distribute data evenly across disks. It will fill up the first disk, then the second, and so on.

It involves the following steps:

**1 - Select** <br/>
Choose the source disk, select the files/folders you want to transfer and choose the target disk(s).
![Select](meta/images/scatter-select.png)

**2 - Plan** <br/>
The logic is:

- Get the contents of the selected files/folders from the source disk
- Order the target disks by free space available
- For each target disk, calculate how much it can be filled up with files/folder from the source disk, leaving some
  headroom (currently set at 1Gb).

Additionally, it will check files/folders permissions, to warn about potential issues during the transfer stage.
![Plan](meta/images/scatter-plan.png)

**3 - Transfer** <br/> You can either MOVE or COPY the files.

- MOVE <br/> Will first copy the files/folders into the target disk(s), and delete them as soon as the copy is finished.

- COPY <br/> Will simply transfer the files/folders into the other disk(s).<br/> **NOTE**: Beware that COPY doesn't
  delete files/folders on the source disk, so you will be essentially duplicating the data.

~~Internally, all move operations are handled by [diskmv](https://github.com/trinapicot/Unraid-diskmv).~~

Internally, it issues a slight variation of
[this rsync command](https://forums.unraid.net/topic/35815-re-format-xfs-on-replacement-drive-convert-from-rfs-to-xfs-discussion-only/page/12/#comment-445880).

Check [this post](https://forums.unraid.net/topic/43651-plug-in-unbalance/page/11/#comment-471957) for additional
information.
![Transfer](meta/images/scatter-transfer.png)

**4 - Validate (optional)** <br/> VALIDATE will only be enabled for a SCATTER / COPY operation. Just click the Validate
button in the [History](#history) screen and the operation will be replayed, but with checksum comparisons (instead of
the simpler size/modification time check).

When using default flags, VALIDATE rsync will be invoked as **-rcvPRX**.

## GATHER Instructions

GATHER will consolidate data from a user share into a single disk.

It involves the following steps:

**1 - Select**

The 'Shares' column lets you navigate your user shares, to choose a folder to consolidate

When a folder is selected, the current selection and the drives where this folder is located are displayed in the 'Presence' column.

Once you've chosen the folder, click NEXT.
![Select](meta/images/gather-select.png)

**2 - Plan**
The logic is:

- Find the drives where the folder is located and how much space they occupy
- Calculate how much data needs to be transferred to each potential target disk

Additionally, it will check files/folders permissions, to warn about potential issues during the transfer stage.

![Plan](meta/images/gather-plan.png)

**3 - Move** <br/>

This page shows which drives have enough space to hold the contents of the folder chosen in the previous step.

They are shown in descending order by how little data transfer will be required.

A star next to a drive means that the folder is present there.

Choose a target disk, then click Move to start the transfer operation.
![Transfer](meta/images/gather-transfer.png)

## Installation

There are 2 ways to install this application

- Apps Tab (Community Application)<br/> Go to the Apps tab<br/> Click on the Plugins button (the last one)<br/> Look for
  unbalanced<br/> Click Install

- Plugins Tab (manual)<br/> Go to the Plugins tab<br/> Click on Install Plugin<br/> Paste the following address in the
  input field: <https://github.com/jbrodriguez/unbalance/releases/latest/download/unbalanced.plg><br/> Click Install

## Running the app

After installing the plugin, you can access the web UI, via the following methods:

- Method 1<br/> Go to Settings > Utilities<br/> Click on unbalanced<br/> Click on Open Web UI<br/>

- Method 2<br/> Go to Plugins > Installed Plugins<br/> Click on unbalanced<br/> Click on Open Web UI<br/>

- Method 3<br/> Navigate with your browser to http(s)://Tower:7090/ (replace Tower with the address/name of your Unraid
  server)<br/>

## Authentication

When authentication is enabled, the first administrator to open the web UI completes a one-time password setup.

Password requirements:

- Minimum length: 8 characters
- Maximum length: 256 characters
- Special characters: allowed
- Complexity rules: none (no uppercase, number, or symbol requirement)

New passwords are stored with Argon2id. Existing bcrypt hashes remain supported and are automatically upgraded to Argon2id after the next successful login.

## Other Features

### Transfer

Here you can monitor the progress of an operation. It shows overall metrics, as well as each invididual command as they
unfold.

![Transfer](meta/images/transfer.png)

### History

unbalanced keeps a history of the operations you have run, showing each command that was executed.

You can replay the most recent operation (excluding dry-runs) or validate the most recent Scatter Copy.

Additionally, you can review operations which contain one or more flagged rsync commands.

If an rsync command has been flagged (applies to Gather and Scatter/Move operations), unbalanced will not delete the source folders/files.

This allows you to check the issue in detail and take any action deemed necessary.

Once you've done that, you can delete the source folders/files through the UI, if you want.

![History](meta/images/history.png)

### Settings

These are pretty much straigthforward.

A word of caution with the custom rsync flags: it's for **power users** only.

unbalanced is optimized to work with the default flags, you must be VERY knowledgeable in rsync if you want to add any
flag.

![Settings](meta/images/settings.png)

### Log

![Log](meta/images/log.png)

## Videos

> NOTE: this refers to an older version of unbalanced, but the concepts are the same.

Thanks to gridrunner (Unraid forum member), you can watch unbalanced in action !

[Must Have Unraid Plugins - Part 3 Disk Utility Plugins](https://www.youtube.com/watch?v=Wz4-YlH1lTk)

The discussion specific to unbalanced starts [here](https://youtu.be/Wz4-YlH1lTk?t=859).

## Credits

~~This app uses the [diskmv](https://github.com/trinapicot/unraid-diskmv) script (check the
[forum thread](https://forums.unraid.net/topic/34547-diskmv-a-set-of-utilities-to-move-files-between-disks/) for additional information).~~

The icon was graciously created by
[hernandito](https://forums.unraid.net/topic/38028-docker-unbalance/#comment-368452) (fellow Unraid forums member)

It was built with:

- [Go](https://golang.org/) - Back End
- [echo](https://github.com/labstack/echo) - REST and websocket api
- [pubsub](https://github.com/cskr/pubsub) - Pub/Sub implementation
- [React](https://facebook.github.io/react/) - Front End
- [zustand](https://github.com/pmndrs/zustand) - Flux/Redux-like React framework
- [tailwind](https://tailwindcss.com) - Css framework
- [vite](https://vitejs.dev) - Tooling

## Development environment

- setup go development environment
- setup node development environment
- clone repository
- make release on root folder will create unbalanced binary (with embedded website)
- transfer binary to unraid server
- run unbalanced on the command line (do a ps aux | grep unbalanced first, to check how to invoke it)
- (optional) if you want to do some ui debugging, set up a proxy on vite.config.ts

### original

```js
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tsconfigPaths from "vite-tsconfig-paths";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tsconfigPaths()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
```

### modified

```js
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tsconfigPaths from "vite-tsconfig-paths";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tsconfigPaths()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http(s)://<unraid-server>:7090",
      },
    },
  },
});
```

then run `npm run dev` and point your browser to the endpoint returned by this command (generally <http://localhost:5173>)

## Activity

![Alt](https://repobeats.axiom.co/api/embed/c5745d8699df25ff4e087e691f104aeeda2bd17b.svg "Repobeats analytics image")

## License

MIT

GATHER controlla l'intero gruppo prima di copiare: versioni di file in conflitto tra sorgenti o con la destinazione bloccano il gruppo preservando tutte le sorgenti. Il controllo della destinazione viene ripetuto prima di ciascuna copia. Sono confrontati contenuti, tipi e destinazioni dei link; le scritture concorrenti sui file interessati devono restare sospese.
Lo stato 6 impedisce la rimozione dalla UI legacy, ma non modifica il backend di una vecchia versione ripristinata. Dopo un downgrade non inviare richieste di rimozione sorgenti via API su questi comandi: la verifica non è garantita dal backend vecchio.
L'installer usa /usr/local/lib/unbalanced-runtime, fuori dalla radice web, per staging e copie legacy. Blocca rsync discendenti del servizio, non processi indipendenti o in altri container. Una PORT malformata o duplicata sospende l'installazione; spazi e commenti sono supportati.

Il preflight GATHER considera anche gli xattr dei file con contenuto ordinario identico: versioni con attributi differenti fra sorgenti sono in conflitto. Gli attributi già presenti sulla destinazione non possono essere sovrascritti o eliminati con valori diversi; gli attributi soltanto mancanti possono essere completati. Il controllo include anche gli xattr delle directory selezionate, annidate e genitrici implicite inviate da -R.

I duplicati con contenuto e xattr uguali possono avere mtime, permessi o proprietario diversi: dopo GATHER prevalgono quelli dell'ultima sorgente elaborata, secondo rsync -a. La stessa regola vale per le cartelle padre implicite inviate da -R: anche se esistono già sulla destinazione e contengono dati estranei al trasferimento, ricevono mtime, permessi e proprietario della sorgente. Permessi più restrittivi possono limitare l'accesso ai contenuti preesistenti; valutare questa conseguenza prima del consolidamento. Le differenze di user.DOSATTRIB sono conflitti come quelle degli altri xattr; nessuna esclusione automatica è applicata, anche se l'attributo viene aggiornato da software esterno.
Il confronto rimuove quiet solo dagli switch, preservando argomenti attaccati/separati. Per opzioni lunghe con arità sconosciuta conserva il token successivo anziché interpretarlo come switch. Formati ambigui non vengono normalizzati silenziosamente.
L'installer accetta assegnazioni Bash numeriche PORT="9876", PORT='9876' o PORT=9876, con spazi iniziali/commento separato; rifiuta spazi intorno a =, commenti attaccati, quote errate e duplicati. Accetta anche export PORT=9876; una riga PORT con CRLF viene rifiutata. Se PORT manca, installer e scripts/start usano 7090. L’avvio valida il valore prima di fermare un servizio esistente. L’installer non esegue il contenuto del file.
