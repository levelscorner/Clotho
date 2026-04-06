-- Seed default tenant, user, and built-in agent presets

INSERT INTO tenants (id, name) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Default');

INSERT INTO users (id, tenant_id, email, name) VALUES
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'dev@clotho.local', 'Developer');

-- 7 built-in agent presets
INSERT INTO agent_presets (id, tenant_id, name, description, category, config, icon, is_built_in) VALUES

-- 1. Script Writer
('00000000-0000-0000-0000-000000000101',
 NULL,
 'Script Writer',
 'Generates screenplays and scripts from story ideas and outlines',
 'writing',
 '{
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
 }',
 'pen-tool',
 true),

-- 2. Image Prompt Crafter
('00000000-0000-0000-0000-000000000102',
 NULL,
 'Image Prompt Crafter',
 'Creates detailed image generation prompts optimized for AI art models',
 'prompt',
 '{
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
 }',
 'image',
 true),

-- 3. Video Prompt Writer
('00000000-0000-0000-0000-000000000103',
 NULL,
 'Video Prompt Writer',
 'Creates prompts optimized for AI video generation models',
 'prompt',
 '{
   "provider": "openai",
   "model": "gpt-4o",
   "role": {
     "system_prompt": "You are a visual director specializing in AI video generation. You craft detailed video prompts that describe motion, camera work, transitions, and temporal flow. You understand how to convey dynamic scenes for models like Runway, Pika, and Sora.",
     "persona": "Visual Director",
     "variables": {}
   },
   "task": {
     "task_type": "video_prompt",
     "output_type": "video_prompt",
     "template": "Create a detailed video generation prompt based on the following description:\n\n{{input}}\n\nInclude camera movement, subject motion, timing, transitions, and visual style. Format as a structured video prompt.",
     "output_schema": null
   },
   "temperature": 0.7,
   "max_tokens": 1024
 }',
 'video',
 true),

-- 4. Character Designer
('00000000-0000-0000-0000-000000000104',
 NULL,
 'Character Designer',
 'Designs detailed character profiles with visual descriptions for consistent representation',
 'creative',
 '{
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
 }',
 'user',
 true),

-- 5. Prompt Enhancer
('00000000-0000-0000-0000-000000000105',
 NULL,
 'Prompt Enhancer',
 'Refines and enhances existing prompts for better AI output quality',
 'prompt',
 '{
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
 }',
 'sparkles',
 true),

-- 6. Story Writer
('00000000-0000-0000-0000-000000000106',
 NULL,
 'Story Writer',
 'Creates narrative stories with rich descriptions and engaging plot structure',
 'writing',
 '{
   "provider": "openai",
   "model": "gpt-4o",
   "role": {
     "system_prompt": "You are a story narrator who crafts engaging narratives with vivid descriptions, compelling characters, and well-paced plot structure. You write stories that are visual and cinematic, lending themselves naturally to visual adaptation.",
     "persona": "Story Narrator",
     "variables": {}
   },
   "task": {
     "task_type": "story",
     "output_type": "text",
     "template": "Write a story based on the following concept:\n\n{{input}}\n\nCreate a narrative with vivid scene descriptions, character development, and a compelling arc. Make it visually rich for potential adaptation.",
     "output_schema": null
   },
   "temperature": 0.9,
   "max_tokens": 4096
 }',
 'book-open',
 true),

-- 7. Story-to-Prompt
('00000000-0000-0000-0000-000000000107',
 NULL,
 'Story-to-Prompt',
 'Converts narrative text into a sequence of image generation prompts',
 'adaptation',
 '{
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
 }',
 'layers',
 true);
