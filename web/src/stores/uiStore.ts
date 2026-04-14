import { create } from 'zustand';

// ---------------------------------------------------------------------------
// UI store — ephemeral UI state that multiple parts of the app coordinate on
// (modals, slide-overs, responsive drawers).
//
// Kept deliberately small. Each new surface adds its own boolean + actions.
// ---------------------------------------------------------------------------

interface UIState {
  templateGalleryOpen: boolean;
  openTemplateGallery: () => void;
  closeTemplateGallery: () => void;
  toggleTemplateGallery: () => void;

  // Responsive: palette drawer at phone breakpoint (<768px)
  mobilePaletteOpen: boolean;
  openMobilePalette: () => void;
  closeMobilePalette: () => void;
  toggleMobilePalette: () => void;

  // Responsive: inspector drawer/modal explicit close (drives backdrop tap /
  // Escape behavior at tablet + phone breakpoints).
  mobileInspectorDismissed: boolean;
  dismissMobileInspector: () => void;
  resetMobileInspectorDismissed: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  templateGalleryOpen: false,
  openTemplateGallery: () => set({ templateGalleryOpen: true }),
  closeTemplateGallery: () => set({ templateGalleryOpen: false }),
  toggleTemplateGallery: () =>
    set((s) => ({ templateGalleryOpen: !s.templateGalleryOpen })),

  mobilePaletteOpen: false,
  openMobilePalette: () => set({ mobilePaletteOpen: true }),
  closeMobilePalette: () => set({ mobilePaletteOpen: false }),
  toggleMobilePalette: () =>
    set((s) => ({ mobilePaletteOpen: !s.mobilePaletteOpen })),

  mobileInspectorDismissed: false,
  dismissMobileInspector: () => set({ mobileInspectorDismissed: true }),
  resetMobileInspectorDismissed: () =>
    set({ mobileInspectorDismissed: false }),
}));
