type DataSection = import('./types').DataSection;
type IConfig = import('./types').IConfig;
type IExternal = import('./types').IExternal;
type IResult = import('./types').IResult;
type ValueItem = import('./types').ValueItem;

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

const externalDict: Record<string, IExternal> = {};

function appendExternalSection(extObj: IExternal) {
	let key = extObj.connect;
	if (!externalDict[key]) {
		externalDict[key] = extObj;
	}
}

function crawlList(sectionId: string, sectionElements: Element[], items: ValueItem[]) {
	let dataArray: any[] = [];
	let renders: Record<string, Function> = {};

	sectionElements.forEach((tableRow) => {
		let data: any = {};
		items.forEach((item) => {
			let { result, node } = crawItem(item, tableRow);
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

function crawlForm(sectionId: string, sectionElement: Element | Document, items: ValueItem[]) {
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

function crawItem(item: ValueItem, parentElement: Element | Document) {
	let node: Element | NodeListOf<Element> | null = null;
	let result: string | (string | null)[] | null = null;

	if (item.itemType === 'text') {
		node = parentElement.querySelector(item.selector);
		if (node) {
			if (item.valueProper) {
				result = (node as HTMLSpanElement).getAttribute(item.valueProper);
			} else {
				result = (node as HTMLSpanElement).innerText.trim();
			}
		}
	} else if (item.itemType === 'textBox') {
		node = parentElement.querySelector(item.selector);
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
		node = parentElement.querySelectorAll(item.selector);
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
		node = parentElement.querySelectorAll(item.selector);
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
		node = parentElement.querySelector(item.selector);
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

function crawlByConfig(dataSection: (ValueItem | DataSection)[]) {
	let data: Record<string, any> = {};
	dataSection?.forEach((secItem) => {
		// is Section
		if ('sectionType' in secItem) {
			let secNode: Element | Element[] | null = null;
			switch (secItem.sectionType) {
				case 'form':
					secNode = document.querySelector(secItem.selector);
					if (secNode) {
						let crwData = crawlForm(secItem.id, secNode, secItem.items);
						data[secItem.id] = assignDeep(data[secItem.id] ?? {}, crwData);
					}
					break;
				case 'list':
					secNode = Array.from(document.querySelectorAll(secItem.selector));
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
			let elems = document.querySelectorAll(dn.selector);
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
