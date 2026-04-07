import { create } from 'zustand';
import type { Node, Edge as RFEdge } from '@xyflow/react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Snapshot {
  nodes: Node[];
  edges: RFEdge[];
}

interface HistoryState {
  past: Snapshot[];
  future: Snapshot[];
  canUndo: boolean;
  canRedo: boolean;

  push: (snapshot: Snapshot) => void;
  undo: () => Snapshot | null;
  redo: () => Snapshot | null;
  clear: () => void;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const MAX_HISTORY = 50;

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useHistoryStore = create<HistoryState>((set, get) => ({
  past: [],
  future: [],
  canUndo: false,
  canRedo: false,

  push: (snapshot) => {
    set((state) => {
      const newPast =
        state.past.length >= MAX_HISTORY
          ? [...state.past.slice(1), snapshot]
          : [...state.past, snapshot];
      return {
        past: newPast,
        future: [],
        canUndo: newPast.length > 0,
        canRedo: false,
      };
    });
  },

  undo: () => {
    const { past, future } = get();
    if (past.length === 0) return null;

    const snapshot = past[past.length - 1];
    const newPast = past.slice(0, -1);

    set({
      past: newPast,
      future: [snapshot, ...future],
      canUndo: newPast.length > 0,
      canRedo: true,
    });

    // Return the state to restore (the one before the snapshot we just popped)
    // The snapshot we popped IS the state we saved before the mutation,
    // so restoring it undoes the mutation.
    return snapshot;
  },

  redo: () => {
    const { past, future } = get();
    if (future.length === 0) return null;

    const snapshot = future[0];
    const newFuture = future.slice(1);
    const newPast = [...past, snapshot];

    set({
      past: newPast,
      future: newFuture,
      canUndo: newPast.length > 0,
      canRedo: newFuture.length > 0,
    });

    return snapshot;
  },

  clear: () => {
    set({
      past: [],
      future: [],
      canUndo: false,
      canRedo: false,
    });
  },
}));
