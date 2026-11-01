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
    Scegli MOVE, COPY oppure dry-run. Con COPY gli originali restano sulla sorgente; lo spazio libero previsto sulla sorgente si riferisce a MOVE.
  </span>;
};
