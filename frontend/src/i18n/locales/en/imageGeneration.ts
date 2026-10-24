export default {
  imageGeneration: {
    eyebrow: 'Creative workspace',
    title: 'AI Image Studio',
    description: 'Turn ideas into production-ready images with your existing API keys, model routing, and billing rules.',
    compose: {
      title: 'Create',
      subtitle: 'Choose an image-enabled API key, then describe what you want to make.'
    },
    form: {
      apiKey: 'Image API key',
      noGroup: 'No group',
      model: 'Model',
      modelPlaceholder: 'gpt-image-2',
      noSupportedModels: 'No image models are available for this key group',
      mode: 'Creation mode',
      prompt: 'Prompt',
      promptPlaceholder: 'Describe the subject, style, composition, lighting, colors, and any exact text you need in the image...',
      promptHint: 'Specific prompts usually produce more consistent results.',
      size: 'Canvas size',
      quality: 'Quality',
      count: 'Outputs',
      templates: 'Prompt starters'
    },
    modes: {
      generate: 'Text to image',
      edit: 'Image to image'
    },
    referenceImages: {
      title: 'Reference images',
      action: 'Choose images',
      remove: 'Remove reference image',
      hint: 'PNG, JPG, or WebP. This model accepts up to {count} images, {fileSize} each and {totalSize} total.',
      errors: {
        count: 'This model accepts up to {count} reference images.',
        type: '“{name}” is not supported. Upload PNG, JPG, or WebP.',
        fileSize: '“{name}” exceeds the {size} per-image limit.',
        totalSize: 'Reference images cannot exceed {size} in total.'
      }
    },
    emptyKeys: {
      title: 'No image-enabled API key',
      description: 'Create or update an active OpenAI, Gemini, or Antigravity API key in a group that allows image generation.',
      action: 'Manage API keys'
    },
    sizes: {
      square1k: 'Square 1024 x 1024',
      landscape1_5k: 'Landscape 1536 x 1024',
      portrait1_5k: 'Portrait 1024 x 1536',
      square2k: 'Square 2048 x 2048',
      landscape2k: 'Landscape 2048 x 1152',
      square2880: 'Square 2880 x 2880',
      landscape4k: 'Landscape 3840 x 2160',
      portrait4k: 'Portrait 2160 x 3840'
    },
    customSize: {
      add: 'Custom size...',
      current: 'Current size: {size}',
      width: 'Width',
      height: 'Height',
      hint: 'Saved sizes are added to this list and kept in this browser.',
      optionLabel: 'Custom {size}',
      saved: 'Custom size saved',
      errors: {
        format: 'Enter a valid width and height.',
        maxSide: 'Width and height must both be {max} px or less.',
        multiple: 'Width and height must both be multiples of {step}.',
        ratio: 'The long-side to short-side ratio cannot exceed {ratio}.',
        pixels: 'Total pixels must be between {min} and {max}.'
      }
    },
    sizeRules: {
      title: 'Image size rules',
      summary: 'Preset and custom sizes must satisfy all of these constraints:',
      maxSide: 'The longest side must be 3840 px or less.',
      multiple: 'Both width and height must be multiples of 16.',
      ratio: 'The long-side to short-side ratio cannot exceed 3:1.',
      pixels: 'Total pixels must be between 655,360 and 8,294,400.',
      presets: 'All sizes in this list already meet these rules.'
    },
    pricing: {
      title: 'Estimated price',
      tier: '{tier} tier',
      noKey: 'Choose an image-enabled API key to preview the estimated charge.',
      configuredDescription: 'Estimated from the current group image price. Outputs: {count}.',
      defaultDescription: 'This group has no configured price for this size tier. The final charge will use the server default model price.',
      multiplier: 'Multiplier',
      unitPrice: 'Per image',
      estimatedTotal: 'Estimated total',
      serverDefault: 'Server default',
      serverCalculated: 'Server calculated'
    },
    qualities: {
      high: 'High',
      medium: 'Medium',
      low: 'Low',
      auto: 'Auto'
    },
    templates: {
      typography: {
        title: 'Precision type poster',
        description: 'Tests complex layout, exact text, and material rendering.',
        prompt: 'Design a vertical key-art poster for an experimental film festival. Center a futuristic theater made of translucent red glass in a neon-lit rainy night, with cinematic volumetric light and physically accurate refraction. Typeset exactly: headline “BEYOND THE FRAME”, subtitle “Experimental Moving Image Season”, date “OCT 18—27”, venue “WEST BUND ART CENTER”. Use a rigorous Swiss grid and bold modern grotesk type. Every character must be legible and exact, with no extra text. Make it worthy of an international design award.'
      },
      characterSheet: {
        title: 'Consistent character sheet',
        description: 'Keeps one character stable across views and expressions.',
        prompt: 'Create a professional animation character sheet for an original character named Lan: a 25-year-old Asian woman mechanical engineer with short silver-gray hair, a thin scar through her left eyebrow, a forest-green utility jacket, and an orange circular pin on her chest. Include front, side, and back full-body views, six clear facial expressions, and one dynamic pose holding a toolbox. Keep facial structure, hairstyle, garment construction, pin placement, and body proportions perfectly consistent in every panel. Light gray studio background, precise concept-art finish.'
      },
      cinematicProduct: {
        title: 'Cinematic product ad',
        description: 'Tests lifelike materials, macro detail, and commercial light.',
        prompt: 'Create a cinematic landscape advertisement for an unbranded precision mechanical watch. The watch floats above rain-wet black volcanic rock. Use brushed titanium, controlled highlights on the sapphire crystal, and clearly resolved gears and indices. Cold dawn mist flows between the rocks, with warm gold rim light from the rear left and soft cool-blue fill from the right. Emulate a 100mm macro lens, shallow depth of field, physically realistic materials, and high dynamic range. Leave clean negative space on the right. No logos, gibberish, or extra components.'
      },
      technicalDiagram: {
        title: 'Technical explainer',
        description: 'Tests spatial reasoning, causal structure, and labels.',
        prompt: 'Draw a landscape museum-grade explainer titled “HOW OFFSHORE WIND POWER WORKS”. On the left, show a cutaway offshore turbine; in the center, connect blades, gearbox, generator, and subsea cable with unambiguous arrows; on the right, show electricity entering the city grid. Use restrained teal, blue, and safety yellow, isometric technical illustration, one line-weight system, and numbered stages. Labels must be limited to “BLADES, GEARBOX, GENERATOR, SUBSEA CABLE, SUBSTATION, CITY GRID”, all perfectly legible and causally correct.'
      },
      productComposite: {
        title: 'Multi-image product composite',
        description: 'Combines products, accessories, and a scene as one shoot.',
        prompt: 'Use the product in reference image 1 as the absolute hero. Preserve its shape, colors, logo, label text, and material details exactly. Take accessories and environmental elements from reference images 2 and 3 and combine them into one premium, photorealistic campaign image. Unify perspective, light direction, shadow softness, and color temperature so everything looks photographed together. Repair cutout edges and reflections, but do not redesign the product or alter any packaging text.'
      },
      characterEdit: {
        title: 'Identity-preserving scene edit',
        description: 'Keeps the person while changing wardrobe, pose, and place.',
        prompt: 'Strictly preserve the referenced person’s identity: face shape, feature proportions, skin tone, hairstyle, and apparent age must not change. Place them on a rainy Tokyo street in the 1960s, wearing a period-accurate tailored trench coat and holding a transparent umbrella while looking back at the camera beneath neon signs. Use a 35mm cinema-film look, natural skin texture, realistic rain, and wet-street reflections. Do not beautify them into a different person or alter facial structure.'
      },
      sketchRender: {
        title: 'Sketch to finished design',
        description: 'Turns line structure into a plausible manufactured object.',
        prompt: 'Transform the reference sketch into a manufacturable premium desktop speaker render. Follow the sketch’s silhouette, button positions, opening proportions, and structural relationships exactly; do not invent extra parts. Use an anodized aluminum body, charcoal woven grille, and matte control knob. Show a three-quarter hero view, with a faint engineering side-profile drawing in the background. Real studio lighting, industrial-design portfolio quality.'
      },
      multiImage: {
        title: 'Multi-reference world blend',
        description: 'Tests subject understanding, relationships, and style unity.',
        prompt: 'Combine the main subjects from every reference image into one coherent scene: they are preparing dinner together inside a glass greenhouse floating above a sea of clouds. Preserve each subject’s recognizable appearance, colors, textures, and relative proportions, and give them natural interactions instead of lining them up. Use a photoreal cinematic look, golden-hour backlight, accurate glass refraction, plant shadows, and contact shadows on the table. Build clear foreground, middle ground, and background. Do not duplicate or omit any subject.'
      }
    },
    actions: {
      generating: 'Generating...',
      generate: 'Generate',
      downloadAll: 'Download all',
      download: 'Download'
    },
    preview: {
      title: 'Preview',
      generating: 'Request submitted. Waiting for the server response. {seconds}s elapsed.',
      readyOne: '1 image ready',
      readyMany: '{count} images ready',
      idle: 'Generated images will appear here.',
      emptyTitle: 'Ready when you are',
      emptyDescription: 'Write a prompt or start from a template, then generate images through your active API key.',
      runningTitle: 'Generating images',
      runningDescription: 'The request has been sent to the gateway. Image generation can take a little while; results will appear here automatically.',
      elapsed: 'Elapsed',
      elapsedSeconds: '{seconds}s',
      requestId: 'Generation request ID',
      logHint: 'Copy this ID into the ops log client_request_id filter to confirm the request reached the backend.',
      imageAlt: 'Generated image {number}',
      imageNumber: 'Image {number}',
      fullscreenAlt: 'Full-screen generated image'
    },
    history: {
      title: 'Recent generations',
      subtitle: 'We do not store your images. They are saved locally in this browser only; clearing site data removes them. Up to 8 recent images are kept.',
      clear: 'Clear',
      empty: 'No recent images yet.'
    },
    errors: {
      loadKeysFailed: 'Failed to load API keys',
      noImages: 'The gateway returned no images.',
      partialImages: 'Generated {count}/{requested} images. The remaining images were not returned.',
      generateFailed: 'Image generation failed'
    },
    messages: {
      generated: 'Image generation complete',
      requestIdCopied: 'Generation request ID copied'
    }
  }
}
