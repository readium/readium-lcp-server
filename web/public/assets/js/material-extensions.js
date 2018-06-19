/**
 * MDL Date Picker
 */
(function (global, factory) {
    typeof exports === 'object' && typeof module !== 'undefined' ? module.exports = factory(require('rome'), require('moment')) :
        typeof define === 'function' && define.amd ? define(['rome', 'moment'], factory) :
            (global.MaterialDatetimePicker = factory(global.rome, global.moment));
}(this, (function (rome, moment) {
    'use strict';

    rome = 'default' in rome ? rome['default'] : rome;
    moment = 'default' in moment ? moment['default'] : moment;

    var popupTemplate = (function () {
        return "<div class=\"c-datepicker\">" +
            "<div class=\"c-datepicker__header\">" +
            "<div class=\"c-datepicker__header-day\">" +
            "<span class=\"js-day\">Monday</span>" +
            "</div>" +
            "<div class=\"c-datepicker__header-date\">" +
            "<span class=\"c-datepicker__header-date__month js-date-month\"></span>" +
            "</div>" +
            "</div>" +
            "<div class=\"c-datepicker__calendar\"></div>" +
            "<div class=\"modal-btns\">" +
            "<a class=\"c-btn c-btn--flat js-cancel\">Cancel</a>" +
            "<a class=\"c-btn c-btn--flat js-ok\">OK</a>" +
            "</div>";
    });

    var scrimTemplate = (function (_ref) {
        var styles = _ref.styles;
        return "<div class=\"" + styles.scrim + "\"></div>";
    });

    var classCallCheck = function (instance, Constructor) {
        if (!(instance instanceof Constructor)) {
            throw new TypeError("Cannot call a class as a function");
        }
    };

    var createClass = function () {
        function defineProperties(target, props) {
            for (var i = 0; i < props.length; i++) {
                var descriptor = props[i];
                descriptor.enumerable = descriptor.enumerable || false;
                descriptor.configurable = true;
                if ("value" in descriptor) descriptor.writable = true;
                Object.defineProperty(target, descriptor.key, descriptor);
            }
        }

        return function (Constructor, protoProps, staticProps) {
            if (protoProps) defineProperties(Constructor.prototype, protoProps);
            if (staticProps) defineProperties(Constructor, staticProps);
            return Constructor;
        };
    }();


    var inherits = function (subClass, superClass) {
        if (typeof superClass !== "function" && superClass !== null) {
            throw new TypeError("Super expression must either be null or a function, not " + typeof superClass);
        }

        subClass.prototype = Object.create(superClass && superClass.prototype, {
            constructor: {
                value: subClass,
                enumerable: false,
                writable: true,
                configurable: true
            }
        });
        if (superClass) Object.setPrototypeOf ? Object.setPrototypeOf(subClass, superClass) : subClass.__proto__ = superClass;
    };


    var possibleConstructorReturn = function (self, call) {
        if (!self) {
            throw new ReferenceError("this hasn't been initialised - super() hasn't been called");
        }

        return call && (typeof call === "object" || typeof call === "function") ? call : self;
    };


    var slicedToArray = function () {
        function sliceIterator(arr, i) {
            var _arr = [];
            var _n = true;
            var _d = false;
            var _e = undefined;

            try {
                for (var _i = arr[Symbol.iterator](), _s; !(_n = (_s = _i.next()).done); _n = true) {
                    _arr.push(_s.value);

                    if (i && _arr.length === i) break;
                }
            } catch (err) {
                _d = true;
                _e = err;
            } finally {
                try {
                    if (!_n && _i["return"]) _i["return"]();
                } finally {
                    if (_d) throw _e;
                }
            }

            return _arr;
        }

        return function (arr, i) {
            if (Array.isArray(arr)) {
                return arr;
            } else if (Symbol.iterator in Object(arr)) {
                return sliceIterator(arr, i);
            } else {
                throw new TypeError("Invalid attempt to destructure non-iterable instance");
            }
        };
    }();


    var toConsumableArray = function (arr) {
        if (Array.isArray(arr)) {
            for (var i = 0, arr2 = Array(arr.length); i < arr.length; i++) arr2[i] = arr[i];

            return arr2;
        } else {
            return Array.from(arr);
        }
    };

//
// basic event triggering and listening
//
    var Events = function () {
        function Events() {
            classCallCheck(this, Events);

            this._events = {
                '*': []
            };
        }

        createClass(Events, [{
            key: 'trigger',
            value: function trigger(eventName, evtData) {
                var _this = this;

                eventName.split(' ').forEach(function (evtName) {
                    // trigger a global event event
                    _this._events['*'].forEach(function (evt) {
                        return evt.fn.call(evt.scope, evtName, evtData);
                    });
                    // if there are any listeners to this event
                    // then fire their handlers
                    if (_this._events[evtName]) {
                        _this._events[evtName].forEach(function (evt) {
                            evt.fn.call(evt.scope, evtData);
                        });
                    }
                });
                return this;
            }
        }, {
            key: 'on',
            value: function on(eventName, fn, scope) {
                if (!this._events[eventName]) this._events[eventName] = [];
                this._events[eventName].push({
                    eventName: eventName,
                    fn: fn,
                    scope: scope || this
                });
                return this;
            }
        }, {
            key: 'off',
            value: function off(eventName, fn) {
                if (!this._events[eventName]) return this;
                if (!fn) {
                    this._events[eventName] = [];
                }
                this._events[eventName] = this._events[eventName].filter(function (evt) {
                    return evt.fn !== fn;
                });
                return this;
            }
        }, {
            key: 'once',
            value: function once(eventName, fn, scope) {
                var _this2 = this;

                var func = function func() {
                    fn.call(scope, eventName, fn, scope);
                    _this2.off(eventName, func);
                };
                return this.on(eventName, func, scope);
            }
        }]);
        return Events;
    }();

    var ESC_KEY = 27;

    var prefix = 'c-datepicker';
    var defaults$$1 = function defaults$$1() {
        return {
            default: moment().startOf('hour'),
            // allow the user to override all the classes
            // used for styling the calendar
            styles: {
                scrim: 'c-scrim',
                back: prefix + '__back',
                container: prefix + '__calendar',
                date: prefix + '__date',
                dayBody: prefix + '__days-body',
                dayBodyElem: prefix + '__day-body',
                dayConcealed: prefix + '__day--concealed',
                dayDisabled: prefix + '__day--disabled',
                dayHead: prefix + '__days-head',
                dayHeadElem: prefix + '__day-head',
                dayRow: prefix + '__days-row',
                dayTable: prefix + '__days',
                month: prefix + '__month',
                next: prefix + '__next',
                positioned: prefix + '--fixed',
                selectedDay: prefix + '__day--selected',
                clockNum: prefix + '__clock__num'
            },
            // format to display in the input, or set on the element
            format: 'DD/MM/YY',
            // the container to append the picker
            container: document.body,
            // allow any dates
            dateValidator: undefined
        };
    };

    var DateTimePicker = function (_Events) {
        inherits(DateTimePicker, _Events);

        function DateTimePicker() {
            var options = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : {};
            classCallCheck(this, DateTimePicker);

            var _this = possibleConstructorReturn(this, (DateTimePicker.__proto__ || Object.getPrototypeOf(DateTimePicker)).call(this));

            var styles = Object.assign(defaults$$1().styles, options.styles);
            _this.options = Object.assign(defaults$$1(), options);
            _this.options.styles = styles;

            // listen to any event
            _this.on('*', function (evtName, evtData) {
                if (_this.options.el) {
                    // if there is a custom element, fire a real dom
                    // event on that now
                    var event = new CustomEvent(evtName, _this, evtData);
                    _this.options.el.dispatchEvent(event);
                }
            });
            return _this;
        }

        // intialize the rom calendar with our default date and
        // style options


        createClass(DateTimePicker, [{
            key: 'initializeRome',
            value: function initializeRome(container, validator) {
                var onData = this.onChangeDate.bind(this);

                return rome(container, {
                    styles: this.options.styles,
                    time: false,
                    dateValidator: validator,
                    initialValue: this.value
                }).on('data', onData);
            }

            // called to open the picker

        }, {
            key: 'open',
            value: function open() {
                var scrimEl = scrimTemplate(this.options);
                _appendTemplate(document.body, scrimEl);
                _appendTemplate(this.options.container, popupTemplate());
                this.pickerEl = this.options.container.querySelector('.' + prefix);
                this.scrimEl = document.body.querySelector('.' + this.options.styles.scrim);

                if (!this.value) {
                    // TODO hack
                    // set/setDate/setTime need refactoring to have single concerns
                    // (set: set the value; setDate/setTime rename to renderDate/renderTime
                    //  and deal with updating the view only).
                    // For now this allows us to set the default time using the same quantize
                    // rules as setting the date explicitly. Setting this.value meets setTime|Date's
                    // expectation that we have a value, and `0` guarantees that we will detect
                    this.value = moment(0);
                    this.setDate(this.options.default);
                } else {
                    this.setDate(this.value);
                }

                this.initializeRome(this.$('.' + this.options.styles.container), this.options.dateValidator);
                this._listenForCloseEvents();

                this._show();
            }
        }, {
            key: 'close',
            value: function close() {
                this._stopListeningForCloseEvents();
                this._hide();
            }
        }, {
            key: '_hide',
            value: function _hide() {
                var _this2 = this;

                this.pickerEl.classList.remove('open');
                window.setTimeout(function () {
                    _this2.options.container.removeChild(_this2.pickerEl);
                    document.body.removeChild(_this2.scrimEl);
                    _this2.trigger('close');
                }, 200);
                return this;
            }
        }, {
            key: '_show',
            value: function _show() {
                var _this3 = this;

                this.delegateEvents();
                // add the animation classes on the next animation tick
                // so that they actually work
                window.requestAnimationFrame(function () {
                    _this3.scrimEl.classList.add(_this3.options.styles.scrim + '--shown');
                    _this3.pickerEl.classList.add(prefix + '--open');
                    _this3.trigger('open');
                });
                return this;
            }
        }, {
            key: '_listenForCloseEvents',
            value: function _listenForCloseEvents() {
                var _this4 = this;

                this._onWindowKeypress = function (e) {
                    if (e.which === ESC_KEY) {
                        _this4.close();
                    }
                };

                window.addEventListener("keydown", this._onWindowKeypress);
            }
        }, {
            key: '_stopListeningForCloseEvents',
            value: function _stopListeningForCloseEvents() {
                window.removeEventListener("keydown", this._onWindowKeypress);
                this._closeHandler = null;
            }
        }, {
            key: 'delegateEvents',
            value: function delegateEvents() {
                var _this5 = this;

                this.$('.js-cancel').addEventListener('click', function () {
                    return _this5.clickCancel();
                }, false);

                this.$('.js-ok').addEventListener('click', function () {
                    return _this5.clickSubmit();
                }, false);

                this.scrimEl.addEventListener('click', function () {
                    return _this5.close();
                }, false);

                return this;
            }
        }, {
            key: 'clickSubmit',
            value: function clickSubmit() {
                this.close();
                this.trigger('submit', this.value, this);
                return this;
            }
        }, {
            key: 'clickCancel',
            value: function clickCancel() {
                this.close();
                this.trigger('cancel', this.value, this);
                return this;
            }
        }, {
            key: 'onChangeDate',
            value: function onChangeDate(dateString) {
                var newValue = moment(this.value);

                var _dateString$split = dateString.split('-'),
                    _dateString$split2 = slicedToArray(_dateString$split, 3),
                    year = _dateString$split2[0],
                    month = _dateString$split2[1],
                    date = _dateString$split2[2];

                newValue.set({year: year, month: month - 1, date: date});

                this.set(newValue);
                return this;
            }
        }, {
            key: 'data',
            value: function data(val) {
                console.warn('MaterialDatetimePicker#data is deprecated and will be removed in a future release. Please use get or set.');
                return val ? this.set(val) : this.value;
            }
        }, {
            key: 'get',
            value: function get$$1() {
                return moment(this.value);
            }

            // update the picker's date/time value
            // value: moment
            // silent: if true, do not fire any events on change

        }, {
            key: 'set',
            value: function set$$1(value) {
                var _ref = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : {},
                    _ref$silent = _ref.silent,
                    silent = _ref$silent === undefined ? false : _ref$silent;

                var m = moment(value);

                // maintain a list of change events to fire all at once later
                var evts = [];
                if (m.date() !== this.value.date() || m.month() !== this.value.month() || m.year() !== this.value.year()) {
                    this.setDate(m);
                    evts.push('change:date');
                }

                if (m.hour() !== this.value.hour() || m.minutes() !== this.value.minutes()) {
                    this.setTime(m);
                    evts.push('change:time');
                }

                if (this.options.el) {
                    // if there is an element to fire events on
                    if (this.options.el.tagName === 'INPUT') {
                        // and it is an input element then set the value
                        this.options.el.value = m.format(this.options.format);
                    } else {
                        // or any other element set a data-value attribute
                        this.options.el.setAttribute('data-value', m.format(this.options.format));
                    }
                }
                if (evts.length > 0 && !silent) {
                    // fire all the events we've collected
                    this.trigger(['change'].concat(evts).join(' '), this.value, this);
                }
            }

            // set the value and header elements to `date`
            // the calendar will be updated automatically
            // by rome when clicked

        }, {
            key: 'setDate',
            value: function setDate(date) {
                var m = moment(date);
                var month = m.format('MMM');
                var day = m.format('Do');
                var dayOfWeek = m.format('dddd');
                var year = m.format('YYYY');

                this.$('.js-day').innerText = dayOfWeek;
                this.$('.js-date-month').innerText = day + ' of ' + month + ' ' + year;

                this.value.year(m.year());
                this.value.month(m.month());
                this.value.date(m.date());
                return this;
            }

            // set the value and header elements to `time`
            // also update the hands of the clock

        }, {
            key: '$',
            value: function $(selector) {
                var els = this.pickerEl.querySelectorAll(selector);
                return els.length > 1 ? [].concat(toConsumableArray(els)) : els[0];
            }
        }]);
        return DateTimePicker;
    }(Events);

    function _appendTemplate(parent, template) {
        var tempEl = document.createElement('div');
        tempEl.innerHTML = template.trim();
        parent.appendChild(tempEl.firstChild);
        return this;
    }

    return DateTimePicker;

})));
/**
 * MDL Select
 */
(function () {
    function whenLoaded() {
        getmdlSelect.init('.getmdl-select');
    };

    window.addEventListener ?
        window.addEventListener("load", whenLoaded, false) :
        window.attachEvent && window.attachEvent("onload", whenLoaded);

}());

var getmdlSelect = {
    _addEventListeners: function (dropdown) {
        var input = dropdown.querySelector('input');
        var hiddenInput = dropdown.querySelector('input[type="hidden"]');
        var list = dropdown.querySelectorAll('li');
        var menu = dropdown.querySelector('.mdl-js-menu');
        var arrow = dropdown.querySelector('.mdl-icon-toggle__label');
        var label = '';
        var previousValue = '';
        var previousDataVal = '';
        var opened = false;

        var setSelectedItem = function (li) {
            var value = li.textContent.trim();
            input.value = value;
            list.forEach(function (li) {
                li.classList.remove('selected');
            });
            li.classList.add('selected');
            dropdown.MaterialTextfield.change(value); // handles css class changes
            setTimeout(function () {
                dropdown.MaterialTextfield.updateClasses_(); //update css class
            }, 250);

            // update input with the "id" value
            hiddenInput.value = li.dataset.val || '';

            previousValue = input.value;
            previousDataVal = hiddenInput.value;

            if ("createEvent" in document) {
                var evt = document.createEvent("HTMLEvents");
                evt.initEvent("change", false, true);
                menu['MaterialMenu'].hide();
                input.dispatchEvent(evt);
            } else {
                input.fireEvent("onchange");
            }
        }

        var hideAllMenus = function () {
            opened = false;
            input.value = previousValue;
            hiddenInput.value = previousDataVal;
            if (!dropdown.querySelector('.mdl-menu__container').classList.contains('is-visible')) {
                dropdown.classList.remove('is-focused');
            }
            var menus = document.querySelectorAll('.getmdl-select .mdl-js-menu');
            [].forEach.call(menus, function (menu) {
                menu['MaterialMenu'].hide();
            });
            var event = new Event('closeSelect');
            menu.dispatchEvent(event);
        };
        document.body.addEventListener('click', hideAllMenus, false);

        //hide previous select after press TAB
        dropdown.onkeydown = function (event) {
            if (event.keyCode == 9) {
                input.value = previousValue;
                hiddenInput.value = previousDataVal;
                menu['MaterialMenu'].hide();
                dropdown.classList.remove('is-focused');
            }
        };

        //show select if it have focus
        input.onfocus = function (e) {
            menu['MaterialMenu'].show();
            menu.focus();
            opened = true;
        };

        input.onblur = function (e) {
            e.stopPropagation();
        };

        //hide all old opened selects and opening just clicked select
        input.onclick = function (e) {
            e.stopPropagation();
            if (!menu.classList.contains('is-visible')) {
                menu['MaterialMenu'].show();
                hideAllMenus();
                dropdown.classList.add('is-focused');
                opened = true;
            } else {
                menu['MaterialMenu'].hide();
                opened = false;
            }
        };

        input.onkeydown = function (event) {
            if (event.keyCode == 27) {
                input.value = previousValue;
                hiddenInput.value = previousDataVal;
                menu['MaterialMenu'].hide();
                dropdown.MaterialTextfield.onBlur_();
                if (label !== '') {
                    dropdown.querySelector('.mdl-textfield__label').textContent = label;
                    label = '';
                }
            }
        };

        menu.addEventListener('closeSelect', function (e) {
            input.value = previousValue;
            hiddenInput.value = previousDataVal;
            dropdown.classList.remove('is-focused');
            if (label !== '') {
                dropdown.querySelector('.mdl-textfield__label').textContent = label;
                label = '';
            }
        });

        //set previous value and data-val if ESC was pressed
        menu.onkeydown = function (event) {
            if (event.keyCode == 27) {
                input.value = previousValue;
                hiddenInput.value = previousDataVal;
                dropdown.classList.remove('is-focused');
                if (label !== '') {
                    dropdown.querySelector('.mdl-textfield__label').textContent = label;
                    label = '';
                }
            }
        };

        if (arrow) {
            arrow.onclick = function (e) {
                e.stopPropagation();
                if (opened) {
                    menu['MaterialMenu'].hide();
                    opened = false;
                    dropdown.classList.remove('is-focused');
                    dropdown.MaterialTextfield.onBlur_();
                    input.value = previousValue;
                    hiddenInput.value = previousDataVal;
                } else {
                    hideAllMenus();
                    dropdown.MaterialTextfield.onFocus_();
                    input.focus();
                    menu['MaterialMenu'].show();
                    opened = true;
                }
            };
        }

        [].forEach.call(list, function (li) {
            li.onfocus = function () {
                dropdown.classList.add('is-focused');
                var value = li.textContent.trim();
                input.value = value;
                if (!dropdown.classList.contains('mdl-textfield--floating-label') && label == '') {
                    label = dropdown.querySelector('.mdl-textfield__label').textContent.trim();
                    dropdown.querySelector('.mdl-textfield__label').textContent = '';
                }
            };

            li.onclick = function () {
                setSelectedItem(li);
            };

            if (li.dataset.selected) {
                setSelectedItem(li);
            }
        });
    },
    init: function (selector) {
        var dropdowns = document.querySelectorAll(selector);
        [].forEach.call(dropdowns, function (dropdown) {
            getmdlSelect._addEventListeners(dropdown);
            componentHandler.upgradeElement(dropdown);
            componentHandler.upgradeElement(dropdown.querySelector('ul'));
        });
    }
};