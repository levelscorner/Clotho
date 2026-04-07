package templates

import "encoding/json"

// Template is a pre-built pipeline graph that users can fork.
type Template struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"` // video, image, text, audio
	NodeCount   int             `json:"node_count"`
	Graph       json.RawMessage `json:"graph"`
}

// TemplateSummary is the list-view representation (no graph payload).
type TemplateSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	NodeCount   int    `json:"node_count"`
}

// Summary returns a TemplateSummary without the graph.
func (t Template) Summary() TemplateSummary {
	return TemplateSummary{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		NodeCount:   t.NodeCount,
	}
}

// All returns every built-in template.
func All() []Template {
	return []Template{
		youtubeStory,
		instagramReel,
		characterSheet,
		scriptToStoryboard,
		promptEnhancerChain,
	}
}

// ByID returns a template by its ID, or nil if not found.
func ByID(id string) *Template {
	for _, t := range All() {
		if t.ID == id {
			return &t
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 1. YouTube Story — 5 nodes, 4 edges
// ---------------------------------------------------------------------------

var youtubeStory = Template{
	ID:          "youtube-story",
	Name:        "YouTube Story",
	Description: "Full video pipeline: script, character design, scene direction, images, and video generation",
	Category:    "video",
	NodeCount:   5,
	Graph: json.RawMessage(`{
  "nodes": [
    {
      "id": "yt_script",
      "type": "agent",
      "label": "Script Writer",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "text", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are an experienced screenwriter skilled in visual storytelling. You write vivid, concise scripts with clear scene descriptions, dialogue, and stage directions. Focus on conveying emotion and visual impact in every scene.",
          "persona": "Screenwriter",
          "variables": {}
        },
        "task": {
          "task_type": "script",
          "output_type": "text",
          "template": "Write a script based on the following input:\n\n{{input}}\n\nInclude scene headings, action lines, and dialogue. Keep it vivid and cinematic.",
          "output_schema": null
        },
        "temperature": 0.8,
        "max_tokens": 4096
      }
    },
    {
      "id": "yt_character",
      "type": "agent",
      "label": "Character Designer",
      "position": {"x": 450, "y": 80},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "text", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are a character artist and designer who creates detailed character profiles. You describe physical appearance, clothing, personality traits, and visual style in a way that ensures consistent representation across multiple images and scenes.",
          "persona": "Character Artist",
          "variables": {}
        },
        "task": {
          "task_type": "character_prompt",
          "output_type": "text",
          "template": "Design a detailed character based on the following concept:\n\n{{input}}\n\nInclude physical description, clothing/accessories, personality traits, color palette, and visual style guide for consistent AI-generated imagery.",
          "output_schema": null
        },
        "temperature": 0.8,
        "max_tokens": 2048
      }
    },
    {
      "id": "yt_scene",
      "type": "agent",
      "label": "Scene Director",
      "position": {"x": 450, "y": 320},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "image_prompt", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are an adaptation specialist who converts narrative stories into sequences of detailed image generation prompts. You identify key visual moments, maintain character consistency, and craft prompts that when generated together tell the story visually.",
          "persona": "Adaptation Specialist",
          "variables": {}
        },
        "task": {
          "task_type": "story_to_prompt",
          "output_type": "image_prompt",
          "template": "Convert the following story into a sequence of image generation prompts:\n\n{{input}}\n\nExtract key visual moments and create detailed, consistent image prompts for each scene. Maintain character appearance and style consistency across all prompts.",
          "output_schema": null
        },
        "temperature": 0.7,
        "max_tokens": 4096
      }
    },
    {
      "id": "yt_imagegen",
      "type": "media",
      "label": "Image Generator",
      "position": {"x": 800, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "image_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "image", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "image",
        "provider": "replicate",
        "model": "flux-1.1-pro",
        "prompt": "{{input}}",
        "aspect_ratio": "16:9",
        "num_outputs": 1
      }
    },
    {
      "id": "yt_videogen",
      "type": "media",
      "label": "Video Generator",
      "position": {"x": 1150, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "video_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "video", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "video",
        "provider": "replicate",
        "model": "wan-2.1",
        "prompt": "{{input}}",
        "aspect_ratio": "16:9",
        "duration": 5,
        "num_outputs": 1
      }
    }
  ],
  "edges": [
    {"id": "yt_e1", "source": "yt_script", "source_port": "out", "target": "yt_character", "target_port": "in"},
    {"id": "yt_e2", "source": "yt_script", "source_port": "out", "target": "yt_scene", "target_port": "in"},
    {"id": "yt_e3", "source": "yt_scene", "source_port": "out", "target": "yt_imagegen", "target_port": "in_prompt"},
    {"id": "yt_e4", "source": "yt_imagegen", "source_port": "out_media", "target": "yt_videogen", "target_port": "in_ref"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}`),
}

// ---------------------------------------------------------------------------
// 2. Instagram Reel — 3 nodes, 2 edges
// ---------------------------------------------------------------------------

var instagramReel = Template{
	ID:          "instagram-reel",
	Name:        "Instagram Reel",
	Description: "Quick reel pipeline: enhance your prompt, generate an image, then animate it into video",
	Category:    "video",
	NodeCount:   3,
	Graph: json.RawMessage(`{
  "nodes": [
    {
      "id": "ig_enhance",
      "type": "agent",
      "label": "Prompt Enhancer",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "image_prompt", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are a prompt engineer specializing in AI image generation. You craft detailed, structured prompts that produce stunning visual results. You understand composition, lighting, style descriptors, and negative prompts for models like Stable Diffusion, Midjourney, and DALL-E.",
          "persona": "Prompt Engineer",
          "variables": {}
        },
        "task": {
          "task_type": "image_prompt",
          "output_type": "image_prompt",
          "template": "Create a detailed image generation prompt based on the following description:\n\n{{input}}\n\nInclude style, composition, lighting, color palette, and mood. Format as a single prompt string optimized for AI image generation.",
          "output_schema": null
        },
        "temperature": 0.7,
        "max_tokens": 1024
      }
    },
    {
      "id": "ig_imagegen",
      "type": "media",
      "label": "Image Generator",
      "position": {"x": 450, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "image_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "image", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "image",
        "provider": "replicate",
        "model": "flux-1.1-pro",
        "prompt": "{{input}}",
        "aspect_ratio": "9:16",
        "num_outputs": 1
      }
    },
    {
      "id": "ig_videogen",
      "type": "media",
      "label": "Video Generator",
      "position": {"x": 800, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "video_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "video", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "video",
        "provider": "replicate",
        "model": "wan-2.1",
        "prompt": "{{input}}",
        "aspect_ratio": "9:16",
        "duration": 5,
        "num_outputs": 1
      }
    }
  ],
  "edges": [
    {"id": "ig_e1", "source": "ig_enhance", "source_port": "out", "target": "ig_imagegen", "target_port": "in_prompt"},
    {"id": "ig_e2", "source": "ig_imagegen", "source_port": "out_media", "target": "ig_videogen", "target_port": "in_ref"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}`),
}

// ---------------------------------------------------------------------------
// 3. Character Sheet — 2 nodes, 1 edge
// ---------------------------------------------------------------------------

var characterSheet = Template{
	ID:          "character-sheet",
	Name:        "Character Sheet",
	Description: "Design a character and generate multiple reference images for consistency",
	Category:    "image",
	NodeCount:   2,
	Graph: json.RawMessage(`{
  "nodes": [
    {
      "id": "cs_designer",
      "type": "agent",
      "label": "Character Designer",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "image_prompt", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are a character artist and designer who creates detailed character profiles. You describe physical appearance, clothing, personality traits, and visual style in a way that ensures consistent representation across multiple images and scenes.",
          "persona": "Character Artist",
          "variables": {}
        },
        "task": {
          "task_type": "character_prompt",
          "output_type": "image_prompt",
          "template": "Design a detailed character based on the following concept:\n\n{{input}}\n\nProvide a single, detailed image generation prompt capturing the character's full appearance, clothing, color palette, and visual style. Optimized for AI image generation.",
          "output_schema": null
        },
        "temperature": 0.8,
        "max_tokens": 2048
      }
    },
    {
      "id": "cs_imagegen",
      "type": "media",
      "label": "Image Generator",
      "position": {"x": 500, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "image_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "image", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "image",
        "provider": "replicate",
        "model": "flux-1.1-pro",
        "prompt": "{{input}}",
        "aspect_ratio": "1:1",
        "num_outputs": 4
      }
    }
  ],
  "edges": [
    {"id": "cs_e1", "source": "cs_designer", "source_port": "out", "target": "cs_imagegen", "target_port": "in_prompt"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}`),
}

// ---------------------------------------------------------------------------
// 4. Script to Storyboard — 3 nodes, 2 edges
// ---------------------------------------------------------------------------

var scriptToStoryboard = Template{
	ID:          "script-to-storyboard",
	Name:        "Script to Storyboard",
	Description: "Turn a story idea into a script, break it into scenes, and generate storyboard images",
	Category:    "image",
	NodeCount:   3,
	Graph: json.RawMessage(`{
  "nodes": [
    {
      "id": "sb_script",
      "type": "agent",
      "label": "Script Writer",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "text", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are an experienced screenwriter skilled in visual storytelling. You write vivid, concise scripts with clear scene descriptions, dialogue, and stage directions. Focus on conveying emotion and visual impact in every scene.",
          "persona": "Screenwriter",
          "variables": {}
        },
        "task": {
          "task_type": "script",
          "output_type": "text",
          "template": "Write a script based on the following input:\n\n{{input}}\n\nInclude scene headings, action lines, and dialogue. Keep it vivid and cinematic.",
          "output_schema": null
        },
        "temperature": 0.8,
        "max_tokens": 4096
      }
    },
    {
      "id": "sb_scene",
      "type": "agent",
      "label": "Scene Director",
      "position": {"x": 450, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "image_prompt", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are an adaptation specialist who converts narrative stories into sequences of detailed image generation prompts. You identify key visual moments, maintain character consistency, and craft prompts that when generated together tell the story visually.",
          "persona": "Adaptation Specialist",
          "variables": {}
        },
        "task": {
          "task_type": "story_to_prompt",
          "output_type": "image_prompt",
          "template": "Convert the following story into a sequence of image generation prompts:\n\n{{input}}\n\nExtract key visual moments and create detailed, consistent image prompts for each scene. Maintain character appearance and style consistency across all prompts.",
          "output_schema": null
        },
        "temperature": 0.7,
        "max_tokens": 4096
      }
    },
    {
      "id": "sb_imagegen",
      "type": "media",
      "label": "Image Generator",
      "position": {"x": 800, "y": 200},
      "ports": [
        {"id": "in_prompt", "name": "Prompt", "type": "image_prompt", "direction": "input", "required": true},
        {"id": "in_ref", "name": "Reference", "type": "any", "direction": "input", "required": false},
        {"id": "out_media", "name": "Output", "type": "image", "direction": "output", "required": false}
      ],
      "config": {
        "media_type": "image",
        "provider": "replicate",
        "model": "flux-1.1-pro",
        "prompt": "{{input}}",
        "aspect_ratio": "16:9",
        "num_outputs": 1
      }
    }
  ],
  "edges": [
    {"id": "sb_e1", "source": "sb_script", "source_port": "out", "target": "sb_scene", "target_port": "in"},
    {"id": "sb_e2", "source": "sb_scene", "source_port": "out", "target": "sb_imagegen", "target_port": "in_prompt"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}`),
}

// ---------------------------------------------------------------------------
// 5. Prompt Enhancer Chain — 2 nodes, 1 edge
// ---------------------------------------------------------------------------

var promptEnhancerChain = Template{
	ID:          "prompt-enhancer-chain",
	Name:        "Prompt Enhancer",
	Description: "Feed text into a prompt enhancer to get optimized, detailed AI-ready prompts",
	Category:    "text",
	NodeCount:   2,
	Graph: json.RawMessage(`{
  "nodes": [
    {
      "id": "pe_input",
      "type": "tool",
      "label": "Text Input",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "out", "name": "Text", "type": "text", "direction": "output", "required": false}
      ],
      "config": {
        "tool_type": "text_box",
        "content": "A warrior standing on a cliff at sunset, looking out over a vast kingdom"
      }
    },
    {
      "id": "pe_enhancer",
      "type": "agent",
      "label": "Prompt Enhancer",
      "position": {"x": 450, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input", "required": false},
        {"id": "out", "name": "Output", "type": "text", "direction": "output", "required": false}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {
          "system_prompt": "You are a prompt optimizer who takes rough or basic prompts and transforms them into highly detailed, well-structured prompts that produce superior AI outputs. You add specificity, style cues, technical parameters, and quality boosters while preserving the original intent.",
          "persona": "Prompt Optimizer",
          "variables": {}
        },
        "task": {
          "task_type": "prompt_enhancement",
          "output_type": "text",
          "template": "Enhance and optimize the following prompt for better AI output quality:\n\n{{input}}\n\nReturn an improved version with added detail, specificity, style cues, and quality descriptors. Preserve the original intent.",
          "output_schema": null
        },
        "temperature": 0.6,
        "max_tokens": 2048
      }
    }
  ],
  "edges": [
    {"id": "pe_e1", "source": "pe_input", "source_port": "out", "target": "pe_enhancer", "target_port": "in"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}`),
}
