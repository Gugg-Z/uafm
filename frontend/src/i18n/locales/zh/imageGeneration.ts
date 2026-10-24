export default {
  imageGeneration: {
    eyebrow: '创作工作台',
    title: 'AI 生图工坊',
    description: '使用现有 API 密钥、模型路由和计费规则，把想法直接生成可用图片。',
    compose: {
      title: '开始创作',
      subtitle: '选择已开启生图权限的 API 密钥，再描述你想生成的画面。'
    },
    form: {
      apiKey: '生图 API 密钥',
      noGroup: '未分组',
      model: '模型',
      modelPlaceholder: 'gpt-image-2',
      noSupportedModels: '当前密钥分组暂无可用生图模型',
      mode: '创作方式',
      prompt: '提示词',
      promptPlaceholder: '描述主体、风格、构图、光线、配色，以及画面里必须出现的文字...',
      promptHint: '提示词越具体，生成结果通常越稳定。',
      size: '画布尺寸',
      quality: '质量',
      count: '生成数量',
      templates: '提示词模板'
    },
    modes: {
      generate: '文生图',
      edit: '图生图'
    },
    referenceImages: {
      title: '参考图片',
      action: '选择图片',
      remove: '移除参考图片',
      hint: '支持 PNG、JPG、WebP；当前模型最多 {count} 张，单张不超过 {fileSize}，合计不超过 {totalSize}。',
      errors: {
        count: '当前模型最多支持 {count} 张参考图片。',
        type: '“{name}”格式不支持，请上传 PNG、JPG 或 WebP。',
        fileSize: '“{name}”超过单张 {size} 的限制。',
        totalSize: '参考图片合计不能超过 {size}。'
      }
    },
    emptyKeys: {
      title: '暂无可生图的 API 密钥',
      description: '请创建或更新一个启用中的 OpenAI、Gemini 或 Antigravity API 密钥，并确保所在分组已允许生图。',
      action: '管理 API 密钥'
    },
    sizes: {
      square1k: '方图 1024 x 1024',
      landscape1_5k: '横图 1536 x 1024',
      portrait1_5k: '竖图 1024 x 1536',
      square2k: '方图 2048 x 2048',
      landscape2k: '横图 2048 x 1152',
      square2880: '方图 2880 x 2880',
      landscape4k: '横图 3840 x 2160',
      portrait4k: '竖图 2160 x 3840'
    },
    customSize: {
      add: '自定义尺寸...',
      current: '当前尺寸：{size}',
      width: '宽度',
      height: '高度',
      hint: '保存后会加入下拉列表，并在当前浏览器保留。',
      optionLabel: '自定义 {size}',
      saved: '自定义尺寸已保存',
      errors: {
        format: '请输入有效的宽度和高度。',
        maxSide: '宽和高都不能超过 {max} px。',
        multiple: '宽和高都必须是 {step} 的倍数。',
        ratio: '长边与短边的比例不能超过 {ratio}。',
        pixels: '总像素需在 {min} 到 {max} 之间。'
      }
    },
    sizeRules: {
      title: '生图尺寸规则',
      summary: '预设或自定义尺寸需要同时满足这些条件：',
      maxSide: '最大边长不能超过 3840 px。',
      multiple: '宽和高都必须是 16 的倍数。',
      ratio: '长边与短边的比例不能超过 3:1。',
      pixels: '总像素需在 655,360 到 8,294,400 之间。',
      presets: '当前下拉列表中的尺寸都已经符合这些规则。'
    },
    pricing: {
      title: '预计价格',
      tier: '{tier} 档',
      noKey: '选择可生图 API 密钥后展示预计扣费。',
      configuredDescription: '按当前分组配置价格预估，数量：{count}。',
      defaultDescription: '当前分组未配置该尺寸档位价格，实际扣费将由服务端默认模型价格计算。',
      multiplier: '倍率',
      unitPrice: '单张扣费',
      estimatedTotal: '预计总计',
      serverDefault: '服务端默认',
      serverCalculated: '服务端计算'
    },
    qualities: {
      high: '高质量',
      medium: '中等',
      low: '快速',
      auto: '自动'
    },
    templates: {
      typography: {
        title: '中文字体海报',
        description: '测试复杂排版、准确文字与材质表现。',
        prompt: '设计一张竖版电影节主视觉海报。画面中心是一座由半透明红色玻璃构成的未来剧院，雨夜霓虹倒影，电影级体积光与真实折射。必须清晰、准确地排版以下中文：主标题“光影之外”，副标题“未来影像实验季”，日期“10.18—10.27”，地点“上海·西岸艺术中心”。使用瑞士国际主义网格，主标题为醒目的现代黑体，所有文字不可乱码、不可增删，整体达到国际设计奖参赛作品水准。'
      },
      characterSheet: {
        title: '角色一致性设定',
        description: '同一角色跨视角、表情与动作保持一致。',
        prompt: '创建一张专业动画角色设定表：原创角色“岚”，25 岁亚洲女性机械工程师，银灰色短发，左眉有一道细疤，穿墨绿色功能夹克，胸前别着橙色圆形徽章。画面包含正面、侧面、背面全身视图，六种清晰表情，以及手持工具箱的动态姿势。每个分格必须保持五官、发型、服装结构、徽章位置和身体比例完全一致；纯浅灰背景，工作室概念设计稿风格，细节精确。'
      },
      cinematicProduct: {
        title: '电影级产品广告',
        description: '测试真实材质、微距细节和商业灯光。',
        prompt: '为一块无品牌的精密机械腕表制作电影级横版广告。腕表悬浮在被雨水打湿的黑色火山岩上，表壳为拉丝钛金属，蓝宝石镜面出现受控高光，表盘齿轮与刻度清晰可见；清晨冷雾从岩石间流过，左后方暖金色轮廓光与右侧冷蓝柔光形成对比。使用 100mm 微距镜头质感、浅景深、真实物理材质与高动态范围，右侧保留干净留白，禁止出现品牌、乱码或多余零件。'
      },
      technicalDiagram: {
        title: '精密科普信息图',
        description: '测试空间推理、结构关系与标签清晰度。',
        prompt: '绘制一张横版“海上风力发电如何工作”的博物馆级科普信息图。左侧展示海上风机的剖面结构，中间用清晰箭头连接叶片、齿轮箱、发电机、海底电缆，右侧展示电力进入城市电网的流程。使用蓝绿与亮黄色的克制配色、等距技术插画、统一线宽和编号系统；中文标签仅包含“叶片、齿轮箱、发电机、海底电缆、变电站、城市电网”，文字必须清晰准确，因果方向不能出错。'
      },
      productComposite: {
        title: '多图商品合成',
        description: '把商品、配件和场景融合成统一广告画面。',
        prompt: '以第 1 张图的商品为绝对主体，完整保留它的造型、颜色、Logo、标签文字和材质细节；从第 2、3 张参考图中提取配件与环境元素，合成为一张真实的高端商业广告。统一透视、光源方向、阴影软硬和色温，让所有物体像在同一次摄影中拍摄；修复抠图边缘与反射关系，但不要重新设计商品，不得改变任何包装文字。'
      },
      characterEdit: {
        title: '人物跨场景一致性',
        description: '保留身份特征，只改变服装、动作和环境。',
        prompt: '严格保留参考人物的身份特征：脸型、五官比例、肤色、发型和年龄感都不能改变。将人物置于 1960 年代东京雨夜街头，穿剪裁准确的复古风衣，手持透明雨伞，从霓虹招牌前回头看向镜头。采用 35mm 电影胶片质感、自然皮肤纹理、真实雨滴与路面反光；不要美化成不同的人，不要改变面部结构。'
      },
      sketchRender: {
        title: '草图转成品',
        description: '把线稿结构转成可落地的真实设计。',
        prompt: '把参考草图转换为可制造的高端桌面音箱产品渲染。严格遵循草图中的轮廓、按键位置、开孔比例和结构关系，不增加不存在的部件；材质改为阳极氧化铝机身、深灰织物网罩和磨砂旋钮。输出一张三分之四视角主图，并在背景中以浅色工程线稿展示侧视轮廓，真实棚拍灯光，工业设计作品集水准。'
      },
      multiImage: {
        title: '多参考图世界融合',
        description: '验证多主体理解、关系推理与风格统一。',
        prompt: '将所有参考图中的主要对象组合进同一个连贯场景：它们正在一座漂浮于云海之上的玻璃温室中共同准备晚餐。每个对象都必须保留可识别的外观、颜色、纹理和比例特征，并安排自然互动而不是简单并排。统一为写实电影画面，黄金时刻逆光，玻璃折射、植物阴影与桌面接触影准确，构图有明确前中后景，不复制对象，不遗漏主体。'
      }
    },
    actions: {
      generating: '生成中...',
      generate: '生成图片',
      downloadAll: '全部下载',
      download: '下载'
    },
    preview: {
      title: '预览',
      generating: '请求已提交，正在等待服务端返回，已等待 {seconds} 秒。',
      readyOne: '已生成 1 张图片',
      readyMany: '已生成 {count} 张图片',
      idle: '生成结果会显示在这里。',
      emptyTitle: '准备好了就开始',
      emptyDescription: '输入提示词，或从模板开始，通过你的启用中 API 密钥生成图片。',
      runningTitle: '正在生成图片',
      runningDescription: '请求已经发到网关，图片生成通常需要一点时间。完成后结果会自动显示在这里。',
      elapsed: '等待时间',
      elapsedSeconds: '{seconds} 秒',
      requestId: '生成请求 ID',
      logHint: '可复制这个 ID 到运营日志的 client_request_id 里搜索，确认请求是否进入后端。',
      imageAlt: '生成图片 {number}',
      imageNumber: '图片 {number}',
      fullscreenAlt: '全屏生成图片'
    },
    history: {
      title: '最近生成',
      subtitle: '我们不存储图片，仅保存在当前浏览器本地；清理缓存会丢失，最多保留最近 8 张。',
      clear: '清空',
      empty: '还没有最近生成记录。'
    },
    errors: {
      loadKeysFailed: '加载 API 密钥失败',
      noImages: '网关没有返回图片。',
      partialImages: '已生成 {count}/{requested} 张图片，剩余图片未返回。',
      generateFailed: '生图失败'
    },
    messages: {
      generated: '图片生成完成',
      requestIdCopied: '生成请求 ID 已复制'
    }
  }
}
