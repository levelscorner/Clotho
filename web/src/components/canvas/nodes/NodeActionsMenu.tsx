import { useState, useImperativeHandle, forwardRef } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { DotsThree } from 'phosphor-react';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { useExecutionStore } from '../../../stores/executionStore';
import {
  resolveFileURL,
  revealInFinder,
  isResolvableFile,
} from '../../../lib/api';
import './NodeActionsMenu.css';

// ---------------------------------------------------------------------------
// Per-node ⋯ actions menu. Replaces the prior hover-× button. Six items,
// two of which (Download / Reveal) only render when the node has a resolvable
// on-disk output. Locked nodes have Duplicate + Delete disabled and the Lock
// item toggles to "Unlock".
// ---------------------------------------------------------------------------

interface NodeActionsMenuProps {
  nodeId: string;
  label?: string;
}

export interface NodeActionsMenuHandle {
  /** Programmatically open the menu (used by right-click on the node body). */
  open: () => void;
}

export const NodeActionsMenu = forwardRef<NodeActionsMenuHandle, NodeActionsMenuProps>(
  function NodeActionsMenu({ nodeId, label }, ref) {
  const [open, setOpen] = useState(false);
  const removeNodes = usePipelineStore((s) => s.removeNodes);
  const duplicateNode = usePipelineStore((s) => s.duplicateNode);
  const toggleLock = usePipelineStore((s) => s.toggleLock);
  const startRename = usePipelineStore((s) => s.startRename);
  const isLocked = usePipelineStore((s) => s.lockedNodes.has(nodeId));
  const stepResult = useExecutionStore((s) => s.stepResults.get(nodeId));

  useImperativeHandle(ref, () => ({ open: () => setOpen(true) }), []);

  const output = stepResult?.output;
  const hasOutput = Boolean(output);
  const canResolveFile = hasOutput && isResolvableFile(output);

  const triggerAria = label ? `Actions for ${label}` : 'Node actions';

  return (
    <DropdownMenu.Root open={open} onOpenChange={setOpen}>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="clotho-node__menu-btn"
          aria-label={triggerAria}
          onClick={(e) => {
            e.stopPropagation();
          }}
          onMouseDown={(e) => e.stopPropagation()}
        >
          <DotsThree size={16} weight="bold" />
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className="clotho-menu"
          sideOffset={4}
          align="end"
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          <DropdownMenu.Item
            className="clotho-menu__item"
            onSelect={() => {
              if (!isLocked) duplicateNode(nodeId);
            }}
            disabled={isLocked}
          >
            <span>Duplicate</span>
            <span className="clotho-menu__shortcut">⌘D</span>
          </DropdownMenu.Item>

          <DropdownMenu.Item
            className="clotho-menu__item"
            onSelect={() => startRename(nodeId)}
          >
            <span>Rename</span>
            <span className="clotho-menu__shortcut">F2</span>
          </DropdownMenu.Item>

          <DropdownMenu.Separator className="clotho-menu__sep" />

          <DropdownMenu.Item
            className="clotho-menu__item"
            onSelect={() => toggleLock(nodeId)}
          >
            <span>{isLocked ? 'Unlock' : 'Lock'}</span>
            <span className="clotho-menu__shortcut">⌘L</span>
          </DropdownMenu.Item>

          <DropdownMenu.Separator className="clotho-menu__sep" />

          <DropdownMenu.Item
            className="clotho-menu__item clotho-menu__item--danger"
            onSelect={() => {
              if (!isLocked) removeNodes([nodeId]);
            }}
            disabled={isLocked}
          >
            <span>Delete</span>
            <span className="clotho-menu__shortcut">Del</span>
          </DropdownMenu.Item>

          {hasOutput && (
            <>
              <DropdownMenu.Separator className="clotho-menu__sep" />
              <DropdownMenu.Item
                className="clotho-menu__item"
                onSelect={() => {
                  if (!output) return;
                  window.open(resolveFileURL(output), '_blank');
                }}
              >
                <span>Download output</span>
              </DropdownMenu.Item>
              {canResolveFile && (
                <DropdownMenu.Item
                  className="clotho-menu__item"
                  onSelect={() => {
                    if (!output) return;
                    void revealInFinder(output);
                  }}
                >
                  <span>Reveal in folder</span>
                </DropdownMenu.Item>
              )}
            </>
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
});
