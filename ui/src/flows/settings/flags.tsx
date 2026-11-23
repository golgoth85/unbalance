import React from 'react';

import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

import {
  useConfigActions,
  useConfigFullVerify,
  useConfigRsyncArgs,
} from '~/state/config';

export const Flags: React.FunctionComponent = () => {
  const flags = useConfigRsyncArgs();
  const [flagsValue, setFlagsValue] = React.useState(flags.join(' '));
  const [error, setError] = React.useState('');
  const fullVerify = useConfigFullVerify();
  const { setRsyncArgs, resetRsyncArgs, setFullVerify } = useConfigActions();
  const [verifyError, setVerifyError] = React.useState('');

  const onFullVerifyChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    try {
      await setFullVerify(e.target.checked);
      setVerifyError('');
    } catch (err) {
      setVerifyError(
        err instanceof Error ? err.message : 'Impossibile salvare l\'impostazione',
      );
    }
  };

  const onFlagsChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFlagsValue(e.target.value);
  };

  const onApply = async () => {
    const flags = flagsValue.split(/\s+/).filter(Boolean);
    try {
      await setRsyncArgs(flags);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unable to save rsync flags');
    }
  };

  const onReset = async () => {
    setFlagsValue('-X');
    try {
      await resetRsyncArgs();
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unable to reset rsync flags');
    }
  };

  return (
    <div className="p-4">
      <h1>
        Internally,{' '}
        <span className="text-lime-600 dark:text-lime-700">unbalanced</span>{' '}
        uses rsync to transfer files across disks. <br />
        By default, rsync is invoked with{' '}
        <span className="font-bold text-blue-900 dark:text-blue-600">
          -avPRX
        </span>{' '}
        flags. Note that the{' '}
        <span className="font-bold text-blue-900 dark:text-blue-600">X</span>{' '}
        flag is customizable, so you can remove it if needed. <br />
        You can add custom flags, except for the dry run flag which will be
        automatically added, if needed. <br />
        <span className="text-red-900 dark:text-red-700 font-bold">
          NOTE: These settings are meant to be changed by advanced users only.
          Destructive or remote-execution rsync options are blocked because
          unbalanced manages source deletion and local transfer behavior itself.
        </span>
      </h1>
      <div className="pb-4" />

      <div className="flex flex-wrap gap-y-3 items-center">
        <Input
          type="text"
          placeholder="rsync flags"
          className="w-40"
          value={flagsValue}
          onChange={onFlagsChange}
        />
        <div className="pr-4" />
        <Button onClick={onApply} variant="secondary">
          Apply
        </Button>
        <div className="pr-4" />
        <Button onClick={onReset} variant="secondary">
          Reset to Defaults
        </Button>
      </div>
      {error !== '' && (
        <p className="pt-3 text-sm text-red-900 dark:text-red-700">{error}</p>
      )}

      <div className="pb-8" />
      <h2 className="text-lg font-bold">Verifica prima di cancellare (MOVE/GATHER)</h2>
      <div className="pb-2" />
      <p className="text-sm max-w-prose">
        Prima di eliminare una sorgente, unbalanced controlla sempre che la
        destinazione contenga tutti i file con dimensione, data e attributi
        uguali. I file copiati in questa esecuzione sono già verificati da rsync
        con un checksum durante il trasferimento; il checksum viene ricalcolato
        solo sui file che erano già presenti sulla destinazione, perché rsync
        potrebbe averli saltati senza leggerne il contenuto.
      </p>
      <div className="pb-3" />
      <div className="flex items-start gap-3">
        <input
          checked={fullVerify}
          id="full-verify"
          type="checkbox"
          aria-describedby="full-verify-help"
          className="mt-1 w-5 h-5 shrink-0 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
          onChange={onFullVerifyChange}
        />
        <div>
          <label htmlFor="full-verify" className="font-medium">
            Verifica checksum completa
          </label>
          <p id="full-verify-help" className="text-sm text-gray-600 dark:text-gray-400 max-w-prose">
            Rilegge per intero sorgente e copia di ogni cartella. Protegge anche
            da errori di scrittura silenziosi, ma può raddoppiare o triplicare
            la durata di un MOVE. Spenta per impostazione predefinita.
          </p>
        </div>
      </div>
      {verifyError !== '' && (
        <p className="pt-3 text-sm text-red-900 dark:text-red-700">{verifyError}</p>
      )}
    </div>
  );
};
