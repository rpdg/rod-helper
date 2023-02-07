function assignDeep(target, ...sources) {
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
            if (!Object.getOwnPropertyDescriptor(source, key).enumerable) {
                return;
            }
            if (typeof source[key] === 'object' && source[key] !== null) {
                let isPropertyDone = false;
                if (!result[key] ||
                    !(typeof result[key] === 'object') ||
                    Array.isArray(result[key]) !== Array.isArray(source[key])) {
                    if (hash.get(source[key])) {
                        result[key] = hash.get(source[key]);
                        isPropertyDone = true;
                    }
                    else {
                        result[key] = Array.isArray(source[key]) ? [] : {};
                        hash.set(source[key], result[key]);
                    }
                }
                if (!isPropertyDone) {
                    result[key]['__hash__'] = hash;
                    assignDeep(result[key], source[key]);
                }
            }
            else {
                Object.assign(result, { [key]: source[key] });
            }
        });
    });
    delete result['__hash__'];
    return result;
}
const externalDict = {};
function appendExternalSection(extObj) {
    let key = extObj.connect;
    if (!externalDict[key]) {
        externalDict[key] = extObj;
    }
}
function crawlList(sectionId, sectionElements, items) {
    let dataArray = [];
    let renders = {};
    sectionElements.forEach((tableRow) => {
        let data = {};
        items.forEach((item, i) => {
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
                }
                catch (err) {
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
function crawlForm(sectionId, sectionElement, items) {
    let dataObject = {};
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
            }
            catch (err) {
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
    if (sectionElement.nodeType === Node.DOCUMENT_NODE) {
        return dataObject[items[0].id];
    }
    else {
        return dataObject;
    }
}
function crawItem(item, parentElement) {
    var _a, _b, _c;
    let node = null;
    let result = null;
    if (item.itemType === 'text') {
        node = parentElement.querySelector(item.selector);
        if (node) {
            if (item.valueProper) {
                result = node.getAttribute(item.valueProper);
            }
            else {
                result = node.innerText.trim();
            }
        }
    }
    else if (item.itemType === 'textBox') {
        node = parentElement.querySelector(item.selector);
        if (node) {
            if (node.tagName === 'INPUT' || node.tagName === 'TEXTAREA') {
                if (item.valueProper) {
                    result = node.getAttribute(item.valueProper);
                }
                else {
                    result = node.value.trim();
                }
            }
        }
    }
    else if (item.itemType === 'radioBox') {
        node = parentElement.querySelectorAll(item.selector);
        if (node.length > 0) {
            for (let i = 0, l = node.length; i < l; i++) {
                let elem = node[i];
                let radioNode;
                if (elem.tagName === 'INPUT') {
                    radioNode = elem;
                }
                else {
                    radioNode = elem.querySelector('input[type=radio]');
                }
                if (radioNode === null || radioNode === void 0 ? void 0 : radioNode.checked) {
                    if (elem.tagName === 'INPUT') {
                        result = radioNode.getAttribute((_a = item.valueProper) !== null && _a !== void 0 ? _a : 'value');
                    }
                    else {
                        if (item.valueProper) {
                            result = elem.getAttribute(item.valueProper);
                        }
                        else {
                            result = elem.innerText.trim();
                        }
                    }
                    break;
                }
            }
        }
    }
    else if (item.itemType === 'checkBox') {
        node = parentElement.querySelectorAll(item.selector);
        result = [];
        if (node.length > 0) {
            for (let i = 0, l = node.length; i < l; i++) {
                let elem = node[i];
                let checkNode;
                if (elem.tagName === 'INPUT') {
                    checkNode = elem;
                }
                else {
                    checkNode = elem.querySelector('input[type=checkbox]');
                }
                if (checkNode === null || checkNode === void 0 ? void 0 : checkNode.checked) {
                    if (elem.tagName === 'INPUT') {
                        result.push(checkNode.getAttribute((_b = item.valueProper) !== null && _b !== void 0 ? _b : 'value'));
                    }
                    else {
                        if (item.valueProper) {
                            result.push(elem.getAttribute(item.valueProper));
                        }
                        else {
                            result.push(elem.innerText.trim());
                        }
                    }
                }
            }
        }
    }
    else if (item.itemType === 'dropBox') {
        node = parentElement.querySelector(item.selector);
        if (node) {
            let x = node.selectedIndex;
            let opt = node.options[x];
            if (((_c = item.valueProper) === null || _c === void 0 ? void 0 : _c.toLowerCase()) === 'value') {
                result = opt.value;
            }
            else {
                result = opt.text;
            }
        }
    }
    return { result, node };
}
function crawlByConfig(dataSection) {
    let data = {};
    dataSection === null || dataSection === void 0 ? void 0 : dataSection.forEach((secItem) => {
        var _a;
        if ('sectionType' in secItem) {
            let secNode = null;
            switch (secItem.sectionType) {
                case 'form':
                    secNode = document.querySelector(secItem.selector);
                    if (secNode) {
                        let crwData = crawlForm(secItem.id, secNode, secItem.items);
                        data[secItem.id] = assignDeep((_a = data[secItem.id]) !== null && _a !== void 0 ? _a : {}, crwData);
                    }
                    break;
                case 'list':
                    secNode = Array.from(document.querySelectorAll(secItem.selector));
                    if (secNode.length) {
                        let crwData = crawlList(secItem.id, secNode, secItem.items);
                        if (secItem.filterRender) {
                            try {
                                const renderFunc = new Function('val , i , arr', secItem.filterRender);
                                crwData = crwData.filter(renderFunc);
                            }
                            catch (err) {
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
                }
                catch (err) {
                    console.error('[' + secItem.id + '.valueRender]', err);
                    data[secItem.id] = `err(${err.message})`;
                }
            }
        }
        else if ('itemType' in secItem) {
            let crwData = crawlForm('', document, [secItem]);
            data[secItem.id] = crwData;
        }
    });
    return data;
}
function run(cfg) {
    let { dataSection, switchSection, downloadSection, downloadRoot } = cfg;
    let result = {
        data: {},
    };
    if (dataSection) {
        result.data = crawlByConfig(dataSection);
    }
    if (switchSection) {
        let swRender = new Function('data, config', switchSection.switchRender);
        let swRes = swRender.call(switchSection, result.data, cfg);
        let matchedCase = switchSection.cases.find((c) => c.case === swRes || (c.case instanceof Array && c.case.indexOf(swRes) > -1));
        if (matchedCase) {
            let swData = crawlByConfig(matchedCase.dataSection);
            result.data = assignDeep(result.data, swData);
        }
    }
    if (downloadRoot) {
        let formatter = (function () {
            let pattern = /\${(\w+([.]*\w*)*)\}(?!})/g;
            return function (template, json) {
                return template.replace(pattern, function (match, key) {
                    try {
                        return eval('(json.' + key + ')');
                    }
                    catch (e) {
                        return null;
                    }
                });
            };
        })();
        result.downloadRoot = formatter(downloadRoot, result);
    }
    if (downloadSection) {
        result.downloads = {};
        let renders = {};
        downloadSection === null || downloadSection === void 0 ? void 0 : downloadSection.forEach((dn) => {
            let elems = document.querySelectorAll(dn.selector);
            let count = elems.length;
            result.downloads[dn.id] = {
                count,
                fileNames: [],
                links: [],
                errors: [],
            };
            let fileNames = result.downloads[dn.id].fileNames;
            let links = result.downloads[dn.id].links;
            elems.forEach((elem, i) => {
                if (elem.getBoundingClientRect().height > 0) {
                    let fileName = dn.nameProper ? elem.getAttribute(dn.nameProper) : elem.text;
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
                        }
                        catch (err) {
                            console.error(dn.id + '-node[' + i + '.nameRender]', err);
                            fileName = err.message;
                        }
                    }
                    fileNames.push(fileName);
                    if (dn.type === 'url') {
                        if (elem.tagName === 'A') {
                            links.push(elem.href);
                        }
                        else {
                            links.push('');
                        }
                    }
                }
                else {
                    result.downloads[dn.id].count--;
                }
            });
        });
    }
    if (Object.keys(externalDict).length) {
        result.externalSection = externalDict;
    }
    return result;
}