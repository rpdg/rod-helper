export interface Json {
	[key: string]: any;
}

/**
 * Overall configuration structure
 */
export interface IConfig {
	/**
	 * Root node of data configuration, an array that combines data configuration sections and data nodes
	 */
	dataSection: (IDataSection | IValueItem)[];

	/**
	 * Configuration node for branching flow, executes the corresponding branch based on the switch result after the root dataSection is executed
	 */
	switchSection?: ISwitchSection;

	/**
	 * Root node of download configuration, an array of download configuration sections
	 */
	downloadSection?: IDownloadSection[];

	/**
	 * Root folder for saving downloads, note the write permission
	 */
	downloadRoot?: string;

	/**
	 * Node configuration for determining whether the page has finished loading
	 */
	pageLoad?: {
		/**
		 * Page completion flag
		 */
		wait: 'show' | 'hide' | 'wait';

		/**
		 * CSS selector for the completion flag element, used when wait is 'show' or 'hide'
		 */
		selector?: string;

		/**
		 * Wait time in seconds, can be set regardless of wait being 'show', 'hide', or 'wait'
		 */
		sleep?: number;
	};
}

/**
 * Base class for configuration sections and nodes
 */
export interface IConfigNode {
	/**
	 * The id must be unique within the current fragment and will be used as the key for the output result
	 */
	id: string;

	/**
	 * CSS selector
	 */
	selector: string;

	/**
	 * JavaScript function string used to modify the DOM result of the selector, optional.
	 *
	 * Function signature: (dom<Element | Element[] | null>) => Element | Element[] | null
	 */
	domRender?: string;

	/**
	 * Description for readability
	 */
	label: string;
}

/**
 * Data configuration section
 */
export interface IDataSection extends IConfigNode {
	/**
	 * Enumeration
	 */
	sectionType: 'form' | 'list';

	/**
	 * Node group
	 */
	items: IValueItem[];

	/**
	 * JavaScript function string used to filter the result of the List Section, optional.
	 *
	 * Function signature is the same as Array.prototype.filter:
	 * It takes three fixed parameters: val, i, arr, and returns false to remove the list item.
	 */
	filterRender?: string;

	/**
	 * JavaScript function string used to transform the result, optional.
	 *
	 * If both filterRender and dataRender exist in the List Section,
	 * dataRender will be executed after filterRender.
	 *
	 * Example: "return parseInt(val, 10)" can convert a string to a number.
	 *
	 * Function signature: It takes two fixed parameters: val, node, and this refers to the current item.
	 */
	dataRender?: string;
}

/**
 * Data node
 */
export interface IValueItem extends IConfigNode {
	/**
	 * Enumeration
	 */
	itemType: 'text' | 'textBox' | 'radioBox' | 'checkBox' | 'dropBox' | 'download';

	/**
	 * DOM attribute for data value, defaults to innerText
	 */
	valueProper?: string;

	/**
	 * JavaScript function string used to transform the result, optional.
	 *
	 * Example: "return parseInt(val.replace(',', ''), 10)" can convert a string to a number.
	 *
	 * Function signature: It takes two fixed parameters: val, node, and this refers to the current item.
	 */
	valueRender?: string;

	/**
	 * When itemType = download, it associates with the corresponding configuration in downloadSection based on the downloadId value.
	 */
	downloadId?: string;

	/**
	 * Configuration for external link pages, optional.
	 */
	external?: {
		/**
		 * Relative path to the linked configuration file for the current configuration file,
		 * or embedded complete config object.
		 */
		config: string | IConfig;

		/**
		 * Key used for outputting the result, optional. If not set, the id of the IValueItem node will be used.
		 */
		id?: string;
	};
}

/**
 * Configuration node for branching flow
 */
export interface ISwitchSection {
	/**
	 * JavaScript function string used to transform the result, optional.
	 *
	 * Function signature: (this<SwitchSection>, data<IResult>, config<IConfig>) => string | number | boolean | null | undefined
	 *
	 * Example: "return data.barCode.startsWith(('SN')" returns a boolean indicating whether barCode starts with SN
	 */
	switchRender: string;

	/**
	 * Array of case items
	 */
	cases: ICaseItem[];
}

/**
 * Case configuration for branching flow
 */
export interface ICaseItem {
	/**
	 * The case that matches the result will be executed
	 */
	case: string | number | boolean | null | undefined | string[] | number[];

	/**
	 * Array of data sections and value items
	 */
	dataSection: (IDataSection | IValueItem)[];
}

/**
 * Download configuration section
 */
export interface IDownloadSection extends IConfigNode {
	/**
	 * Subdirectory for saving files (relative path to downloadRoot), if empty, the id will be used as the subdirectory name
	 */
	savePath: string;

	/**
	 * JavaScript function string used to filter the result of the List Section, optional.
	 *
	 * Function signature is the same as Array.prototype.filter:
	 * It takes three fixed parameters: val, i, arr, and returns false to remove the list item.
	 */
	// filterRender?: string;

	/**
	 * DOM attribute for the file name, defaults to innerText
	 */
	nameProper?: string;

	/**
	 * JavaScript function string used to transform the result, optional.
	 *
	 * Function signature: (this<DownloadSection>, name<string>, node<HTMLElement>) => string
	 *
	 * Example: "let parts = name.split('/'); return parts[parts.length - 1];"
	 *
	 * If nameRender = "auto", the recommended file name from the download information will be used.
	 */
	nameRender?: string;

	/**
	 * DOM attribute for the link, defaults to innerText
	 */
	linkProper: string;

	/**
	 * JavaScript function string used to transform the result, optional.
	 *
	 * Function signature: (this<DownloadSection>, link<string>, node<HTMLElement>) => string
	 *
	 * Example: "let parts = link.split('/'); return parts[parts.length - 1];"
	 */
	linkRender?: string;

	/**
	 * Enumeration
	 */
	downloadType: 'url' | 'element' | 'toPDF';

	/**
	 * Path to insert into the corresponding Result Data
	 */
	insertTo?: string;
}

/**
 * Result object
 */
export interface IResult {
	/**
	 * Data storage section
	 */
	data: Record<string, any>;

	/**
	 * Download section after execution
	 */
	downloads: Record<string, IDownloadResult>;

	/**
	 * Parsed base path for downloads
	 */
	downloadRoot?: string;

	/**
	 * Parsed external section
	 */
	externalSection?: Record<string, IExternal>;
}

/**
 * File information object
 */
export interface IFileInfo {
	/**
	 * File name
	 */
	name: string;

	/**
	 * If it is a url or toPDF type download, the download link will be stored here
	 */
	url: string;

	/**
	 * Error message, only present if there is an error
	 */
	error: string;
}

/**
 * Download section of IResult
 */
export interface IDownloadResult {
	/**
	 * Corresponding label of IDownloadSection
	 */
	label: string;

	/**
	 * List of downloaded files
	 */
	files: IFileInfo[];
}

/**
 * External section of IResult
 */
export interface IExternal {
	id: string;

	/**
	 * Relative path to the linked configuration file for the current configuration file, path should use \\ or /
	 * or embedded complete config object
	 */
	config: string | IConfig;

	/**
	 * Associated field
	 */
	connect: string;
}