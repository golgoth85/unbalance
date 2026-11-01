import React from 'react';

import { Icon } from '~/shared/icons/icon';
import diskmv from '~/assets/diskmv.png';
import unraid from '~/assets/unraid.png';
import jb from '~/assets/jb.png';
import { useConfigVersion } from '~/state/config';

export const Footer: React.FunctionComponent = () => {
  const version = useConfigVersion();

  return (
    <section className="flex flex-wrap gap-3 items-center justify-between bg-gray-800 dark:bg-gray-800 text-sky-700 dark:text-slate-300 p-2 my-4">
      <div>
        <span className="text-slate-500 dark:text-slate-600 mr-1">
          Copyright &copy;
        </span>
        <a
          href="https://jbrio.net/"
          target="_blank"
          aria-label="Sito dello sviluppatore" title="jbrio.net"
          className="text-lime-500 break-words"
        >
          Juan B. Rodriguez
        </a>
      </div>
      <div className="text-lime-500 break-words">
        <span>unbalanced</span>
        {version !== '' && <span>&nbsp;{version}</span>}
      </div>
      <div className="flex flex-wrap gap-y-2 items-center">
        <a
          href="https://jbrio.net/unbalanced"
          aria-label="Support Fund" title="Support Fund"
          rel="noreferrer noopener"
          target="_blank"
        >
          <Icon name="gift" size={24} style="fill-lime-600" />
        </a>

        <a
          href="https://x.com/jbrodriguezio"
          aria-label="Profilo X" title="@jbrodriguezio"
          rel="noreferrer noopener"
          target="_blank"
          className="ml-2"
        >
          <Icon name="x" size={20} style="fill-neutral-300" />
        </a>

        <a
          href="https://github.com/jbrodriguez"
          aria-label="Profilo GitHub" title="github.com/jbrodriguez"
          rel="noreferrer noopener"
          target="_blank"
          className="ml-2"
        >
          <Icon name="github" size={24} style="fill-neutral-300" />
        </a>

        <a
          href="https://forums.unraid.net/topic/34547-diskmv-a-set-of-utilities-to-move-files-between-disks/"
          aria-label="diskmv" title="diskmv"
          rel="noreferrer noopener"
          target="_blank"
        >
          <img src={diskmv} alt="logo" className="h-10" />
        </a>

        <a
          className="ml-2"
          href="https://unraid.net/"
          aria-label="Unraid" title="unraid.net"
          rel="noreferrer noopener"
          target="_blank"
        >
          <img src={unraid} alt="logo" className="h-8" />
        </a>

        <a
          className="ml-3"
          href="https://jbrio.net/"
          aria-label="Sito dello sviluppatore" title="jbrio.net"
          rel="noreferrer noopener"
          target="_blank"
        >
          <img src={jb} alt="logo" className="h-8" />
        </a>
      </div>
    </section>
  );
};
