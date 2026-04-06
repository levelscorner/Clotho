# Clotho

## Requirement

I create content via openart and higgsfield as they are two best options, then there is weavy.ai but i did not buy its subscription.
I work with with openart and higgsfield for video creations,  I use nanobanan via gemini, chatgpt,etc for image creation and gemini, claude, chatgpt,etc for script writing.
I even tried creating gemsn in gemini but its tedious to work with and hinders the creativiy, also them creating a new chat and they all haave to be opened in new windows. SOmething like weavy in terms of workflow is what i want.
Example: There is an amazing application called <https://app.weavy.ai/> this has an amazing workflow but the subscription is abysmall. There is alos n8n for the same but at larger scale and no-code production automation.A good reference for workflow automation is <https://n8n.io/>.
So i want to create my own weavy/n8n like application,we can start small, and few variations.

### AI/LLM

I want to be able to use the LLM in the components form that we can drag and drop a text box kind of widget lile i weavy/n8n and use that to create my own script and prompt for video and images. Agent is a box to keep the AI model, It can LLM, or video, audio or any other model.

#### Components for our APP

**Objective**: I want to be able to use [agent] with a [role] to perform a [task]

[Agent] - Agent is a LLM/companion
[Role] - Role is a job or personality of the agent
[Task] - Task is a job to be done by the agent,Job can be creating a script, prompt, image, video, audio, etc.

use this a initial component that we make, and then we can add more components as we go on, as following.

[ScriptPromptAgent]     :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (script or prompt)
[ImagePromptAgent]      :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (image prompt)
[VideoPromptAgent]      :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (video prompt)
[CharacterPromptAgent]  :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (character prompt)
[PromptEnhacerAgent]    :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (prompt enhancer)
[StoryAgent]            :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (story)
[StoryToPromptAgent]    :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (story to prompt)

Some of these could be redundant, we can remove them later, like Story agent just and agent with (personality or job or a) [role] to create a (story) then [StoryAgent] is redundant.
Also [StoryToPromptAgent] is just [ScriptPromptAgent] + [StoryAgent] and [StoryAgent] is just a [Agent] with a [role] to create a (story).
So at some level we need to go atomic and create a base component [Agent] which can be used to create other components.

##### Roles

Role is a personality or job or a task that Agent has to take/perform.
This should be a property of component which is injectable in Agent.

##### Properties of Component

- Drag and droppable
- Connectable to other components
- Deletable
- Editable

##### Inputs and Outputs

- Each component should have inputs and outputs, which can be connected to other components.
- Only correct type of inputs should be allowed to be connected to a component.

#### Pipeline

Pipeline is a graph of components. It can be linear or non-linear.

#### Projects

Project is a collection of pipelines.

##### Properties

Projects should be managable. Hence it will be

- Editable
- Deletable
- Shareable(Eventually, not in MVP)

##### Tools

There are tools which can be simply coded to create what we want and these are simple and may not need AI/LLMs/Models. This category are tools for Image, Text, Audio and Video.
[TextBox]               :       I want to be able to use a text box to create a text
[ImageBox]              :       I want to be able to use an image box to use and plug it to next component which might use image to perform a [task].
[VideoBox]              :       I want to be able to use an video box to use and plug it to next component which might use video to perform a [task].

##### Future Enhancements

When we have local models we can do this. But if we can use API to create images or video we should use it for now too.
[ImageGeneratorAgent]   :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (image)
[VideoGeneratorAgent]   :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (video)
[AudioBox]              :       I want to be able to use an audio box to use and plug it to next component which might use audio to perform a [task].
[AudioGeneratorAgent]   :       I want to be able to use an [agent] with a (personality or job or a) [role] to create a (audio)

###### MiscroSaas

[ImageEditor]           :       I want to be able to use an image editor to edit an image.
[ObjectRemover]         :       A small AI widget which can remove objects from image or given a prompt remove an object from image.

## UserInterface

Application [Clotho] should have a user interface where we can use [Agent]s in our [workspace]. [Workspace] is where all our [Components] reside. [Workspace] is part of [Project]. [Project] is part of [User]. Hence we would eventually be needing a user management package, project management package and component management package.

Components inside workspace should be drag and drop and we should be able to connect them to each other to create a [pipeline].

## Future enhancements

I will be open to working by downloading the models locally and running them locally, later once we have the basic setup done. I dont have the best hardware or online services like AWS/GCP/Azure etc for it, but i can upgrade it later.

- This Clotho app needs to have a language which can be used to create components and pipelines. This language should be simple and easy to use. It should be a declarative language, where we can define the components and pipelines in a declarative way. All the way upto Project.
- This will make is sharable easily, we can create a marketplace for components and pipelines.
- I want to use the open source models and run them locally, in future i may use it for commercial purposes, so i want to be careful about the licensing and all.
 I will be open to working by downloading the models locally and running them locally, later once we have the basic setup done. I dont have the best hardware or online services like AWS/GCP/Azure etc for it, but i can upgrade it later.
