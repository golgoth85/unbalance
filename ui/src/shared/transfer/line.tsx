import React from 'react';

import { useUnraidOperation } from '~/state/unraid';

export const Line: React.FunctionComponent = () => {
  const operation = useUnraidOperation();

  if (!operation) {
    return null;
  }

  return <span className="break-all">{operation.line}</span>;
};
