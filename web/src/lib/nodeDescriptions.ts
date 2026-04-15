// Short one-line descriptions per built-in node type + media type + preset.
// Used on the node body (teaser) and in the inspector's "About" section.

export interface NodeDescription {
  teaser: string;   // one-liner shown on node body (max ~80 chars)
  full: string;     // longer description for the inspector
  input: string;    // what the node expects (for the inspector's IO table)
  output: string;   // what the node produces
}

// Keyed by a synthetic "kind" string:
//   "agent"                              - base agent (Prompt Agent tile)
//   "agent:<preset_category>"            - agent with a specific preset
//                                           (script / crafter / generic)
//   "media:<media_type>"                 - image / video / audio
//   "tool:<tool_type>"                   - text_box / image_box / video_box
const DESCRIPTIONS: Record<string, NodeDescription> = {
  'agent': {
    teaser: 'Text LLM — send a prompt, get a response',
    full: 'A generic text agent. Configure the prompt, system message, and provider in the inspector. Uses any registered LLM provider (OpenAI, Gemini, Ollama, etc.).',
    input: 'text (optional — the previous step\'s output, referenced as {{input}})',
    output: 'text',
  },
  'agent:script': {
    teaser: 'Writes narrative scenes and voice-over scripts',
    full: 'A script-writing personality that turns your idea into a structured scene or voice-over. Accepts optional context; returns prose text you can feed into a prompt crafter or TTS node.',
    input: 'text (prior context, optional)',
    output: 'text (script)',
  },
  'agent:crafter': {
    teaser: 'Turns narrative text into image/video generation prompts',
    full: 'Converts loose narrative description into a precise, detailed prompt for image or video models. Enumerates subject, lighting, composition, camera, and style automatically.',
    input: 'text (scene description)',
    output: 'image_prompt / video_prompt',
  },
  'media:image': {
    teaser: 'Generates an image from a text prompt',
    full: 'Sends the prompt to the selected image model (ComfyUI/FLUX locally, or a hosted provider). The resulting PNG is saved to your Documents/Clotho folder and displayed in the node.',
    input: 'image_prompt / text',
    output: 'image (PNG, saved to disk)',
  },
  'media:video': {
    teaser: 'Generates a video clip from a prompt',
    full: 'Submits the prompt to the selected video model. Runtime varies by provider (~30s–3min). The resulting MP4 is saved to your Documents/Clotho folder.',
    input: 'video_prompt / text (optionally: image for image-to-video)',
    output: 'video (MP4, saved to disk)',
  },
  'media:audio': {
    teaser: 'Generates speech from text (TTS)',
    full: 'Text-to-speech using the selected voice model. Kokoro runs locally (~4s/sentence). The resulting MP3 is saved to your Documents/Clotho folder.',
    input: 'text (script or line)',
    output: 'audio (MP3, saved to disk)',
  },
  'tool:text_box': {
    teaser: 'Static text input — paste or type content to feed the pipeline',
    full: 'A passive source of text. Nothing runs when the pipeline executes — the content you type here is passed downstream as-is. Useful for pinning a system prompt, reference copy, or sample input.',
    input: '(none)',
    output: 'text',
  },
  'tool:image_box': {
    teaser: 'Static image input — reference image for downstream nodes',
    full: 'A passive source for an image URL. The image isn\'t generated; it\'s carried forward into the pipeline so downstream nodes (e.g., image-to-video) can reference it.',
    input: '(none)',
    output: 'image (URL)',
  },
  'tool:video_box': {
    teaser: 'Static video input — reference video for downstream nodes',
    full: 'A passive source for a video URL. Useful for video-to-video pipelines where you want to remix or extend existing footage.',
    input: '(none)',
    output: 'video (URL)',
  },
};

export interface DescriptionLookupArgs {
  nodeType: 'agent' | 'media' | 'tool';
  mediaType?: string;      // image/video/audio
  toolType?: string;       // text_box / image_box / video_box
  presetCategory?: string; // script / crafter / generic
  presetDescription?: string; // if a personality preset has its own description
}

const FALLBACK: NodeDescription = {
  teaser: 'Custom node',
  full: 'A user-defined node.',
  input: '(unknown)',
  output: '(unknown)',
};

export function describeNode(args: DescriptionLookupArgs): NodeDescription {
  // If this is an agent with a preset that provides its own description,
  // prefer that — it's the most user-intentional source.
  if (args.nodeType === 'agent' && args.presetDescription) {
    return {
      teaser: args.presetDescription.slice(0, 80),
      full: args.presetDescription,
      input: DESCRIPTIONS['agent'].input,
      output: DESCRIPTIONS['agent'].output,
    };
  }

  let key: string = args.nodeType;
  if (args.nodeType === 'agent' && args.presetCategory) {
    key = `agent:${args.presetCategory}`;
  }
  if (args.nodeType === 'media' && args.mediaType) {
    key = `media:${args.mediaType}`;
  }
  if (args.nodeType === 'tool' && args.toolType) {
    key = `tool:${args.toolType}`;
  }

  return DESCRIPTIONS[key] ?? DESCRIPTIONS[args.nodeType] ?? FALLBACK;
}
