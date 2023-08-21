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
function queryElem(selectorString, parentElement = document, domRender) {
    let secNode = null;
    if (!selectorString) {
        secNode = parentElement;
    }
    else {
        let { doc, selector } = replacePseudo(selectorString, parentElement);
        secNode = doc.querySelector(selector);
    }
    if (domRender) {
        let rFn = new Function('dom', domRender);
        secNode = rFn.call(window, secNode);
    }
    return secNode;
}
function queryElems(selectorString, parentElement = document, domRender) {
    let secNodes = [];
    if (!selectorString) {
        secNodes = [parentElement];
    }
    else {
        let { doc, selector } = replacePseudo(selectorString, parentElement);
        secNodes = Array.from(doc.querySelectorAll(selector));
    }
    if (domRender) {
        let rFn = new Function('dom', domRender);
        secNodes = rFn.call(window, secNodes);
    }
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
        node = queryElem(sectionItem.selector, parentElement, sectionItem.domRender);
        if (node) {
            let crwData = crawlForm(sectionItem.id, node, sectionItem.items, cncPath);
            result = assignDeep(result !== null && result !== void 0 ? result : {}, crwData);
        }
    }
    else if (sectionItem.sectionType === 'list') {
        node = queryElems(sectionItem.selector, parentElement, sectionItem.domRender);
        if (node === null || node === void 0 ? void 0 : node.length) {
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
    if (sectionItem.dataRender) {
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
    var _a, _b, _c, _d;
    let node = null;
    let result = null;
    switch (item.itemType) {
        case 'text':
            node = queryElem(item.selector, parentElement, item.domRender);
            if (node) {
                if (item.valueProper) {
                    result = node.getAttribute(item.valueProper);
                }
                else {
                    result = node.innerText.trim();
                }
            }
            break;
        case 'textBox':
            node = queryElem(item.selector, parentElement, item.domRender);
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
            break;
        case 'radioBox':
            node = queryElems(item.selector, parentElement, item.domRender);
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
            break;
        case 'checkBox':
            node = queryElems(item.selector, parentElement, item.domRender);
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
            break;
        case 'dropBox':
            node = queryElem(item.selector, parentElement, item.domRender);
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
            break;
        case 'download':
            node = queryElem(item.selector, parentElement, item.domRender);
            let dnCfg = (_d = GlobalConfig.downloadSection) === null || _d === void 0 ? void 0 : _d.find((c) => c.id === item.downloadId);
            if (node && dnCfg) {
                result = crawlDownloadItem(dnCfg, node);
            }
            break;
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
const crawlDownloadItem = (function () {
    let renders = {};
    return function (dn, elem) {
        let fileInfo = {
            name: '',
            url: '',
            error: '',
        };
        if (elem.getBoundingClientRect().height > 0) {
            let fileName = dn.nameProper ? elem.getAttribute(dn.nameProper) : elem.text.trim();
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
                }
                catch (err) {
                    console.error(dn.id + '-node[nameRender]', err);
                    fileInfo.error = err.message;
                }
            }
            if (dn.downloadType === 'toPDF' && !fileInfo.error) {
                fileName += '.pdf';
            }
            fileInfo.name = fileName;
            if (dn.downloadType === 'url' || dn.downloadType === 'toPDF') {
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
                        let dnLinkRender = renders[renderFnName];
                        let res = dnLinkRender.call(dn, link, elem);
                        if (res) {
                            link = res.toString();
                        }
                    }
                    catch (err) {
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
let GlobalConfig;
function run(cfg) {
    GlobalConfig = cfg;
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
                    const value = key.split('.').reduce((obj, k) => obj[k], json);
                    if (value === undefined) {
                        return null;
                    }
                    else {
                        return value;
                    }
                });
            };
        })();
        result.downloadRoot = formatter(downloadRoot, result);
    }
    if (downloadSection) {
        result.downloads = {};
        downloadSection === null || downloadSection === void 0 ? void 0 : downloadSection.forEach((dn) => {
            let elems = queryElems(dn.selector, document, dn.domRender);
            result.downloads[dn.id] = {
                label: dn.label,
                files: [],
            };
            let files = result.downloads[dn.id].files;
            elems.forEach((elem, i) => {
                if (elem.getBoundingClientRect().height > 0) {
                    let fileInfo = crawlDownloadItem(dn, elem);
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
