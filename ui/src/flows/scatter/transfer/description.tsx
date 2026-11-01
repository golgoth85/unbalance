import React from 'react';
import { GatherPolicy } from '~/shared/gather-policy/gather-policy';
import { useUnraidPlan } from '~/state/unraid';
import { humanBytes } from '~/helpers/units';

export const Description: React.FunctionComponent = () => {
  const plan = useUnraidPlan();
  const sourceSize = Object.values(plan?.vdisks ?? {}).reduce((sum, disk) =>
    sum + (disk.bin?.items ?? []).reduce((n, item) => n + item.size, 0), 0);
  return <div className="space-y-2">
    <p>Cartelle incluse: {humanBytes(sourceSize)}. Dati da trasferire: {humanBytes(plan?.bytesToTransfer ?? 0)} (stima rsync).</p>
    <p>COPY conserva gli originali. MOVE verifica la destinazione prima di cancellarli. Lo spazio libero previsto sulla sorgente si riferisce a MOVE.</p>
    <details>
      <summary className="cursor-pointer min-h-11 flex items-center font-medium">Verifiche, metadati e conflitti</summary>
      <ul className="list-disc pl-5 space-y-2">
        <li>COPY predefinita confronta dimensione e data. Per i contenuti usare --checksum o Validate checksum dalla cronologia.</li>
        <li>La stima include temporanei e margine per i metadati. File con gli stessi percorsi relativi su altri dischi preservano la sorgente; cartelle vuote e contenuti disgiunti della share non bloccano MOVE.</li>
        <li>MOVE controlla contenuti e metadati. Filtri, --chmod o copia senza -X possono lasciare differenze e preservare gli originali.</li>
        <li><GatherPolicy /></li>
      </ul>
    </details>
  </div>;
};
