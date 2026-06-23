(function () {
    if (window._apiInterceptorInjected) return;
    window._apiInterceptorInjected = true;

    function shouldIntercept(url) {
        return url.includes("_glaxy_c66_");
    }

    function reportRequest(data) {
        JSBridge.onApiResponse(JSON.stringify(data));
    }

    const origFetch = window.fetch;
    window.fetch = function (input, init = {}) {
        const url = typeof input === 'string' ? input : input.url;
        const method = init.method || 'GET';
        const headers = init.headers || {};
        const body = init.body || null;
        return origFetch(input, init).then(resp => {
            if (shouldIntercept(url)) {
                resp.clone().text().then(text => {
                    reportRequest({ url, method, headers, body, response: text });
                });
            }
            return resp;
        });
    };

    const origOpen = XMLHttpRequest.prototype.open;
    const origSend = XMLHttpRequest.prototype.send;
    const origSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader;

    XMLHttpRequest.prototype.open = function (method, url) {
        this._url = url;
        this._method = method;
        this._headers = {};
        return origOpen.apply(this, arguments);
    };
    XMLHttpRequest.prototype.setRequestHeader = function (key, value) {
        this._headers[key] = value;
        return origSetRequestHeader.call(this, key, value);
    };
    XMLHttpRequest.prototype.send = function (body) {
        this._body = body;
        this.addEventListener('load', function () {
            if (shouldIntercept(this._url)) {
                reportRequest({
                    url: this._url,
                    method: this._method,
                    headers: this._headers,
                    body: this._body,
                    response: this.responseText
                });
            }
        });
        return origSend.call(this, body);
    };

    // 重放请求方法（供页面侧主动复检使用）
    window.replayRequest = function (lastUserApiJsonStr) {
        try {
            const data = JSON.parse(lastUserApiJsonStr);
            const url = data.url || data.apiUrl;
            const method = (data.method || 'GET').toUpperCase();
            const headers = data.headers || {};
            const body = data.body || null;

            const fetchOptions = { method, headers };

            if (method === 'POST' || method === 'PUT') {
                fetchOptions.body = body;
            }

            return fetch(url, fetchOptions)
                .then(response => response.text())
                .then(text => {
                    console.log('Replay response:', text);
                    return text;
                })
                .catch(err => {
                    console.error('Replay error:', err);
                    throw err;
                });
        } catch (e) {
            console.error('Invalid JSON for replayLastRequest:', e);
            return Promise.reject(e);
        }
    };
})();
