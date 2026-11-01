import React from 'react';

interface GaugeProps {
  value: string;
  unit?: string;
  label: string;
}

export const Gauge: React.FunctionComponent<GaugeProps> = ({
  value,
  unit = '',
  label,
}) => {
  return (
    <div className="border border-stroke dark:border-gray-800 border-slate-300 min-w-0 px-3 py-3 shadow-default dark:border-strokedark dark:bg-boxdark">
      <div className="flex justify-between items-center">
        <span className="text-xl lg:text-3xl break-words dark:text-slate-400 text-slate-600">
          {value}
        </span>
        <span className="text-blue-600">{unit}</span>
      </div>
      <span className="text-sm font-medium dark:text-slate-600 text-slate-400">
        {label}
      </span>
    </div>
  );
};
