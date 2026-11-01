import React from 'react';

import { Icon } from '~/shared/icons/icon';

interface Props {
  checked: boolean;
  onCheck: () => void;
  label?: string;
}

export const Checkbox: React.FunctionComponent<Props> = ({
  checked,
  onCheck,
  label = "Seleziona",
}) => {
  return (
    <button type="button" role="checkbox" aria-checked={!!checked} aria-label={label} className="min-w-11 min-h-11 shrink-0 inline-flex items-center justify-center" onClick={onCheck}>
      {checked ? (
        <Icon
          name="checked"
          size={20}
          style="fill-slate-700 dark:fill-lime-600"
        />
      ) : (
        <Icon
          name="unchecked"
          size={20}
          style="fill-slate-700 dark:fill-slate-200"
        />
      )}
    </button>
  );
};
