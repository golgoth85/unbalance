import React from 'react';
import { useUnraidPlan } from '~/state/unraid';
import { humanBytes } from '~/helpers/units';

export const Description: React.FunctionComponent = () => {
  const plan = useUnraidPlan();
  const sourceSize = Object.values(plan?.vdisks ?? {}).reduce((sum, disk) =>
    sum + (disk.bin?.items ?? []).reduce((n, item) => n + item.size, 0), 0);
  return <span>
    Cartelle incluse: {humanBytes(sourceSize)}. Dati da trasferire: {humanBytes(plan?.bytesToTransfer ?? 0)} (stima rsync).
    I file già corrispondenti per dimensione e data vengono saltati; lo spazio riservato include i file temporanei e un margine per i metadati.
    COPY confronta dimensione e data: per conflitti di contenuto con dimensione/data uguali usare Validate checksum dalla cronologia.
    MOVE verifica contenuti e metadati prima della cancellazione. Senza -X, oppure con filtri o --chmod, eventuali differenze preservano la sorgente.
    Copie della stessa cartella su altri dischi impediscono la cancellazione della sorgente: consolidarle prima di MOVE.
    Scegli MOVE, COPY oppure dry-run. Con COPY gli originali restano sulla sorgente; lo spazio libero previsto sulla sorgente si riferisce a MOVE.
  </span>;
};
