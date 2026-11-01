import React from 'react';
import { GatherPolicy } from '~/shared/gather-policy/gather-policy';

export const Description: React.FunctionComponent = () => <div className="space-y-2">
  <p>Seleziona un disco di destinazione e poi MOVE. I dischi sono ordinati per dati da trasferire; la stella indica cartelle già presenti. Il preflight blocca i conflitti prima della copia.</p>
  <details>
    <summary className="cursor-pointer min-h-11 flex items-center font-medium">Metadati del consolidamento</summary>
    <GatherPolicy />
  </details>
</div>;
