import React from 'react';

import { Node, Icons } from '~/types';

interface TreeNodeProps {
  node: Node;
  getChildNodes: (node: Node) => Node[];
  onExpandCollapse: (node: Node) => void;
  onCheckUncheck: (node: Node, shiftKey: boolean) => void;
  icons: Icons;
  level: number;
}

export const TreeNode: React.FunctionComponent<TreeNodeProps> = ({
  node,
  getChildNodes,
  onExpandCollapse,
  onCheckUncheck,
  icons,
  level,
}) => {
  const renderNode = (node: Node) => {
    if (node.loading) {
      return (
        <>
          <span className="ml-1">{icons.loadingIcon}</span>
          <span className="ml-1">{node.label}</span>
        </>
      );
    }

    return (
      <>
        {!node.leaf ? (
          <button type="button" aria-label={`Apri o chiudi ${node.label}`} aria-expanded={node.expanded} className="min-w-11 min-h-11 flex items-center justify-center" onClick={() => onExpandCollapse(node)}>
            {node.expanded ? icons.collapseIcon : icons.expandIcon}
          </button>
        ) : (
          <span>{icons.hiddenIcon}</span>
        )}

        <button
          type="button"
          aria-label={`Seleziona ${node.label}`}
          aria-pressed={node.checked}
          className="ml-1 min-w-11 min-h-11 flex items-center justify-center"
          onMouseDown={(e) => e.shiftKey && e.preventDefault()}
          onClick={(e) => onCheckUncheck(node, e.shiftKey)}
        >
          {node.checked ? icons.checkedIcon : icons.uncheckedIcon}
        </button>

        <span className="ml-1">
          {node.leaf
            ? node.dir
              ? icons.parentIcon
              : icons.leafIcon
            : icons.parentIcon}
        </span>

        <span className="ml-1">{node.label}</span>
      </>
    );
  };

  return (
    <>
      <div
        style={{ paddingLeft: `${level * 24}px` }}
        className="flex flex-1 flex-row items-center whitespace-nowrap"
      >
        {renderNode(node)}
      </div>
      {node.expanded &&
        getChildNodes(node).map((childNode) => (
          <TreeNode
            key={childNode.id}
            node={childNode}
            getChildNodes={getChildNodes}
            onExpandCollapse={onExpandCollapse}
            onCheckUncheck={onCheckUncheck}
            level={level + 1}
            icons={icons}
          />
        ))}
    </>
  );
};
