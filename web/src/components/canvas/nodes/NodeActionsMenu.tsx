import { useState } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import * as ContextMenu from '@radix-ui/react-context-menu';
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
// Per-node ⋯ actions menu AND right-click context menu.
//
// Two triggers, same items:
//   - <NodeActionsMenu> renders the visible three-dot button in the corner
//     of every node. Opens a Radix DropdownMenu anchored to that button.
//   - <NodeContextMenuProvider> wraps the whole node body and opens a Radix
//     ContextMenu anchored to the cursor on right-click. Same items, same
//     callbacks.
//
// The two share the MenuItems component below so adding/removing an item
// stays a one-place edit.
// ---------------------------------------------------------------------------

interface NodeActionsProps {
  nodeId: string;
  label?: string;
}

/**
 * Hook that centralises every menu-item action + the flags used to
 * disable/relabel them. Used by both the DropdownMenu (three-dot button)
 * and the ContextMenu (right-click) so behaviour cannot drift between
 * the two surfaces.
 */
function useNodeActions(nodeId: string) {
  const removeNodes = usePipelineStore((s) => s.removeNodes);
  const duplicateNode = usePipelineStore((s) => s.duplicateNode);
  const toggleLock = usePipelineStore((s) => s.toggleLock);
  const startRename = usePipelineStore((s) => s.startRename);
  const isLocked = usePipelineStore((s) => s.lockedNodes.has(nodeId));
  const stepResult = useExecutionStore((s) => s.stepResults.get(nodeId));

  const output = stepResult?.output;
  const hasOutput = Boolean(output);
  const canResolveFile = hasOutput && isResolvableFile(output);

  return {
    isLocked,
    hasOutput,
    canResolveFile,
    onDuplicate: () => {
      if (!isLocked) duplicateNode(nodeId);
    },
    onRename: () => startRename(nodeId),
    onToggleLock: () => toggleLock(nodeId),
    onDelete: () => {
      if (!isLocked) removeNodes([nodeId]);
    },
    onDownload: () => {
      if (!output) return;
      window.open(resolveFileURL(output), '_blank');
    },
    onReveal: () => {
      if (!output) return;
      void revealInFinder(output);
    },
  };
}

// --- Dropdown variant --------------------------------------------------------

function DropdownItems({ nodeId }: { nodeId: string }) {
  const a = useNodeActions(nodeId);
  return (
    <>
      <DropdownMenu.Item
        className="clotho-menu__item"
        onSelect={a.onDuplicate}
        disabled={a.isLocked}
      >
        <span>Duplicate</span>
        <span className="clotho-menu__shortcut">⌘D</span>
      </DropdownMenu.Item>

      <DropdownMenu.Item
        className="clotho-menu__item"
        onSelect={a.onRename}
      >
        <span>Rename</span>
        <span className="clotho-menu__shortcut">F2</span>
      </DropdownMenu.Item>

      <DropdownMenu.Separator className="clotho-menu__sep" />

      <DropdownMenu.Item
        className="clotho-menu__item"
        onSelect={a.onToggleLock}
      >
        <span>{a.isLocked ? 'Unlock' : 'Lock'}</span>
        <span className="clotho-menu__shortcut">⌘L</span>
      </DropdownMenu.Item>

      <DropdownMenu.Separator className="clotho-menu__sep" />

      <DropdownMenu.Item
        className="clotho-menu__item clotho-menu__item--danger"
        onSelect={a.onDelete}
        disabled={a.isLocked}
      >
        <span>Delete</span>
        <span className="clotho-menu__shortcut">Del</span>
      </DropdownMenu.Item>

      {a.hasOutput && (
        <>
          <DropdownMenu.Separator className="clotho-menu__sep" />
          <DropdownMenu.Item className="clotho-menu__item" onSelect={a.onDownload}>
            <span>Download output</span>
          </DropdownMenu.Item>
          {a.canResolveFile && (
            <DropdownMenu.Item className="clotho-menu__item" onSelect={a.onReveal}>
              <span>Reveal in folder</span>
            </DropdownMenu.Item>
          )}
        </>
      )}
    </>
  );
}

// --- Context-menu variant ----------------------------------------------------

function ContextItems({ nodeId }: { nodeId: string }) {
  const a = useNodeActions(nodeId);
  return (
    <>
      <ContextMenu.Item
        className="clotho-menu__item"
        onSelect={a.onDuplicate}
        disabled={a.isLocked}
      >
        <span>Duplicate</span>
        <span className="clotho-menu__shortcut">⌘D</span>
      </ContextMenu.Item>

      <ContextMenu.Item
        className="clotho-menu__item"
        onSelect={a.onRename}
      >
        <span>Rename</span>
        <span className="clotho-menu__shortcut">F2</span>
      </ContextMenu.Item>

      <ContextMenu.Separator className="clotho-menu__sep" />

      <ContextMenu.Item
        className="clotho-menu__item"
        onSelect={a.onToggleLock}
      >
        <span>{a.isLocked ? 'Unlock' : 'Lock'}</span>
        <span className="clotho-menu__shortcut">⌘L</span>
      </ContextMenu.Item>

      <ContextMenu.Separator className="clotho-menu__sep" />

      <ContextMenu.Item
        className="clotho-menu__item clotho-menu__item--danger"
        onSelect={a.onDelete}
        disabled={a.isLocked}
      >
        <span>Delete</span>
        <span className="clotho-menu__shortcut">Del</span>
      </ContextMenu.Item>

      {a.hasOutput && (
        <>
          <ContextMenu.Separator className="clotho-menu__sep" />
          <ContextMenu.Item className="clotho-menu__item" onSelect={a.onDownload}>
            <span>Download output</span>
          </ContextMenu.Item>
          {a.canResolveFile && (
            <ContextMenu.Item className="clotho-menu__item" onSelect={a.onReveal}>
              <span>Reveal in folder</span>
            </ContextMenu.Item>
          )}
        </>
      )}
    </>
  );
}

// --- Public components -------------------------------------------------------

/**
 * The three-dot button rendered in the node corner. Opens the menu
 * anchored to itself. Kept a drop-in replacement for the prior export.
 */
export function NodeActionsMenu({ nodeId, label }: NodeActionsProps) {
  const [open, setOpen] = useState(false);
  const triggerAria = label ? `Actions for ${label}` : 'Node actions';

  return (
    <DropdownMenu.Root open={open} onOpenChange={setOpen}>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="clotho-node__menu-btn"
          aria-label={triggerAria}
          onClick={(e) => e.stopPropagation()}
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
          <DropdownItems nodeId={nodeId} />
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

/**
 * Wraps a node's visible body so a right-click anywhere on it opens the
 * same menu at the cursor position. The `children` is the entire node
 * content; ContextMenu.Trigger renders it untouched in normal flow.
 */
export function NodeContextMenuProvider({
  nodeId,
  children,
}: {
  nodeId: string;
  children: React.ReactNode;
}) {
  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger asChild>{children}</ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content
          className="clotho-menu"
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          <ContextItems nodeId={nodeId} />
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  );
}
