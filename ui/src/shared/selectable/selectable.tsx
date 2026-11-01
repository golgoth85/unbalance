import React from 'react';

type SelectableProps = {
  onClick?: () => void;
  selected?: boolean;
  children?: React.ReactNode;
};

const selectedBackground = (selected: boolean) =>
  selected ? 'rounded dark:bg-gray-900 bg-neutral-300' : '';

export const Selectable: React.FunctionComponent<SelectableProps> = ({
  onClick,
  selected = false,
  children,
}) => (
  <div
    role={onClick ? "button" : undefined}
    tabIndex={onClick ? 0 : undefined}
    aria-pressed={onClick ? selected : undefined}
    onKeyDown={(event) => {
      if (onClick && event.target === event.currentTarget && (event.key === "Enter" || event.key === " ")) { event.preventDefault(); onClick(); }
    }}
    className={`min-h-11 py-2 px-3 ${selectedBackground(selected)}`}
    onClick={onClick}
  >
    {children}
  </div>
);
