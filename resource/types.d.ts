/**
 * 配置整体结构
 */
export interface IConfig {
	/**
	 * 数据配置的根节点，是数据配置段与数据节点的联合数组
	 */
	dataSection: (IDataSection | IValueItem)[];

	/**
	 * 分支流程的配置节点，将在根dataSection执行完后，根据switch的结果执行对应分支
	 */
	switchSection?: ISwitchSection;

	/**
	 * 下载配置的根节点，下载配置段的数组
	 */
	downloadSection?: IDownloadSection[];

	/**
	 * 下载保存的根文件夹，注意可写权限
	 */
	downloadRoot?: string;

	/**
	 * 判断页面加载完成与否的节点配置
	 */
	pageLoad?: {
		/**
		 * 页面结束标志
		 */
		wait: 'show' | 'hide' | 'wait';

		/**
		 * 页面结束标志元素 css selector，当 wait 为show，hide时候，由此查找标志物
		 */
		selector?: string;

		/**
		 * 等待时间秒，无论 wait 为show，hide，wait，皆可设置等待
		 */
		sleep?: number;
	};
}

/**
 * 配置段与节点的基类
 */
export interface IConfigNode {
	/**
	 * id必须在当前片段内唯一，将作为结果的 key 输出
	 */
	id: string;

	/**
	 * css selector
	 */
	selector: string;

	/**
	 * 中文描述，供阅读方便用
	 */
	label: string;
}

/**
 * 数据配置段
 */
export interface IDataSection extends IConfigNode {
	/**
	 * 枚举
	 */
	sectionType: 'form' | 'list';

	/**
	 * 节点组
	 */
	items: IValueItem[];

	/***
	 * Javascript function String，用来对List Section 的结果进行过滤，可选。
	 *
	 * 函数签名同 array.proto.filter：
	 * 固定传入 val, i, arr 三个参数，返回false的list item将被去除
	 */
	filterRender?: string;


	/**
	 * Javascript function String，用来对结果进行转换，可选。
	 *
	 * 如果在List Section中同时有 filterRender & dataRender,
	 * 那么 dataRender 将在 filterRender 之后执行
	 *
	 * 例： "return parseInt(val, 10)" 即可将string转为number，
	 *
	 * 函数签名：固定传入 val, node 两个参数，this 指向当前的item
	 */
	dataRender?: string;
}

/**
 * 数据节点
 */
export interface IValueItem extends IConfigNode {
	/**
	 * 枚举
	 */
	itemType: 'text' | 'textBox' | 'radioBox' | 'checkBox' | 'dropBox';

	/**
	 * 数据取值的dom属性，默认取 innerText
	 */
	valueProper?: string;

	/**
	 * Javascript function String，用来对结果进行转换，可选。
	 *
	 * 例： "return parseInt(val.replace(',', ''), 10)" 即可将string转为number，
	 *
	 * 函数签名：固定传入 val, node 两个参数，this 指向当前的item
	 */
	valueRender?: string;

	/**
	 * 外链页面的配置，可选。
	 */
	external?: {
		/**
		 * 链接的配置文件对于当前配置文件的相对路径，
		 * 或者嵌入完整的 config对象
		 */
		config: string | IConfig;

		/**
		 * 将作为结果的 key 输出，可选，如果不设则会使用所在IValueItem节点的id
		 */
		id?: string;
	};
}

/**
 * 分支流程的配置节点
 */
export interface ISwitchSection {
	/**
	 * Javascript function String，用来对结果进行转换，可选。
	 *
	 * 函数签名：(this<SwitchSection>, data<IResult>, config<IConfig>) => string | number | bool | null | undefined
	 *
	 * 例： "return data.barCode.startsWith(('SN')" 即返回barCode是否以SN开头的bool
	 */
	switchRender: string;
	cases: ICaseItem[];
}

/**
 * 分支流程的Case配置
 */
export interface ICaseItem {
	/**
	 * 匹配结果的case将会执行
	 */
	case: string | number | boolean | null | undefined | string[] | number[];
	dataSection: (IDataSection | IValueItem)[];
}

/**
 * 下载配置段
 */
export interface IDownloadSection extends IConfigNode {
	/**
	 * 保存文件夹子目录（downloadRoot的相对路径），如果空则取 id 为子目录名
	 */
	savePath: string;

	/**
	 * 文件名对应的dom属性，默认取 innerText
	 */
	nameProper?: string;

	/**
	 * Javascript function String，用来对结果进行转换，可选。
	 *
	 * 函数签名：(this<DownloadSection>, name<string>, node<HTMLElement>) => string
	 *
	 * 例："let parts = name.split('/'); return parts[parts.length - 1];"
	 */
	nameRender?: string;

	/**
	 * 链接对应的dom属性，默认取 innerText
	 */
	linkProper: string;
	/**
	 * Javascript function String，用来对结果进行转换，可选。
	 *
	 * 函数签名：(this<DownloadSection>, link<string>, node<HTMLElement>) => string
	 *
	 * 例："let parts = link.split('/'); return parts[parts.length - 1];"
	 */
	linkRender?: string;

	/**
	 * '枚举，与阿里RPA下载模式对应
	 */
	type: 'url' | 'element' | 'toPDF';
}

/**
 * 返回结果
 */
export interface IResult {
	/**
	 * 数据存放段
	 */
	data: Record<string, any>;

	/**
	 * 执行后的下载段
	 */
	downloads?: Record<string, IDownloadResult>;

	/**
	 * 解析后的下载基础路径
	 */
	downloadRoot?: string;

	/**
	 * 解析后的外链段
	 */
	externalSection?: Record<string, IExternal>;
}

/**
 * IResult的下载段
 */
export interface IDownloadResult {
	/**
	 * 即 IDownloadSection 的对应 label
	 */
	label : string;
	/**
	 * 下载数量
	 */
	count: number;
	/**
	 * 文件名列表
	 */
	fileNames: string[];
	/**
	 * 若是url类型的下载，则会在此处存放下载链接
	 */
	links: string[];
	/**
	 * 出错index
	 */
	errors: number[];
}

/**
 * IResult的外链段
 */
export interface IExternal {
	id: string;

	/**
	 * 链接的配置文件对于当前配置文件的相对路径，路径注意用 \\ 或 /
	 * 或者嵌入完整的 config对象
	 */
	config: string | IConfig;

	/**
	 * 关联字段
	 */
	connect: string;
}
