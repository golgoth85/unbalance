import React from 'react';

export const GatherPolicy: React.FunctionComponent = () => <div className="space-y-2">
  <p>GATHER blocca xattr diversi di file, cartelle e genitori impliciti, compreso user.DOSATTRIB. Questi attributi non vengono ignorati automaticamente.</p>
  <p>Nei duplicati identici per contenuto e xattr, GATHER conserva mtime, permessi e proprietario dell’ultima sorgente elaborata. Valutare questa regola prima del consolidamento.</p>
</div>;
