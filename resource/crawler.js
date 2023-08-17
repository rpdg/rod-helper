"use strict";
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
function replacePseudo(selector, parentElement = document) {
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
                    ? pseudoElem.contentWindow.document
                    : pseudoElem.shadowRoot;
            selector = selector.slice(pseudoMatch[0].length).trim();
            ctxChanged = true;
        }
    }
    if (/^:(frame|shadow)\(/.test(selector)) {
        return replacePseudo(selector, doc);
    }
    return { doc, selector, ctxChanged };
}
function queryElem(selectorString, parentElement = document) {
    let secNode = null;
    let { doc, selector } = replacePseudo(selectorString, parentElement);
    secNode = doc.querySelector(selector);
    return secNode;
}
function queryElems(selectorString, parentElement = document) {
    let secNodes = [];
    let { doc, selector } = replacePseudo(selectorString, parentElement);
    secNodes = Array.from(doc.querySelectorAll(selector));
    return secNodes;
}
const externalDict = {};
function appendExternalSection(extObj) {
    let key = extObj.connect;
    if (!externalDict[key]) {
        externalDict[key] = extObj;
    }
}
function crawlList(sectionId, sectionElements, items, cncPath) {
    let dataArray = [];
    let renders = {};
    sectionElements.forEach((element) => {
        let data = {};
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
                        connect: `${cncPath}/${sectionId}/${item.id}`,
                    });
                }
            }
            else if ('sectionType' in item) {
                let { result } = crawSection(item, element, cncPath + '/' + sectionId);
                data[item.id] = result;
            }
        });
        dataArray.push(data);
    });
    return dataArray;
}
function crawlForm(sectionId, sectionElement, items, cncPath) {
    let dataObject = {};
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
                    connect: `${cncPath}/${sectionId}/${item.id}`,
                });
            }
        }
        else if ('sectionType' in item) {
            let { result } = crawSection(item, sectionElement, cncPath + '/' + sectionId);
            dataObject[item.id] = result;
        }
    });
    if (sectionElement.nodeType === Node.DOCUMENT_NODE) {
        return dataObject[items[0].id];
    }
    else {
        return dataObject;
    }
}
function crawSection(sectionItem, parentElement = document, cncPath = '') {
    let result;
    let node = parentElement;
    if (sectionItem.sectionType === 'form') {
        if (sectionItem.selector) {
            node = queryElem(sectionItem.selector, parentElement);
        }
        if (node) {
            let crwData = crawlForm(sectionItem.id, node, sectionItem.items, cncPath);
            result = assignDeep(result !== null && result !== void 0 ? result : {}, crwData);
        }
    }
    else if (sectionItem.sectionType === 'list') {
        if (sectionItem.selector) {
            node = queryElems(sectionItem.selector, parentElement);
        }
        else {
            node = [parentElement];
        }
        if (node.length) {
            let crwData = crawlList(sectionItem.id, node, sectionItem.items, cncPath);
            if (sectionItem.filterRender) {
                try {
                    const renderFunc = new Function('val , i , arr', sectionItem.filterRender);
                    crwData = crwData.filter(renderFunc);
                }
                catch (err) {
                    console.error('[' + sectionItem.id + '.filterRender]', err);
                    crwData = [err.message];
                }
            }
            result = result ? result.push(...crwData) : crwData;
        }
    }
    if (sectionItem.dataRender && sectionItem.id in result) {
        try {
            let val = result;
            let render = new Function('val, node', sectionItem.dataRender);
            let res = render.call(sectionItem, val, node);
            if (res !== undefined) {
                result = res;
            }
        }
        catch (err) {
            console.error('[' + sectionItem.id + '.valueRender]', err);
            result = `err(${err.message})`;
        }
    }
    return { result, node };
}
function crawItem(item, parentElement = document) {
    var _a, _b, _c;
    let node = null;
    let result = null;
    if (item.itemType === 'text') {
        node = queryElem(item.selector, parentElement);
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
        node = queryElem(item.selector, parentElement);
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
        node = queryElems(item.selector, parentElement);
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
        node = queryElems(item.selector, parentElement);
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
        node = queryElem(item.selector, parentElement);
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
        if ('sectionType' in secItem) {
            let { result } = crawSection(secItem);
            data[secItem.id] = result;
        }
        else if ('itemType' in secItem) {
            let crwData = crawlForm('', document, [secItem], '');
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
            let elems = queryElems(dn.selector, document);
            let count = elems.length;
            result.downloads[dn.id] = {
                label: dn.label,
                count,
                fileNames: [],
                links: [],
                errors: [],
            };
            let fileNames = result.downloads[dn.id].fileNames;
            let links = result.downloads[dn.id].links;
            elems.forEach((elem, i) => {
                if (elem.getBoundingClientRect().height > 0) {
                    let fileName = dn.nameProper
                        ? elem.getAttribute(dn.nameProper)
                        : elem.text.trim();
                    if (dn.nameRender) {
                        try {
                            let renderFnName = dn.id + '_dn_nameRender';
                            if (!renders[renderFnName]) {
                                renders[renderFnName] = new Function('name, node', dn.nameRender);
                            }
                            let render = renders[renderFnName];
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
                    if (dn.type === 'toPDF') {
                        fileName += '.pdf';
                    }
                    fileNames.push(fileName);
                    if (dn.type === 'url' || dn.type === 'toPDF') {
                        let link;
                        if (dn.linkProper) {
                            link = elem.getAttribute(dn.linkProper) || '';
                        }
                        else if (elem.tagName === 'A') {
                            link = elem.href;
                        }
                        else {
                            link = '';
                        }
                        if (dn.linkRender) {
                            try {
                                let renderFnName = dn.id + '_dn_linkRender';
                                if (!renders[renderFnName]) {
                                    renders[renderFnName] = new Function('link, node', dn.linkRender);
                                }
                                let render = renders[renderFnName];
                                let res = render.call(dn, link, elem);
                                if (res) {
                                    link = res.toString();
                                }
                            }
                            catch (err) {
                                console.error(dn.id + '-node[' + i + '.linkRender]', err);
                                link = err.message;
                            }
                        }
                        links.push(link);
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
