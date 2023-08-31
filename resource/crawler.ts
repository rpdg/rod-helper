type IFileInfo = import('./types').IFileInfo;
type IDataSection = import('./types').IDataSection;
type IConfig = import('./types').IConfig;
type IExternal = import('./types').IExternal;
type IResult = import('./types').IResult;
type IValueItem = import('./types').IValueItem;
type IDownloadSection = import('./types').IDownloadSection;

function assignDeep(
	target: any,
	...sources: {
		[x: string]: any;
	}[]
) {
	if (target == null) {
		throw new TypeError('Cannot convert undefined or null to object');
	}

	let result = Object(target);

	if (!result['__hash__']) {
		result['__hash__'] = new WeakMap();
	}
	let hash = result['__hash__'];

	sources.forEach((v) => {
		let source = Object(v);
		Reflect.ownKeys(source).forEach((key) => {
			if (!Object.getOwnPropertyDescriptor(source, key)!.enumerable) {
				return;
			}
			if (typeof source[key] === 'object' && source[key] !== null) {
				let isPropertyDone = false;
				if (
					!result[key] ||
					!(typeof result[key] === 'object') ||
					Array.isArray(result[key]) !== Array.isArray(source[key])
				) {
					if (hash.get(source[key])) {
						result[key] = hash.get(source[key]);
						isPropertyDone = true;
					} else {
						result[key] = Array.isArray(source[key]) ? [] : {};
						hash.set(source[key], result[key]);
					}
				}
				if (!isPropertyDone) {
					result[key]['__hash__'] = hash;
					assignDeep(result[key], source[key]);
				}
			} else {
				Object.assign(result, { [key]: source[key] });
			}
		});
	});

	delete result['__hash__'];
	return result;
}

// function findBracketSubstring(str: string): string {
// 	if (!str || !str.includes('(')) return '';
// 	let stack = [];
// 	for (let i = 0; i < str.length; i++) {
// 		if (str[i] === '(') {
// 			stack.push(i);
// 		} else if (str[i] === ')') {
// 			let leftIndex = stack.pop()!;
// 			if (stack.length === 0) {
// 				return str.substring(leftIndex + 1, i);
// 			}
// 		}
// 	}
// 	throw new Error('Unbalanced brackets');
// }

function replacePseudo(
	selector: string,
	parentElement: Element | Document | ShadowRoot = document
): { doc: Element | Document | ShadowRoot; selector: string; ctxChanged: boolean } {
	let doc = parentElement;
	let ctxChanged = false;
	let pseudoMatch = selector.match(/^:(frame|shadow)\((.+?)\)/);
	if (pseudoMatch) {
		let pseudoType = pseudoMatch[1];
		let pseudoSelector = pseudoMatch[2];
		let pseudoElem = parentElement.querySelector(pseudoSelector);
		if (pseudoElem) {
			doc =
				pseudoType === 'frame'
					? (pseudoElem as HTMLIFrameElement).contentWindow!.document
					: (pseudoElem as HTMLSlotElement).shadowRoot!;
			selector = selector.slice(pseudoMatch[0].length).trim();
			ctxChanged = true;
		}
	}
	if (/^:(frame|shadow)\(/.test(selector)) {
		return replacePseudo(selector, doc);
	}
	return { doc, selector, ctxChanged };
}

function queryElem(
	selectorString: string,
	parentElement: Element | Document | ShadowRoot = document,
	domRender?: string
): Element | null {
	let secNode: Element | null = null;
	if (!selectorString) {
		secNode = parentElement as Element;
	} else {
		let { doc, selector } = replacePseudo(selectorString, parentElement);
		secNode = doc.querySelector(selector);
	}

	if (domRender) {
		let rFn = new Function('dom', domRender);
		secNode = rFn.call(window, secNode);
	}
	return secNode;
}

function queryElems(
	selectorString: string,
	parentElement: Element | Document | ShadowRoot = document,
	domRender?: string
): Element[] {
	let secNodes: Element[] = [];
	if (!selectorString) {
		secNodes = [parentElement as Element];
	} else {
		let { doc, selector } = replacePseudo(selectorString, parentElement);
		secNodes = Array.from(doc.querySelectorAll(selector));
	}

	if (domRender) {
		let rFn = new Function('dom', domRender);
		secNodes = rFn.call(window, secNodes);
	}
	return secNodes;
}

const externalDict: Record<string, IExternal> = {};

function appendExternalSection(extObj: IExternal) {
	let key = extObj.connect;
	if (!externalDict[key]) {
		externalDict[key] = extObj;
	}
}

function crawlList(
	sectionId: string,
	sectionElements: Element[],
	items: (IValueItem | IDataSection)[],
	cncPath: string
): any[] {
	let dataArray: any[] = [];
	let renders: Record<string, Function> = {};

	sectionElements.forEach((element) => {
		let data: any = {};
		items.forEach((item) => {
			if ('itemType' in item) {
				let { result, node } = crawItem(item, element);
				data[item.id] = result;

				if (item.valueRender) {
					try {
						if (!renders[item.id]) {
							renders[item.id] = new Function('val, node', item.valueRender);
						}
						let render = renders[item.id];
						let val = data[item.id];
						let res = render.call(item, val, node);
						if (res !== undefined) {
							data[item.id] = res;
						}
					} catch (err: any) {
						console.error('[' + item.id + '.valueRender]', err);
						data[item.id] = `err(${err.message})`;
					}
				}

				if (item.external) {
					let { id, config } = item.external;
					appendExternalSection({
						id: id || item.id,
						config,
						connect: `${cncPath}/${sectionId}/${item.id}`,
					});
				}
			} else if ('sectionType' in item) {
				let { result } = crawSection(item, element, cncPath + '/' + sectionId);
				data[item.id] = result;
			}
		});
		dataArray.push(data);
	});
	return dataArray;
}

function crawlForm(
	sectionId: string,
	sectionElement: Element | Document | ShadowRoot,
	items: (IValueItem | IDataSection)[],
	cncPath: string
): any {
	let dataObject: any = {};
	items.forEach((item) => {
		if ('itemType' in item) {
			let { result, node } = crawItem(item, sectionElement);
			dataObject[item.id] = result;

			if (item.valueRender) {
				try {
					let val = dataObject[item.id];
					let render = new Function('val, node', item.valueRender);
					let res = render.call(item, val, node);
					if (res !== undefined) {
						dataObject[item.id] = res;
					}
				} catch (err: any) {
					console.error('[' + item.id + '.valueRender]', err);
					dataObject[item.id] = `err(${err.message})`;
				}
			}

			if (item.external) {
				let { id, config } = item.external;
				appendExternalSection({
					id: id || item.id,
					config,
					connect: `${cncPath}/${sectionId}/${item.id}`,
				});
			}
		} else if ('sectionType' in item) {
			let { result } = crawSection(item, sectionElement, cncPath + '/' + sectionId);
			dataObject[item.id] = result;
		}
	});

	// dataSection 下直接挂 valueItem 的情况
	// if (sectionElement.nodeType === Node.DOCUMENT_NODE) {
	// 	return dataObject[items[0].id];
	// } else {
	// 	return dataObject;
	// }

	return dataObject;
}

function crawSection(
	sectionItem: IDataSection,
	parentElement: Element | Document | ShadowRoot = document,
	cncPath = ''
) {
	let result: any;
	let node: Element | Element[] | Document | ShadowRoot | null = parentElement;
	if (sectionItem.sectionType === 'form') {
		node = queryElem(sectionItem.selector, parentElement, sectionItem.domRender);
		if (node) {
			let crwData = crawlForm(sectionItem.id, node, sectionItem.items, cncPath);
			result = assignDeep(result ?? {}, crwData);
		}
	} else if (sectionItem.sectionType === 'list') {
		node = queryElems(sectionItem.selector, parentElement, sectionItem.domRender);
		if (node?.length) {
			let crwData = crawlList(sectionItem.id, node, sectionItem.items, cncPath);
			if (sectionItem.filterRender) {
				try {
					const renderFunc = new Function('val , i , arr', sectionItem.filterRender) as () => boolean;
					crwData = crwData.filter(renderFunc);
				} catch (err: any) {
					console.error('[' + sectionItem.id + '.filterRender]', err);
					crwData = [err.message];
				}
			}
			result = result ? result.push(...crwData) : crwData;
		}
	}
	if (sectionItem.dataRender) {
		try {
			let val = result;
			let render = new Function('val, node', sectionItem.dataRender);
			let res = render.call(sectionItem, val, node);
			if (res !== undefined) {
				result = res;
			}
		} catch (err: any) {
			console.error('[' + sectionItem.id + '.valueRender]', err);
			result = `err(${err.message})`;
		}
	}

	return { result, node };
}

function crawItem(item: IValueItem, parentElement: Element | Document | ShadowRoot = document) {
	let node: Element | Element[] | null = null;
	let result: string | (string | null)[] | IFileInfo | null = null;

	switch (item.itemType) {
		case 'text':
			node = queryElem(item.selector, parentElement, item.domRender);
			if (node) {
				if (item.valueProper) {
					result = (node as HTMLSpanElement).getAttribute(item.valueProper);
				} else {
					result = (node as HTMLSpanElement).innerText.trim();
				}
			}
			break;
		case 'textBox':
			node = queryElem(item.selector, parentElement, item.domRender);
			if (node) {
				if (node.tagName === 'INPUT' || node.tagName === 'TEXTAREA') {
					if (item.valueProper) {
						result = (node as HTMLInputElement).getAttribute(item.valueProper);
					} else {
						result = (node as HTMLInputElement).value.trim();
					}
				}
			}
			break;
		case 'radioBox':
			node = queryElems(item.selector, parentElement, item.domRender);
			if (node.length > 0) {
				for (let i = 0, l = node.length; i < l; i++) {
					let elem = node[i];
					let radioNode: HTMLInputElement;
					if (elem.tagName === 'INPUT') {
						radioNode = elem as HTMLInputElement;
					} else {
						radioNode = elem.querySelector('input[type=radio]') as HTMLInputElement;
					}
					// check prop
					if (radioNode?.checked) {
						if (elem.tagName === 'INPUT') {
							result = radioNode.getAttribute(item.valueProper ?? 'value');
						} else {
							if (item.valueProper) {
								result = elem.getAttribute(item.valueProper);
							} else {
								result = (elem as HTMLSpanElement).innerText.trim();
							}
						}
						break;
					}
				}
			}
			break;
		case 'checkBox':
			node = queryElems(item.selector, parentElement, item.domRender);
			result = [];
			if (node.length > 0) {
				for (let i = 0, l = node.length; i < l; i++) {
					let elem = node[i];
					let checkNode: HTMLInputElement;
					if (elem.tagName === 'INPUT') {
						checkNode = elem as HTMLInputElement;
					} else {
						checkNode = elem.querySelector('input[type=checkbox]') as HTMLInputElement;
					}
					// check prop
					if (checkNode?.checked) {
						if (elem.tagName === 'INPUT') {
							result.push(checkNode.getAttribute(item.valueProper ?? 'value'));
						} else {
							if (item.valueProper) {
								result.push(elem.getAttribute(item.valueProper));
							} else {
								result.push((elem as HTMLSpanElement).innerText.trim());
							}
						}
					}
				}
			}
			break;
		case 'dropBox':
			node = queryElem(item.selector, parentElement, item.domRender);
			if (node) {
				let x = (node as HTMLSelectElement).selectedIndex;
				let opt = (node as HTMLSelectElement).options[x];
				if (item.valueProper?.toLowerCase() === 'value') {
					result = opt.value;
				} else {
					result = opt.text;
				}
			}
			break;
		case 'download':
			node = queryElem(item.selector, parentElement, item.domRender);
			let dnCfg = GlobalConfig.downloadSection?.find((c) => c.id === item.downloadId);
			if (node && dnCfg) {
				result = crawlDownloadItem(dnCfg, node);
			}
			break;
	}

	return { result, node };
}

function crawlByConfig(dataSection: (IValueItem | IDataSection)[]) {
	let data: Record<string, any> = {};
	dataSection?.forEach((secItem) => {
		// is Section
		if ('sectionType' in secItem) {
			let { result } = crawSection(secItem);
			data[secItem.id] = result;
		} else if ('itemType' in secItem) {
			let crwData = crawlForm('', document, [secItem], '');
			data[secItem.id] = crwData[secItem.id];
		}
	});

	return data;
}

const crawlDownloadItem: (dn: IDownloadSection, elem: Element) => IFileInfo = (function () {
	let renders: Record<string, Function> = {};

	return function (dn: IDownloadSection, elem: Element): IFileInfo {
		let fileInfo: IFileInfo = {
			name: '',
			url: '',
			error: '',
		};
		if (elem.getBoundingClientRect().height > 0) {
			let fileName = dn.nameProper ? elem.getAttribute(dn.nameProper) : (elem as HTMLAnchorElement).text.trim();
			if (dn.nameRender) {
				try {
					let renderFnName = dn.id + '_dn_nameRender';
					if (!renders[renderFnName]) {
						renders[renderFnName] = new Function('name, node', dn.nameRender);
					}
					let dnNameRender = renders[renderFnName];
					let res = dnNameRender.call(dn, fileName, elem);
					if (res) {
						fileName = res.toString();
					}
				} catch (err: any) {
					console.error(dn.id + '-node[nameRender]', err);
					fileInfo.error = err.message;
				}
			}

			if (dn.downloadType === 'toPDF' && !fileInfo.error) {
				fileName += '.pdf';
			}

			fileInfo.name = fileName!;

			if (dn.downloadType === 'url' || dn.downloadType === 'toPDF') {
				let link: string;
				if (dn.linkProper) {
					link = elem.getAttribute(dn.linkProper) || '';
				} else if (elem.tagName === 'A') {
					link = (elem as HTMLAnchorElement).href;
				} else {
					link = '';
				}
				if (dn.linkRender) {
					try {
						let renderFnName = dn.id + '_dn_linkRender';
						if (!renders[renderFnName]) {
							renders[renderFnName] = new Function('link, node', dn.linkRender);
						}
						let dnLinkRender = renders[renderFnName];
						let res = dnLinkRender.call(dn, link, elem);
						if (res) {
							link = res.toString();
						}
					} catch (err: any) {
						console.error(dn.id + '-node[linkRender]', err);
						fileInfo.error = err.message;
					}
				}

				fileInfo.url = link;
			}
		}
		return fileInfo;
	};
})();

let GlobalConfig: IConfig;

function run(cfg: IConfig) {
	GlobalConfig = cfg;
	let { dataSection, switchSection, downloadSection, downloadRoot } = cfg;
	let result: IResult = {
		data: {},
		downloads: {}
	};

	if (dataSection) {
		result.data = crawlByConfig(dataSection);
	}

	if (switchSection) {
		let swRender = new Function('data, config', switchSection.switchRender);
		let swRes = swRender.call(switchSection, result.data, cfg);
		let matchedCase = switchSection.cases.find(
			(c) => c.case === swRes || (c.case instanceof Array && (c.case as string[]).indexOf(swRes) > -1)
		);
		if (matchedCase) {
			let swData = crawlByConfig(matchedCase.dataSection);
			result.data = assignDeep(result.data, swData);
		}
	}

	if (downloadRoot) {
		let formatter = (function () {
			let pattern = /\${(\w+([.]*\w*)*)\}(?!})/g;
			return function (template: string, json: any) {
				return template.replace(pattern, function (match, key) {
					const value = key.split('.').reduce((obj: any, k: string) => obj[k], json);
					if (value === undefined) {
						return null;
					} else {
						return value;
					}
				});
			};
		})();
		result.downloadRoot = formatter(downloadRoot, result);
	}

	if (downloadSection) {
		downloadSection.forEach((dn) => {
			let elems = queryElems(dn.selector, document, dn.domRender);
			// let count = elems.length;
			result.downloads![dn.id] = {
				label: dn.label,
				files: [],
			};
			// if (dn.filterRender) {
			// 	try {
			// 		const renderFunc = new Function('val , i , arr', dn.filterRender) as () => boolean;
			// 		elems = elems.filter(renderFunc);
			// 	} catch (err: any) {
			// 		console.error('[' + dn.id + '.filterRender]', err);
			// 	}
			// }
			let files = result.downloads![dn.id].files;
			elems.forEach((elem, i) => {
				if (elem.getBoundingClientRect().height > 0) {
					let fileInfo: IFileInfo = crawlDownloadItem(dn, elem);
					files.push(fileInfo);
				}
			});
		});
	}

	if (Object.keys(externalDict).length) {
		result.externalSection = externalDict;
	}

	return result;
}
