import React from 'react';

import { CommandStatus, Command as ICommand } from '~/types';
import { getCommandStatus } from '~/helpers/operation';

interface Props {
  command: ICommand;
  rsyncStrArgs: string;
  canBeFlagged?: boolean;
  onFlag?: (command: ICommand) => void;
}

export const Command: React.FunctionComponent<Props> = ({
  command,
  rsyncStrArgs,
  canBeFlagged = false,
  onFlag,
}) => {
  const progress = command.size > 0
    ? Math.min(100, (command.transferred / command.size) * 100).toFixed(0)
    : command.status === CommandStatus.Complete ? '100' : '0';
  const onClick = () => onFlag?.(command);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-2 items-center text-sm text-gray-500 dark:text-gray-500 p-2 border-b border-slate-300 dark:border-gray-700 ">
      <div className="lg:col-span-2 flex min-w-0 break-all items-center">
        {(command.status === CommandStatus.SourceRemoval ||
          command.status === CommandStatus.Flagged) &&
        canBeFlagged ? (
          <button onClick={onClick}>
            <span className="bg-red-100 text-red-800 text-xs font-medium px-2.5 py-0.5 rounded dark:bg-red-900 dark:text-red-300">
              rmsrc
            </span>
          </button>
        ) : (
          getCommandStatus(command.status)
        )}
        <span className="pr-2" />
        {command.src}
      </div>
      <div className="lg:col-span-8 min-w-0 break-all">
        rsync {rsyncStrArgs} &quot;{command.entry}&quot; &quot;{command.dst}
        &quot;
      </div>
      <div className="lg:col-span-2 flex flex-1 flex-row items-center">
        {progress !== '100' && <span className="text-xs">{progress} %</span>}
        <span className="px-1" />
        <div className="flex-1 rounded bg-gray-400 dark:bg-gray-800">
          <div
            className="p-0.5 leading-none rounded bg-blue-900 "
            style={{ width: `${progress}%` }}
          ></div>
        </div>
      </div>
      {command.status === CommandStatus.VerificationFailed && <div className="lg:col-span-12 text-amber-500">Sorgente preservata: verifica non superata. Correggere la copia e ripetere il trasferimento.</div>}
      {command.reason && (
        <div className="lg:col-span-12 break-all text-xs text-yellow-600 dark:text-yellow-600">
          {command.reason}
        </div>
      )}
    </div>
  );
};
