type IDataSection = import('./types').IDataSection;
type IConfig = import('./types').IConfig;
type IExternal = import('./types').IExternal;
type IResult = import('./types').IResult;
type IValueItem = import('./types').IValueItem;

function assignDeep(
	target: any,
	...sources: {
		[x: string]: any;
	}[]
) {
	// 1. 参数校验
	if (target == null) {
		throw new TypeError('Cannot convert undefined or null to object');
	}

	// 2. 如果是基本类型数据转为包装对象
	let result = Object(target);

	// 3. 缓存已拷贝过的对象，解决引用关系丢失问题
	if (!result['__hash__']) {
		result['__hash__'] = new WeakMap();
	}
	let hash = result['__hash__'];

	sources.forEach((v) => {
		// 4. 如果是基本类型数据转为对象类型
		let source = Object(v);
		// 5. 遍历原对象属性，基本类型则值拷贝，对象类型则递归遍历
		Reflect.ownKeys(source).forEach((key) => {
			// 6. 跳过自有的不可枚举的属性
			if (!Object.getOwnPropertyDescriptor(source, key)!.enumerable) {
				return;
			}
			if (typeof source[key] === 'object' && source[key] !== null) {
				// 7. 属性的冲突处理和拷贝处理
				let isPropertyDone = false;
				if (
					!result[key] ||
					!(typeof result[key] === 'object') ||
					Array.isArray(result[key]) !== Array.isArray(source[key])
				) {
					// 当 target 没有该属性，或者属性类型和 source 不一致时，直接整个覆盖
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

function findBracketSubstring(str: string): string {
	let leftCount = 0;
	let leftStartIndex = -1;
	let result = '';

	if (str.indexOf('(') === -1 || str.length === 0) {
		return result;
	}

	for (let i = 0; i < str.length; i++) {
		if (str[i] === '(') {
			leftCount++;

			if (leftStartIndex === -1) {
				leftStartIndex = i;
			}
		} else if (str[i] === ')') {
			leftCount--;

			if (leftCount === 0) {
				result = str.substring(leftStartIndex + 1, i);
				break;
			}
		}
	}

	if (leftCount > 0) {
		throw new Error('Unbalanced brackets');
	}
	return result;
}

function replacePseudo(
	selector: string,
	parentElement: Element | Document | ShadowRoot = document
): { doc: Element | Document | ShadowRoot; selector: string } {
	let doc: Element | Document | ShadowRoot = parentElement;
	if (selector.startsWith(':frame(')) {
		let iframeSelector = findBracketSubstring(selector);
		let iframeElem = parentElement.querySelector(iframeSelector);
		if (iframeElem !== null) {
			doc = (iframeElem as HTMLIFrameElement)?.contentWindow!.document;
			selector = selector.substring(8 + iframeSelector.length + 1);
		}
	} else if (selector.startsWith(':shadow(')) {
		let slotSelector = findBracketSubstring(selector);
		let slotElem = parentElement.querySelector(slotSelector);
		if (slotElem !== null) {
			doc = (slotElem as HTMLSlotElement).shadowRoot!;
			selector = selector.substring(9 + slotSelector.length + 1);
		}
	}
	return { doc, selector };
}

function queryElem(selectorString: string, parentElement: Element | Document | ShadowRoot = document): Element | null {
	let secNode: Element | null = null;
	let { doc, selector } = replacePseudo(selectorString, parentElement);
	secNode = doc.querySelector(selector);
	return secNode;
}

function queryElems(selectorString: string, parentElement: Element | Document | ShadowRoot = document): Element[] {
	let secNodes: Element[] = [];
	let { doc, selector } = replacePseudo(selectorString, parentElement);
	secNodes = Array.from(doc.querySelectorAll(selector));
	return secNodes;
}

const externalDict: Record<string, IExternal> = {};

function appendExternalSection(extObj: IExternal) {
	let key = extObj.connect;
	if (!externalDict[key]) {
		externalDict[key] = extObj;
	}
}

function crawlList(sectionId: string, sectionElements: Element[], items: IValueItem[]): any[] {
	let dataArray: any[] = [];
	let renders: Record<string, Function> = {};

	sectionElements.forEach((element) => {
		let data: any = {};
		items.forEach((item) => {
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
					connect: `${sectionId}/${item.id}`,
				});
			}
		});
		dataArray.push(data);
	});
	return dataArray;
}

function crawlForm(sectionId: string, sectionElement: Element | Document | ShadowRoot, items: IValueItem[]): any {
	let dataObject: any = {};
	items.forEach((item) => {
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
				connect: `${sectionId}/${item.id}`,
			});
		}
	});

	// dataSection 下直接挂 valueItem 的情况
	if (sectionElement.nodeType === Node.DOCUMENT_NODE) {
		return dataObject[items[0].id];
	} else {
		return dataObject;
	}
}

function crawItem(item: IValueItem, parentElement: Element | Document | ShadowRoot = document) {
	let node: Element | Element[] | null = null;
	let result: string | (string | null)[] | null = null;

	if (item.itemType === 'text') {
		node = queryElem(item.selector, parentElement);
		if (node) {
			if (item.valueProper) {
				result = (node as HTMLSpanElement).getAttribute(item.valueProper);
			} else {
				result = (node as HTMLSpanElement).innerText.trim();
			}
		}
	} else if (item.itemType === 'textBox') {
		node = queryElem(item.selector, parentElement);
		if (node) {
			if (node.tagName === 'INPUT' || node.tagName === 'TEXTAREA') {
				if (item.valueProper) {
					result = (node as HTMLInputElement).getAttribute(item.valueProper);
				} else {
					result = (node as HTMLInputElement).value.trim();
				}
			}
		}
	} else if (item.itemType === 'radioBox') {
		node = queryElems(item.selector, parentElement);
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
	} else if (item.itemType === 'checkBox') {
		node = queryElems(item.selector, parentElement);
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
	} else if (item.itemType === 'dropBox') {
		node = queryElem(item.selector, parentElement);
		if (node) {
			let x = (node as HTMLSelectElement).selectedIndex;
			let opt = (node as HTMLSelectElement).options[x];
			if (item.valueProper?.toLowerCase() === 'value') {
				result = opt.value;
			} else {
				result = opt.text;
			}
		}
	}

	return { result, node };
}

function crawlByConfig(dataSection: (IValueItem | IDataSection)[]) {
	let data: Record<string, any> = {};
	dataSection?.forEach((secItem) => {
		// is Section
		if ('sectionType' in secItem) {
			let secNode: Element | Element[] | null = null;
			switch (secItem.sectionType) {
				case 'form':
					secNode = queryElem(secItem.selector, document);
					if (secNode) {
						let crwData = crawlForm(secItem.id, secNode, secItem.items);
						data[secItem.id] = assignDeep(data[secItem.id] ?? {}, crwData);
					}
					break;
				case 'list':
					secNode = queryElems(secItem.selector, document);
					if (secNode.length) {
						let crwData = crawlList(secItem.id, secNode, secItem.items);
						if (secItem.filterRender) {
							try {
								const renderFunc = new Function('val , i , arr', secItem.filterRender) as () => boolean;
								crwData = crwData.filter(renderFunc);
							} catch (err: any) {
								console.error('[' + secItem.id + '.filterRender]', err);
								crwData = [err.message];
							}
						}
						data[secItem.id] = data[secItem.id] ? data[secItem.id].push(...crwData) : crwData;
					}
					break;
				default:
			}
			if (secItem.dataRender && secItem.id in data) {
				try {
					let val = data[secItem.id];
					let render = new Function('val, node', secItem.dataRender);
					let res = render.call(secItem, val, secNode);
					if (res !== undefined) {
						data[secItem.id] = res;
					}
				} catch (err: any) {
					console.error('[' + secItem.id + '.valueRender]', err);
					data[secItem.id] = `err(${err.message})`;
				}
			}
		} else if ('itemType' in secItem) {
			let crwData = crawlForm('', document, [secItem]);
			data[secItem.id] = crwData;
		}
	});

	return data;
}

function run(cfg: IConfig) {
	let { dataSection, switchSection, downloadSection, downloadRoot } = cfg;
	let result: IResult = {
		data: {},
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
				return template.replace(pattern, function (match, key: string) {
					try {
						return eval('(json.' + key + ')');
					} catch (e) {
						return null;
					}
				});
			};
		})();
		result.downloadRoot = formatter(downloadRoot, result);
	}

	if (downloadSection) {
		result.downloads = {};
		let renders: Record<string, Function> = {};
		downloadSection?.forEach((dn) => {
			let elems = queryElems(dn.selector, document);
			let count = elems.length;
			result.downloads![dn.id] = {
				count,
				fileNames: [],
				links: [],
				errors: [],
			};
			let fileNames = result.downloads![dn.id].fileNames;
			let links = result.downloads![dn.id].links;
			elems.forEach((elem, i) => {
				if (elem.getBoundingClientRect().height > 0) {
					let fileName = dn.nameProper ? elem.getAttribute(dn.nameProper) : (elem as HTMLAnchorElement).text;
					if (dn.nameRender) {
						try {
							if (!renders[dn.id]) {
								renders[dn.id] = new Function('name, node', dn.nameRender);
							}
							let render = renders[dn.id];
							let res = render.call(dn, fileName, elem);
							if (res) {
								fileName = res.toString();
							}
						} catch (err: any) {
							console.error(dn.id + '-node[' + i + '.nameRender]', err);
							fileName = err.message;
						}
					}
					fileNames.push(fileName!);
					if (dn.type === 'url') {
						if (elem.tagName === 'A') {
							links.push((elem as HTMLAnchorElement).href);
						} else {
							links.push('');
						}
					}
				} else {
					result.downloads![dn.id].count--;
				}
			});
		});
	}

	if (Object.keys(externalDict).length) {
		result.externalSection = externalDict;
	}

	return result;
}
