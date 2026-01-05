(function() {
	'use strict';

	function Probe(options, inputValues) {
		this.options = options;
		this.inputValues = inputValues || {};
		this.requestsPrintQueue = [];
		this.sentAjax = [];
		this.curElement = {};
		this.winOpen = [];
		this.resources = [];
		this.eventsMap = [];
		this.triggeredEvents = [];
		this.websockets = [];
		this.html = "";
		this.printedRequests = [];
		this.DOMSnapshot = [];
		this._pendingJsonp = [];
		this._pendingWebsocket = [];
		this.currentUserScriptParameters = [];
		this._lastRequestId = 0;
		this.started_at = null;
		this.textComparator = null;
		this.setTimeout = window.setTimeout.bind(window);
		this.setInterval = window.setInterval.bind(window);
		this.DOMMutations = [];
		this.DOMMutationsToPop = [];
		this.totalDOMMutations = 0;
		this.UI = null;
		this.originals = {};
	}

	Probe.prototype.getRootNodes = function(elements) {
		const rootElements = [];
		for (let a = 0; a < elements.length; a++) {
			var p = elements[a];
			var root = null;
			while (p) {
				if (elements.indexOf(p) != -1) {
					root = p;
				}
				p = p.parentNode;
			}
			if (root && rootElements.indexOf(root) == -1) {
				rootElements.push(root);
			}
		}
		return rootElements;
	};

	Probe.prototype.popMutation = function() {
		const roots = this.getRootNodes(this.DOMMutations);
		this.DOMMutations = [];
		this.DOMMutationsToPop = this.DOMMutationsToPop.concat(roots);
		const first = this.DOMMutationsToPop.splice(0, 1);
		return first.length == 1 ? first[0] : null;
	};

	Probe.prototype.objectInArray = function(arr, el, ignoreProperties) {
		ignoreProperties = ignoreProperties || [];
		if (arr.length == 0) return false;
		if (typeof arr[0] != 'object')
			return arr.indexOf(el) > -1;
		for (let a = 0; a < arr.length; a++) {
			var found = true;
			for (let k in arr[a]) {
				if (arr[a][k] != el[k] && ignoreProperties.indexOf(k) == -1) {
					found = false;
				}
			}
			if (found) return true;
		}
		return false;
	};

	Probe.prototype.arrayUnique = function(arr, ignoreProperties) {
		var ret = [];
		for (let a = 0; a < arr.length; a++) {
			if (!this.objectInArray(ret, arr[a], ignoreProperties))
				ret.push(arr[a]);
		}
		return ret;
	};

	Probe.prototype.compareObjects = function(obj1, obj2) {
		var p;
		for (p in obj1)
			if (obj1[p] != obj2[p]) return false;
		for (p in obj2)
			if (obj2[p] != obj1[p]) return false;
		return true;
	};

	Probe.prototype.replaceUrlQuery = function(url, qs) {
		var anchor = document.createElement("a");
		anchor.href = url;
		return anchor.protocol + "//" + anchor.host + anchor.pathname + (qs ? "?" + qs : "") + anchor.hash;
	};

	Probe.prototype.removeUrlParameter = function(url, par) {
		var anchor = document.createElement("a");
		anchor.href = url;
		var pars = anchor.search.substr(1).split(/(?:&amp;|&)+/);
		for (let a = pars.length - 1; a >= 0; a--) {
			if (pars[a].split("=")[0] == par)
				pars.splice(a, 1);
		}
		return anchor.protocol + "//" + anchor.host + anchor.pathname + (pars.length > 0 ? "?" + pars.join("&") : "") + anchor.hash;
	};

	Probe.prototype.getAbsoluteUrl = function(url) {
		var anchor = document.createElement('a');
		anchor.href = url;
		return anchor.href;
	};

	Probe.prototype.randomizeArray = function(arr) {
		var a, ri;
		for (a = arr.length - 1; a > 0; a--) {
			ri = Math.floor(Math.random() * (a + 1));
			[arr[a], arr[ri]] = [arr[ri], arr[a]];
		}
	};

	Probe.prototype.Request = function(type, method, url, data, trigger, extra_headers) {
		this.type = type;
		this.method = method;
		this.url = url;
		this.data = data || null;
		this.trigger = trigger || null;
		this.extra_headers = extra_headers || {};
	};

	Probe.prototype.Request.prototype.key = function() {
		var key = "" + this.type + this.method + this.url + (this.data ? this.data : "") + (this.trigger ? this.trigger : "");
		return key;
	};

	Probe.prototype.requestToJson = function(req) {
		return JSON.stringify(this.requestToObject(req));
	};

	Probe.prototype.requestToObject = function(req) {
		var obj = {
			type: req.type,
			method: req.method,
			url: req.url,
			data: req.data || null,
			extra_headers: req.extra_headers
		};
		if (req.trigger) obj.trigger = { element: this.describeElement(req.trigger.element), event: req.trigger.event };
		return obj;
	};

	Probe.prototype.setVal = async function(el) {
		var options = this.options;
		var _this = this;

		var ueRet = await this.dispatchProbeEvent("fillinput", { element: this.getElementSelector(el) });
		if (ueRet === false) return;

		var getv = function(type) {
			if (!(type in _this.inputValues))
				type = "string";
			return _this.inputValues[type];
		};

		var setv = function(name) {
			var ret = getv('string');
			for (var a = 0; a < options.inputNameMatchValue.length; a++) {
				try {
					var regexp = new RegExp(options.inputNameMatchValue[a].name, "gi");
					if (name.match(regexp)) {
						ret = getv(options.inputNameMatchValue[a].value);
					}
				} catch (e) {}
			}
			return ret;
		};

		var triggerChange = function() {
			_this.trigger(el, 'input');
		};

		if (el.nodeName.toLowerCase() == 'textarea') {
			el.value = setv(el.name);
			triggerChange();
			return true;
		}

		if (el.nodeName.toLowerCase() == 'select') {
			var opts = el.getElementsByTagName('option');
			if (opts.length > 1) {
				el.value = opts[opts.length - 1].value;
			} else {
				el.value = setv(el.name);
			}
			triggerChange();
			return true;
		}

		var type = el.type.toLowerCase();

		switch (type) {
			case 'button':
			case 'hidden':
			case 'submit':
			case 'file':
				return false;
			case '':
			case 'text':
			case 'search':
				el.value = setv(el.name);
				break;
			case 'radio':
			case 'checkbox':
				el.setAttribute('checked', !(el.getAttribute('checked')));
				break;
			case 'range':
			case 'number':
				if ('min' in el && el.min) {
					el.value = (parseInt(el.min) + parseInt(('step' in el) ? el.step : 1));
				} else {
					el.value = parseInt(getv('number'));
				}
				break;
			case 'password':
			case 'color':
			case 'date':
			case 'email':
			case 'month':
			case 'time':
			case 'url':
			case 'week':
			case 'tel':
				el.value = getv(type);
				break;
			case 'datetime-local':
				el.value = getv('datetimeLocal');
				break;
			default:
				return false;
		}

		triggerChange();
		return true;
	};

	Probe.prototype.fillInputValues = async function(element) {
		const inputs = ["input", "select", "textarea"];
		element = element || document;
		var els;
		try {
			els = element.querySelectorAll(inputs.join(","));
		} catch (e) {
			return false;
		}
		if (inputs.indexOf(element.nodeName.toLowerCase()) > -1) {
			await this.setVal(element);
			this.trigger(element, 'input');
		}
		for (var a = 0; a < els.length; a++) {
			await this.setVal(els[a]);
			this.trigger(els[a], 'input');
		}
	};

	Probe.prototype.trigger = function(el, evname) {
		if (el.tagName == "INPUT" && el.type.toLowerCase() == 'color' && evname == 'click') {
			return;
		}

		if (typeof evname != "string") return;

		var pdh = function(e) {
			if (el.matches("a") || el.form) {
				var newUrl;
				try {
					newUrl = el.form ? new URL(el.form.action) : new URL(el.href);
				} catch (e) {
					newUrl = null;
				}
				if (newUrl) {
					if (newUrl.protocol == "javascript:") {
						return;
					}
					const curUrl = new URL(document.location.href);
					if (newUrl.hash && newUrl.hash != curUrl.hash) {
						newUrl.hash = "";
						curUrl.hash = "";
						if (newUrl.toString() == curUrl.toString()) {
							return;
						}
					}
				}
			}
			e.preventDefault();
			e.stopPropagation();
			e.stopImmediatePropagation();
		};

		if ('createEvent' in document) {
			var evt = null;
			if (this.options.simulateRealEvents) {
				if (this.options.mouseEvents.indexOf(evname) != -1) {
					evt = new MouseEvent(evname, { view: window, bubbles: true, cancelable: true });
					if (evname.toLowerCase() == "click" && el.matches('a, button, input[type="submit"], input[type="file"]')) {
						el.addEventListener(evname, pdh);
					}
				}
			}

			if (evt == null) {
				evt = document.createEvent('HTMLEvents');
				evt.initEvent(evname, true, false);
			}

			el.dispatchEvent(evt);
		} else {
			evname = 'on' + evname;
			if (evname in el && typeof el[evname] == "function") {
				el[evname]();
			}
		}
		try {
			el.removeEventListener(evname, pdh);
		} catch (e) {}
	};

	Probe.prototype.isEventTriggerable = function(event) {
		return ['load', 'unload', 'beforeunload'].indexOf(event) == -1;
	};

	Probe.prototype.getEventsForElement = function(element) {
		var events = [];
		var map = this.options.eventsMap;
		try {
			for (var selector in map) {
				if (element.webkitMatchesSelector(selector)) {
					events = events.concat(map[selector]);
				}
			}
		} catch (e) {
			return events;
		}
		return events;
	};

	Probe.prototype.triggerElementEvent = function(element, event) {
		var teObj = { el: element, ev: event };
		this.setTrigger({});
		if (!event) return;
		if (!this.isEventTriggerable(event) || this.objectInArray(this.triggeredEvents, teObj))
			return;

		this.setTrigger({ element: element, event: event });
		this.triggeredEvents.push(teObj);
		this.trigger(element, event);
	};

	Probe.prototype.getTrigger = function() {
		if (!this.curElement || !this.curElement.element)
			return null;
		return {
			element: this.describeElement(this.curElement.element),
			event: this.curElement.event
		};
	};

	Probe.prototype.describeElements = function(els) {
		var ret = [];
		for (let el of els) {
			ret.push(this.describeElement(el));
		}
		return ret;
	};

	Probe.prototype.describeElement = function(el) {
		return this.getElementSelector(el);
	};

	Probe.prototype.stringifyElement = function(el) {
		if (!el)
			return "[]";
		var tagName = (el == document ? "DOCUMENT" : (el == window ? "WINDOW" : el.tagName));
		var text = null;
		if (el.textContent) {
			text = el.textContent.trim().replace(/\s/, " ").substring(0, 10);
			if (text.indexOf(" ") > -1) text = "'" + text + "'";
		}
		var className = (el.className && typeof el.className == 'string') ? (el.className.indexOf(" ") != -1 ? "'" + el.className + "'" : el.className) : "";
		var descr = "[" +
			(tagName ? tagName + " " : "") +
			(el.name && typeof el.name == 'string' ? el.name + " " : "") +
			(className ? "." + className + " " : "") +
			(el.id ? "#" + el.id + " " : "") +
			(el.src ? "src=" + el.src + " " : "") +
			(el.action ? "action=" + el.action + " " : "") +
			(el.method ? "method=" + el.method + " " : "") +
			(el.value ? "v=" + el.value + " " : "") +
			(text ? "txt=" + text : "") +
			"]";
		return descr;
	};

	Probe.prototype._isHTMLElement = function(element) {
		try {
			return element instanceof window.top.HTMLElement || element instanceof element.ownerDocument?.defaultView?.HTMLElement;
		} catch (e) {
			return false;
		}
	};

	Probe.prototype._isSVGElement = function(element) {
		try {
			return element instanceof window.top.SVGElement || element instanceof element.ownerDocument?.defaultView?.SVGElement;
		} catch (e) {
			return false;
		}
	};

	Probe.prototype._getElementSelector = function(element) {
		if (!element || (!this._isHTMLElement(element) && !this._isSVGElement(element))) {
			return "";
		}
		var name = element.nodeName.toLowerCase();
		var ret = [];
		var selector = "";
		var id = element.getAttribute("id");

		if (id && id.match(/^[a-z][a-z0-9\-_:\.]*$/i) && element.ownerDocument.querySelectorAll(`#${id}`).length == 1) {
			selector = "#" + id;
		} else {
			let p = element;
			let cnt = 1;
			while (p = p.previousSibling) {
				if (this._isHTMLElement(p) && p.nodeName.toLowerCase() == name) {
					cnt++;
				}
			}
			selector = name + (cnt > 1 ? `:nth-of-type(${cnt})` : "");
			if (element != element.ownerDocument.documentElement && name != "body" && element.parentNode) {
				ret.push(this._getElementSelector(element.parentNode));
			}
		}
		ret.push(selector);
		return ret.join(" > ");
	};

	Probe.prototype.getElementSelector = function(element) {
		let elSelector = this._getElementSelector(element);
		const selectors = !!elSelector ? [elSelector] : [];
		let frame = element.ownerDocument.defaultView.frameElement;
		while (frame) {
			elSelector = frame.ownerDocument.defaultView.__PROBE__._getElementSelector(frame);
			if (elSelector) {
				selectors.push(elSelector);
			}
			frame = frame.ownerDocument.defaultView.frameElement;
		}
		if (selectors.length == 1) {
			return selectors[0];
		}
		return "inframe/" + selectors.reverse().join(" ; ");
	};

	Probe.prototype.getFormAsRequest = function(form) {
		var formObj = {};
		var inputs = null;
		var par;

		formObj.method = form.getAttribute("method");
		if (!formObj.method) {
			formObj.method = "GET";
		} else {
			formObj.method = formObj.method.toUpperCase();
		}

		formObj.url = form.getAttribute("action");
		if (!formObj.url) formObj.url = document.location.href;
		formObj.data = [];
		inputs = form.querySelectorAll("input, select, textarea");
		for (var a = 0; a < inputs.length; a++) {
			if (!inputs[a].name) continue;
			par = encodeURIComponent(inputs[a].name) + "=" + encodeURIComponent(inputs[a].value);
			if (inputs[a].tagName == "INPUT" && inputs[a].type != null) {
				switch (inputs[a].type.toLowerCase()) {
					case "button":
					case "submit":
						break;
					case "checkbox":
					case "radio":
						if (inputs[a].checked)
							formObj.data.push(par);
						break;
					default:
						formObj.data.push(par);
				}
			} else {
				formObj.data.push(par);
			}
		}

		formObj.data = formObj.data.join("&");

		if (formObj.method == "GET") {
			var url = this.replaceUrlQuery(formObj.url, formObj.data);
			req = new this.Request("form", "GET", url);
		} else {
			var req = new this.Request("form", "POST", formObj.url, formObj.data, this.getTrigger());
		}

		return req;
	};

	Probe.prototype.addEventToMap = function(element, event) {
		for (var a = 0; a < this.eventsMap.length; a++) {
			if (this.eventsMap[a].element == element) {
				this.eventsMap[a].events.push(event);
				return;
			}
		}
		this.eventsMap.push({
			element: element,
			events: [event]
		});
	};

	Probe.prototype.dispatchProbeEvent = async function(name, params) {
		return await window.__htcrawl_probe_event__(name, params);
	};

	Probe.prototype.jsonpHook = function(node) {
		if (!this._isHTMLElement(node) || !node.matches("script")) return;
		var src = node.getAttribute("src");
		if (!src) return;
		var _this = this;

		var a = document.createElement("a");
		a.href = src;

		if (!a.search) return;

		var req = new this.Request("jsonp", "GET", src, null, this.getTrigger());
		node.__request = req;

		this._pendingJsonp.push(node);

		var ev = function() {
			var i = _this._pendingJsonp.indexOf(node);
			if (i == -1) {
			} else {
				_this._pendingJsonp.splice(i, 1);
			}

			_this.dispatchProbeEvent("jsonpCompleted", {
				request: req,
				script: _this.describeElement(node)
			});
			node.removeEventListener("load", ev);
			node.removeEventListener("error", ev);
		};

		node.addEventListener("load", ev);
		node.addEventListener("error", ev);

		this.dispatchProbeEvent("jsonp", {
			request: req
		});
	};

	Probe.prototype.triggerWebsocketEvent = function(url) {
		var req = new this.Request("websocket", "GET", url, null, this.getTrigger());
		this.dispatchProbeEvent("websocket", { request: req });
	};

	Probe.prototype.triggerWebsocketMessageEvent = function(url, message) {
		var req = new this.Request("websocket", "GET", url, null, null);
		this.dispatchProbeEvent("websocketMessage", { request: req, message: message });
	};

	Probe.prototype.triggerWebsocketSendEvent = async function(url, message) {
		var req = new this.Request("websocket", "GET", url, null, null);
		return await this.dispatchProbeEvent("websocketSend", { request: req, message: message });
	};

	Probe.prototype.triggerFormSubmitEvent = function(form) {
		var req = this.getFormAsRequest(form);
		this.dispatchProbeEvent("formSubmit", {
			request: req,
			form: this.describeElement(form)
		});
	};

	Probe.prototype.triggerNavigationEvent = function(url, method, data) {
		var req = null;
		method = method || "GET";
		url = url.split("#")[0];
		req = new this.Request("navigation", method, url, data);
		this.dispatchProbeEvent("navigation", {
			request: req
		});
	};

	Probe.prototype.triggerPostMessageEvent = async function(destination, message, targetOrigin, transfer) {
		return await this.dispatchProbeEvent("postmessage", {
			destination: destination,
			message: message,
			targetOrigin: targetOrigin,
			transfer: transfer,
		});
	};

	Probe.prototype.waitRequests = async function(requests) {
		var _this = this;
		var reqPerformed = false;
		return new Promise((resolve, reject) => {
			var timeout = _this.options.ajaxTimeout;
			var t = _this.setInterval(function() {
				if (timeout <= 0 || requests.length == 0) {
					clearInterval(t);
					resolve(reqPerformed);
					return;
				}
				timeout -= 1;
				reqPerformed = true;
			}, 0);
		});
	};

	Probe.prototype.waitJsonp = async function() {
		await this.waitRequests(this._pendingJsonp);
		if (this._pendingJsonp.length > 0) {
			for (let req of this._pendingJsonp) {
				await this.dispatchProbeEvent("jsonpCompleted", {
					request: req.__request,
					response: null,
					timedout: true
				});
			}
		}
		this._pendingJsonp = [];
	};

	Probe.prototype.waitWebsocket = async function() {
		await this.waitRequests(this._pendingWebsocket);
		this._pendingWebsocket = [];
	};

	Probe.prototype.setTrigger = function(val) {
		this.curElement = val;
	};

	Probe.prototype._newMutationObserver = function(element) {
		var _this = this;
		element = element || document.documentElement;

		var observer = new MutationObserver(function(mutations) {
			for (let m of mutations) {
				for (let n of m.addedNodes) {
					if (_this._isHTMLElement(n) || _this._isSVGElement(n)) {
						if (_this.DOMMutations.indexOf(n) == -1) {
							_this.DOMMutations.push(n);
							_this.totalDOMMutations++;
						}
					}
				}
			}
		});

		observer.observe(element, {
			childList: true,
			subtree: true
		});
	};

	window.__PROBE__ = new Probe(options, inputValues);
})();
