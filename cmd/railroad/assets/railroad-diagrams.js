"use strict";
/*
Railroad Diagrams
by Tab Atkins Jr. (and others)
http://xanthir.com
http://twitter.com/tabatkins
http://github.com/tabatkins/railroad-diagrams

This document and all associated files in the github project are licensed under CC0: http://creativecommons.org/publicdomain/zero/1.0/
This means you can reuse, remix, or otherwise appropriate this project for your own use WITHOUT RESTRICTION.
(The actual legal meaning can be found at the above link.)
Don't ask me for permission to use any part of this project, JUST USE IT.
I would appreciate attribution, but that is not required by the license.
*/

/*
This file uses a module pattern to avoid leaking names into the global scope.
Should be compatible with AMD, CommonJS, and plain ol' browser JS.

As well, several configuration constants are passed into the module function at the bottom of this file.
At runtime, these constants can be found on the Diagram class,
and can be changed before creating a Diagram.
*/

(function(options) {
	var funcs = {};

	function subclassOf(baseClass, superClass) {
		baseClass.prototype = Object.create(superClass.prototype);
		baseClass.prototype.$super = superClass.prototype;
	}

	function unnull(/* children */) {
		return [].slice.call(arguments).reduce(function(sofar, x) { return sofar !== undefined ? sofar : x; });
	}

	function determineGaps(outer, inner) {
		var diff = outer - inner;
		switch(Diagram.INTERNAL_ALIGNMENT) {
			case 'left': return [0, diff]; break;
			case 'right': return [diff, 0]; break;
			case 'center':
			default: return [diff/2, diff/2]; break;
		}
	}

	function wrapString(value) {
		return value instanceof FakeSVG ? value : new Terminal(""+value);
	}

	function sum(iter, func) {
		if(!func) func = function(x) { return x; };
		return iter.map(func).reduce(function(a,b){return a+b}, 0);
	}

	function max(iter, func) {
		if(!func) func = function(x) { return x; };
		return Math.max.apply(null, iter.map(func));
	}

	function* enumerate(iter) {
		var count = 0;
		for(const x of iter) {
			yield [count, x];
			count++;
		}
	}

	var SVG = funcs.SVG = function SVG(name, attrs, text) {
		attrs = attrs || {};
		text = text || '';
		var el = document.createElementNS("http://www.w3.org/2000/svg",name);
		for(var attr in attrs) {
			if(attr === 'xlink:href')
				el.setAttributeNS("http://www.w3.org/1999/xlink", 'href', attrs[attr]);
			else
				el.setAttribute(attr, attrs[attr]);
		}
		el.textContent = text;
		return el;
	}

	var FakeSVG = funcs.FakeSVG = function FakeSVG(tagName, attrs, text){
		if(!(this instanceof FakeSVG)) return new FakeSVG(tagName, attrs, text);
		if(text) this.children = text;
		else this.children = [];
		this.tagName = tagName;
		this.attrs = unnull(attrs, {});
		return this;
	};
	FakeSVG.prototype.format = function(x, y, width) {
		// Virtual
	};
	FakeSVG.prototype.addTo = function(parent) {
		if(parent instanceof FakeSVG) {
			parent.children.push(this);
			return this;
		} else {
			var svg = this.toSVG();
			parent.appendChild(svg);
			return svg;
		}
	};
	FakeSVG.prototype.escapeString = function(string) {
		// Escape markdown and HTML special characters
		return string.replace(/[*_\`\[\]<&]/g, function(charString) {
			return '&#' + charString.charCodeAt(0) + ';';
		});
	};
	FakeSVG.prototype.toSVG = function() {
		var el = SVG(this.tagName, this.attrs);
		if(typeof this.children == 'string') {
			el.textContent = this.children;
		} else {
			this.children.forEach(function(e) {
				el.appendChild(e.toSVG());
			});
		}
		return el;
	};
	FakeSVG.prototype.toString = function() {
		var str = '<' + this.tagName;
		var group = this.tagName == "g" || this.tagName == "svg";
		for(var attr in this.attrs) {
			str += ' ' + attr + '="' + (this.attrs[attr]+'').replace(/&/g, '&amp;').replace(/"/g, '&quot;') + '"';
		}
		str += '>';
		if(group) str += "\n";
		if(typeof this.children == 'string') {
			str += FakeSVG.prototype.escapeString(this.children);
		} else {
			this.children.forEach(function(e) {
				str += e;
			});
		}
		str += '</' + this.tagName + '>\n';
		return str;
	}
	FakeSVG.prototype.walk = function(cb) {
		cb(this);
	}

	var Path = funcs.Path = function Path(x,y) {
		if(!(this instanceof Path)) return new Path(x,y);
		FakeSVG.call(this, 'path');
		this.attrs.d = "M"+x+' '+y;
	}
	subclassOf(Path, FakeSVG);
	Path.prototype.m = function(x,y) {
		this.attrs.d += 'm'+x+' '+y;
		return this;
	}
	Path.prototype.h = function(val) {
		this.attrs.d += 'h'+val;
		return this;
	}
	Path.prototype.right = function(val) { return this.h(Math.max(0, val)); }
	Path.prototype.left = function(val) { return this.h(-Math.max(0, val)); }
	Path.prototype.v = function(val) {
		this.attrs.d += 'v'+val;
		return this;
	}
	Path.prototype.down = function(val) { return this.v(Math.max(0, val)); }
	Path.prototype.up = function(val) { return this.v(-Math.max(0, val)); }
	Path.prototype.arc = function(sweep){
		// 1/4 of a circle
		var x = Diagram.ARC_RADIUS;
		var y = Diagram.ARC_RADIUS;
		if(sweep[0] == 'e' || sweep[1] == 'w') {
			x *= -1;
		}
		if(sweep[0] == 's' || sweep[1] == 'n') {
			y *= -1;
		}
		if(sweep == 'ne' || sweep == 'es' || sweep == 'sw' || sweep == 'wn') {
			var cw = 1;
		} else {
			var cw = 0;
		}
		this.attrs.d += "a"+Diagram.ARC_RADIUS+" "+Diagram.ARC_RADIUS+" 0 0 "+cw+' '+x+' '+y;
		return this;
	}
	Path.prototype.arc_8 = function(start, dir) {
		// 1/8 of a circle
		const arc = Diagram.ARC_RADIUS;
		const s2 = 1/Math.sqrt(2) * arc;
		const s2inv = (arc - s2);
		let path = "a " + arc + " " + arc + " 0 0 " + (dir=='cw' ? "1" : "0") + " ";
		const sd = start+dir;
		const offset =
			sd == 'ncw'   ? [s2, s2inv] :
			sd == 'necw'  ? [s2inv, s2] :
			sd == 'ecw'   ? [-s2inv, s2] :
			sd == 'secw'  ? [-s2, s2inv] :
			sd == 'scw'   ? [-s2, -s2inv] :
			sd == 'swcw'  ? [-s2inv, -s2] :
			sd == 'wcw'   ? [s2inv, -s2] :
			sd == 'nwcw'  ? [s2, -s2inv] :
			sd == 'nccw'  ? [-s2, s2inv] :
			sd == 'nwccw' ? [-s2inv, s2] :
			sd == 'wccw'  ? [s2inv, s2] :
			sd == 'swccw' ? [s2, s2inv] :
			sd == 'sccw'  ? [s2, -s2inv] :
			sd == 'seccw' ? [s2inv, -s2] :
			sd == 'eccw'  ? [-s2inv, -s2] :
			sd == 'neccw' ? [-s2, -s2inv] : null
		;
		path += offset.join(" ");
		this.attrs.d += path;
		return this;
	}
	Path.prototype.l = function(x, y) {
		this.attrs.d += 'l'+x+' '+y;
		return this;
	}
	Path.prototype.format = function() {
		// All paths in this library start/end horizontally.
		// The extra .5 ensures a minor overlap, so there's no seams in bad rasterizers.
		this.attrs.d += 'h.5';
		return this;
	}


	var DiagramMultiContainer = funcs.DiagramMultiContainer = function DiagramMultiContainer(tagName, items, attrs, text) {
		FakeSVG.call(this, tagName, attrs, text);
		this.items = items.map(wrapString);
	}
	subclassOf(DiagramMultiContainer, FakeSVG);
	DiagramMultiContainer.prototype.walk = function(cb) {
		cb(this);
		this.items.forEach(x=>w.walk(cb));
	}


	var Diagram = funcs.Diagram = function Diagram(items) {
		if(!(this instanceof Diagram)) return new Diagram([].slice.call(arguments));
		DiagramMultiContainer.call(this, 'svg', items, {class: Diagram.DIAGRAM_CLASS});
		if(!(this.items[0] instanceof Start)) {
			this.items.unshift(new Start());
		}
		if(!(this.items[this.items.length-1] instanceof End)) {
			this.items.push(new End());
		}
		this.up = this.down = this.height = this.width = 0;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			this.width += item.width + (item.needsSpace?20:0);
			this.up = Math.max(this.up, item.up - this.height);
			this.height += item.height;
			this.down = Math.max(this.down - item.height, item.down);
		}
		this.formatted = false;
	}
	subclassOf(Diagram, DiagramMultiContainer);
	for(var option in options) {
		Diagram[option] = options[option];
	}
	Diagram.prototype.format = function(paddingt, paddingr, paddingb, paddingl) {
		paddingt = unnull(paddingt, 20);
		paddingr = unnull(paddingr, paddingt, 20);
		paddingb = unnull(paddingb, paddingt, 20);
		paddingl = unnull(paddingl, paddingr, 20);
		var x = paddingl;
		var y = paddingt;
		y += this.up;
		var g = FakeSVG('g', Diagram.STROKE_ODD_PIXEL_LENGTH ? {transform:'translate(.5 .5)'} : {});
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			if(item.needsSpace) {
				Path(x,y).h(10).addTo(g);
				x += 10;
			}
			item.format(x, y, item.width).addTo(g);
			x += item.width;
			y += item.height;
			if(item.needsSpace) {
				Path(x,y).h(10).addTo(g);
				x += 10;
			}
		}
		this.attrs.width = this.width + paddingl + paddingr;
		this.attrs.height = this.up + this.height + this.down + paddingt + paddingb;
		this.attrs.viewBox = "0 0 " + this.attrs.width + " " + this.attrs.height;
		g.addTo(this);
		this.formatted = true;
		return this;
	}
	Diagram.prototype.addTo = function(parent) {
		if(!parent) {
			var scriptTag = document.getElementsByTagName('script');
			scriptTag = scriptTag[scriptTag.length - 1];
			parent = scriptTag.parentNode;
		}
		return this.$super.addTo.call(this, parent);
	}
	Diagram.prototype.toSVG = function() {
		if (!this.formatted) {
			this.format();
		}
		return this.$super.toSVG.call(this);
	}
	Diagram.prototype.toString = function() {
		if (!this.formatted) {
			this.format();
		}
		return this.$super.toString.call(this);
	}
	Diagram.DEBUG = false;

	var ComplexDiagram = funcs.ComplexDiagram = function ComplexDiagram() {
		var diagram = new Diagram([].slice.call(arguments));
		var items = diagram.items;
		items.shift();
		items.pop();
		items.unshift(new Start({type:"complex"}));
		items.push(new End({type:"complex"}));
		diagram.items = items;
		return diagram;
	}

	var Sequence = funcs.Sequence = function Sequence(items) {
		if(!(this instanceof Sequence)) return new Sequence([].slice.call(arguments));
		DiagramMultiContainer.call(this, 'g', items);
		var numberOfItems = this.items.length;
		this.needsSpace = true;
		this.up = this.down = this.height = this.width = 0;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			this.width += item.width + (item.needsSpace?20:0);
			this.up = Math.max(this.up, item.up - this.height);
			this.height += item.height;
			this.down = Math.max(this.down - item.height, item.down);
		}
		if(this.items[0].needsSpace) this.width -= 10;
		if(this.items[this.items.length-1].needsSpace) this.width -= 10;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "sequence"
		}
	}
	subclassOf(Sequence, DiagramMultiContainer);
	Sequence.prototype.format = function(x,y,width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y+this.height).h(gaps[1]).addTo(this);
		x += gaps[0];

		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			if(item.needsSpace && i > 0) {
				Path(x,y).h(10).addTo(this);
				x += 10;
			}
			item.format(x, y, item.width).addTo(this);
			x += item.width;
			y += item.height;
			if(item.needsSpace && i < this.items.length-1) {
				Path(x,y).h(10).addTo(this);
				x += 10;
			}
		}
		return this;
	}

	var Stack = funcs.Stack = function Stack(items) {
		if(!(this instanceof Stack)) return new Stack([].slice.call(arguments));
		DiagramMultiContainer.call(this, 'g', items);
		if( items.length === 0 ) {
			throw new RangeError("Stack() must have at least one child.");
		}
		this.width = Math.max.apply(null, this.items.map(function(e) { return e.width + (e.needsSpace?20:0); }));
		//if(this.items[0].needsSpace) this.width -= 10;
		//if(this.items[this.items.length-1].needsSpace) this.width -= 10;
		if(this.items.length > 1){
			this.width += Diagram.ARC_RADIUS*2;
		}
		this.needsSpace = true;
		this.up = this.items[0].up;
		this.down = this.items[this.items.length-1].down;

		this.height = 0;
		var last = this.items.length - 1;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			this.height += item.height;
			if(i > 0) {
				this.height += Math.max(Diagram.ARC_RADIUS*2, item.up + Diagram.VERTICAL_SEPARATION);
			}
			if(i < last) {
				this.height += Math.max(Diagram.ARC_RADIUS*2, item.down + Diagram.VERTICAL_SEPARATION);
			}
		}
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "stack"
		}
	}
	subclassOf(Stack, DiagramMultiContainer);
	Stack.prototype.format = function(x,y,width) {
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		x += gaps[0];
		var xInitial = x;
		if(this.items.length > 1) {
			Path(x, y).h(Diagram.ARC_RADIUS).addTo(this);
			x += Diagram.ARC_RADIUS;
		}

		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			var innerWidth = this.width - (this.items.length>1 ? Diagram.ARC_RADIUS*2 : 0);
			item.format(x, y, innerWidth).addTo(this);
			x += innerWidth;
			y += item.height;

			if(i !== this.items.length-1) {
				Path(x, y)
					.arc('ne').down(Math.max(0, item.down + Diagram.VERTICAL_SEPARATION - Diagram.ARC_RADIUS*2))
					.arc('es').left(innerWidth)
					.arc('nw').down(Math.max(0, this.items[i+1].up + Diagram.VERTICAL_SEPARATION - Diagram.ARC_RADIUS*2))
					.arc('ws').addTo(this);
				y += Math.max(item.down + Diagram.VERTICAL_SEPARATION, Diagram.ARC_RADIUS*2) + Math.max(this.items[i+1].up + Diagram.VERTICAL_SEPARATION, Diagram.ARC_RADIUS*2);
				//y += Math.max(Diagram.ARC_RADIUS*4, item.down + Diagram.VERTICAL_SEPARATION*2 + this.items[i+1].up)
				x = xInitial+Diagram.ARC_RADIUS;
			}

		}

		if(this.items.length > 1) {
			Path(x,y).h(Diagram.ARC_RADIUS).addTo(this);
			x += Diagram.ARC_RADIUS;
		}
		Path(x,y).h(gaps[1]).addTo(this);

		return this;
	}

	var OptionalSequence = funcs.OptionalSequence = function OptionalSequence(items) {
		if(!(this instanceof OptionalSequence)) return new OptionalSequence([].slice.call(arguments));
		DiagramMultiContainer.call(this, 'g', items);
		if( items.length === 0 ) {
			throw new RangeError("OptionalSequence() must have at least one child.");
		}
		if( items.length === 1 ) {
			return new Sequence(items);
		}
		var arc = Diagram.ARC_RADIUS;
		this.needsSpace = false;
		this.width = 0;
		this.up = 0;
		this.height = sum(this.items, function(x){return x.height});
		this.down = this.items[0].down;
		var heightSoFar = 0;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			this.up = Math.max(this.up, Math.max(arc*2, item.up + Diagram.VERTICAL_SEPARATION) - heightSoFar);
			heightSoFar += item.height;
			if(i > 0) {
				this.down = Math.max(this.height + this.down, heightSoFar + Math.max(arc*2, item.down + Diagram.VERTICAL_SEPARATION)) - this.height;
			}
			var itemWidth = (item.needsSpace?10:0) + item.width;
			if(i == 0) {
				this.width += arc + Math.max(itemWidth, arc);
			} else {
				this.width += arc*2 + Math.max(itemWidth, arc) + arc;
			}
		}
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "optseq"
		}
	}
	subclassOf(OptionalSequence, DiagramMultiContainer);
	OptionalSequence.prototype.format = function(x, y, width) {
		var arc = Diagram.ARC_RADIUS;
		var gaps = determineGaps(width, this.width);
		Path(x, y).right(gaps[0]).addTo(this);
		Path(x + gaps[0] + this.width, y + this.height).right(gaps[1]).addTo(this);
		x += gaps[0]
		var upperLineY = y - this.up;
		var last = this.items.length - 1;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			var itemSpace = (item.needsSpace?10:0);
			var itemWidth = item.width + itemSpace;
			if(i == 0) {
				// Upper skip
				Path(x,y)
					.arc('se')
					.up(y - upperLineY - arc*2)
					.arc('wn')
					.right(itemWidth - arc)
					.arc('ne')
					.down(y + item.height - upperLineY - arc*2)
					.arc('ws')
					.addTo(this);
				// Straight line
				Path(x, y)
					.right(itemSpace + arc)
					.addTo(this);
				item.format(x + itemSpace + arc, y, item.width).addTo(this);
				x += itemWidth + arc;
				y += item.height;
				// x ends on the far side of the first element,
				// where the next element's skip needs to begin
			} else if(i < last) {
				// Upper skip
				Path(x, upperLineY)
					.right(arc*2 + Math.max(itemWidth, arc) + arc)
					.arc('ne')
					.down(y - upperLineY + item.height - arc*2)
					.arc('ws')
					.addTo(this);
				// Straight line
				Path(x,y)
					.right(arc*2)
					.addTo(this);
				item.format(x + arc*2, y, item.width).addTo(this);
				Path(x + item.width + arc*2, y + item.height)
					.right(itemSpace + arc)
					.addTo(this);
				// Lower skip
				Path(x,y)
					.arc('ne')
					.down(item.height + Math.max(item.down + Diagram.VERTICAL_SEPARATION, arc*2) - arc*2)
					.arc('ws')
					.right(itemWidth - arc)
					.arc('se')
					.up(item.down + Diagram.VERTICAL_SEPARATION - arc*2)
					.arc('wn')
					.addTo(this);
				x += arc*2 + Math.max(itemWidth, arc) + arc;
				y += item.height;
			} else {
				// Straight line
				Path(x, y)
					.right(arc*2)
					.addTo(this);
				item.format(x + arc*2, y, item.width).addTo(this);
				Path(x + arc*2 + item.width, y + item.height)
					.right(itemSpace + arc)
					.addTo(this);
				// Lower skip
				Path(x,y)
					.arc('ne')
					.down(item.height + Math.max(item.down + Diagram.VERTICAL_SEPARATION, arc*2) - arc*2)
					.arc('ws')
					.right(itemWidth - arc)
					.arc('se')
					.up(item.down + Diagram.VERTICAL_SEPARATION - arc*2)
					.arc('wn')
					.addTo(this);
			}
		}
		return this;
	}

	var AlternatingSequence = funcs.AlternatingSequence = function AlternatingSequence(items) {
		if(!(this instanceof AlternatingSequence)) return new AlternatingSequence([].slice.call(arguments));
		DiagramMultiContainer.call(this, 'g', items);
		if( items.length === 1 ) {
			return new Sequence(items);
		}
		if( items.length !== 2 ) {
			throw new RangeError("AlternatingSequence() must have one or two children.");
		}
		this.needsSpace = false;

		const arc = Diagram.ARC_RADIUS;
		const vert = Diagram.VERTICAL_SEPARATION;
		const max = Math.max;
		const first = this.items[0];
		const second = this.items[1];

		const arcX = 1 / Math.sqrt(2) * arc * 2;
		const arcY = (1 - 1 / Math.sqrt(2)) * arc * 2;
		const crossY = Math.max(arc, Diagram.VERTICAL_SEPARATION);
		const crossX = (crossY - arcY) + arcX;

		const firstOut = max(arc + arc, crossY/2 + arc + arc, crossY/2 + vert + first.down);
		this.up = firstOut + first.height + first.up;

		const secondIn = max(arc + arc, crossY/2 + arc + arc, crossY/2 + vert + second.up);
		this.down = secondIn + second.height + second.down;

		this.height = 0;

		const firstWidth = 2*(first.needsSpace?10:0) + first.width;
		const secondWidth = 2*(second.needsSpace?10:0) + second.width;
		this.width = 2*arc + max(firstWidth, crossX, secondWidth) + 2*arc;

		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "altseq"
		}
	}
	subclassOf(AlternatingSequence, DiagramMultiContainer);
	AlternatingSequence.prototype.format = function(x, y, width) {
		const arc = Diagram.ARC_RADIUS;
		const gaps = determineGaps(width, this.width);
		Path(x,y).right(gaps[0]).addTo(this);
		console.log(gaps);
		x += gaps[0];
		Path(x+this.width, y).right(gaps[1]).addTo(this);
		// bounding box
		//Path(x+gaps[0], y).up(this.up).right(this.width).down(this.up+this.down).left(this.width).up(this.down).addTo(this);
		const first = this.items[0];
		const second = this.items[1];

		// top
		const firstIn = this.up - first.up;
		const firstOut = this.up - first.up - first.height;
		Path(x,y).arc('se').up(firstIn-2*arc).arc('wn').addTo(this);
		first.format(x + 2*arc, y - firstIn, this.width - 4*arc).addTo(this);
		Path(x + this.width - 2*arc, y - firstOut).arc('ne').down(firstOut - 2*arc).arc('ws').addTo(this);

		// bottom
		const secondIn = this.down - second.down - second.height;
		const secondOut = this.down - second.down;
		Path(x,y).arc('ne').down(secondIn - 2*arc).arc('ws').addTo(this);
		second.format(x + 2*arc, y + secondIn, this.width - 4*arc).addTo(this);
		Path(x + this.width - 2*arc, y + secondOut).arc('se').up(secondOut - 2*arc).arc('wn').addTo(this);

		// crossover
		const arcX = 1 / Math.sqrt(2) * arc * 2;
		const arcY = (1 - 1 / Math.sqrt(2)) * arc * 2;
		const crossY = Math.max(arc, Diagram.VERTICAL_SEPARATION);
		const crossX = (crossY - arcY) + arcX;
		const crossBar = (this.width - 4*arc - crossX)/2;
		Path(x+arc, y - crossY/2 - arc).arc('ws').right(crossBar)
			.arc_8('n', 'cw').l(crossX - arcX, crossY - arcY).arc_8('sw', 'ccw')
			.right(crossBar).arc('ne').addTo(this);
		Path(x+arc, y + crossY/2 + arc).arc('wn').right(crossBar)
			.arc_8('s', 'ccw').l(crossX - arcX, -(crossY - arcY)).arc_8('nw', 'cw')
			.right(crossBar).arc('se').addTo(this);

		return this;
	}

	var Choice = funcs.Choice = function Choice(normal, items) {
		if(!(this instanceof Choice)) return new Choice(normal, [].slice.call(arguments,1));
		DiagramMultiContainer.call(this, 'g', items);
		if( typeof normal !== "number" || normal !== Math.floor(normal) ) {
			throw new TypeError("The first argument of Choice() must be an integer.");
		} else if(normal < 0 || normal >= items.length) {
			throw new RangeError("The first argument of Choice() must be an index for one of the items.");
		} else {
			this.normal = normal;
		}
		var first = 0;
		var last = items.length - 1;
		this.width = Math.max.apply(null, this.items.map(function(el){return el.width})) + Diagram.ARC_RADIUS*4;
		this.height = this.items[normal].height;
		this.up = this.items[first].up;
		for(var i = first; i < normal; i++) {
			if(i == normal-1) var arcs = Diagram.ARC_RADIUS*2;
			else var arcs = Diagram.ARC_RADIUS;
			this.up += Math.max(arcs, this.items[i].height + this.items[i].down + Diagram.VERTICAL_SEPARATION + this.items[i+1].up);
		}
		this.down = this.items[last].down;
		for(var i = normal+1; i <= last; i++) {
			if(i == normal+1) var arcs = Diagram.ARC_RADIUS*2;
			else var arcs = Diagram.ARC_RADIUS;
			this.down += Math.max(arcs, this.items[i-1].height + this.items[i-1].down + Diagram.VERTICAL_SEPARATION + this.items[i].up);
		}
		this.down -= this.items[normal].height; // already counted in Choice.height
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "choice"
		}
	}
	subclassOf(Choice, DiagramMultiContainer);
	Choice.prototype.format = function(x,y,width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y+this.height).h(gaps[1]).addTo(this);
		x += gaps[0];

		var last = this.items.length -1;
		var innerWidth = this.width - Diagram.ARC_RADIUS*4;

		// Do the elements that curve above
		for(var i = this.normal - 1; i >= 0; i--) {
			var item = this.items[i];
			if( i == this.normal - 1 ) {
				var distanceFromY = Math.max(Diagram.ARC_RADIUS*2, this.items[this.normal].up + Diagram.VERTICAL_SEPARATION + item.down + item.height);
			}
			Path(x,y)
				.arc('se')
				.up(distanceFromY - Diagram.ARC_RADIUS*2)
				.arc('wn').addTo(this);
			item.format(x+Diagram.ARC_RADIUS*2,y - distanceFromY,innerWidth).addTo(this);
			Path(x+Diagram.ARC_RADIUS*2+innerWidth, y-distanceFromY+item.height)
				.arc('ne')
				.down(distanceFromY - item.height + this.height - Diagram.ARC_RADIUS*2)
				.arc('ws').addTo(this);
			distanceFromY += Math.max(Diagram.ARC_RADIUS, item.up + Diagram.VERTICAL_SEPARATION + (i == 0 ? 0 : this.items[i-1].down+this.items[i-1].height));
		}

		// Do the straight-line path.
		Path(x,y).right(Diagram.ARC_RADIUS*2).addTo(this);
		this.items[this.normal].format(x+Diagram.ARC_RADIUS*2, y, innerWidth).addTo(this);
		Path(x+Diagram.ARC_RADIUS*2+innerWidth, y+this.height).right(Diagram.ARC_RADIUS*2).addTo(this);

		// Do the elements that curve below
		for(var i = this.normal+1; i <= last; i++) {
			var item = this.items[i];
			if( i == this.normal + 1 ) {
				var distanceFromY = Math.max(Diagram.ARC_RADIUS*2, this.height + this.items[this.normal].down + Diagram.VERTICAL_SEPARATION + item.up);
			}
			Path(x,y)
				.arc('ne')
				.down(distanceFromY - Diagram.ARC_RADIUS*2)
				.arc('ws').addTo(this);
			item.format(x+Diagram.ARC_RADIUS*2, y+distanceFromY, innerWidth).addTo(this);
			Path(x+Diagram.ARC_RADIUS*2+innerWidth, y+distanceFromY+item.height)
				.arc('se')
				.up(distanceFromY - Diagram.ARC_RADIUS*2 + item.height - this.height)
				.arc('wn').addTo(this);
			distanceFromY += Math.max(Diagram.ARC_RADIUS, item.height + item.down + Diagram.VERTICAL_SEPARATION + (i == last ? 0 : this.items[i+1].up));
		}

		return this;
	}


	var HorizontalChoice = funcs.HorizontalChoice = function HorizontalChoice(items) {
		if(!(this instanceof HorizontalChoice)) return new HorizontalChoice([].slice.call(arguments));
		if( items.length === 0 ) {
			throw new RangeError("HorizontalChoice() must have at least one child.");
		}
		if( items.length === 1) {
			return new Sequence(items);
		}
		DiagramMultiContainer.call(this, 'g', items);

		const allButLast = this.items.slice(0, -1);
		const middles = this.items.slice(1, -1);
		const first = this.items[0];
		const last = this.items[this.items.length - 1];
		this.needsSpace = false;

		this.width = Diagram.ARC_RADIUS; // starting track
		this.width += Diagram.ARC_RADIUS*2 * (this.items.length-1); // inbetween tracks
		this.width += sum(this.items, x=>x.width + (x.needsSpace?20:0)); // items
		this.width += (last.height > 0 ? Diagram.ARC_RADIUS : 0); // needs space to curve up
		this.width += Diagram.ARC_RADIUS; //ending track

		// Always exits at entrance height
		this.height = 0;

		// All but the last have a track running above them
		this._upperTrack = Math.max(
			Diagram.ARC_RADIUS*2,
			Diagram.VERTICAL_SEPARATION,
			max(allButLast, x=>x.up) + Diagram.VERTICAL_SEPARATION
		);
		this.up = Math.max(this._upperTrack, last.up);

		// All but the first have a track running below them
		// Last either straight-lines or curves up, so has different calculation
		this._lowerTrack = Math.max(
			Diagram.VERTICAL_SEPARATION,
			max(middles, x=>x.height+Math.max(x.down+Diagram.VERTICAL_SEPARATION, Diagram.ARC_RADIUS*2)),
			last.height + last.down + Diagram.VERTICAL_SEPARATION
		);
		if(first.height < this._lowerTrack) {
			// Make sure there's at least 2*AR room between first exit and lower track
			this._lowerTrack = Math.max(this._lowerTrack, first.height + Diagram.ARC_RADIUS*2);
		}
		this.down = Math.max(this._lowerTrack, first.height + first.down);

		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "horizontalchoice"
		}
	}
	subclassOf(HorizontalChoice, DiagramMultiContainer);
	HorizontalChoice.prototype.format = function(x,y,width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		new Path(x,y).h(gaps[0]).addTo(this);
		new Path(x+gaps[0]+this.width,y+this.height).h(gaps[1]).addTo(this);
		x += gaps[0];

		const first = this.items[0];
		const last = this.items[this.items.length-1];
		const allButFirst = this.items.slice(1);
		const allButLast = this.items.slice(0, -1);

		// upper track
		var upperSpan = (sum(allButLast, x=>x.width+(x.needsSpace?20:0))
			+ (this.items.length - 2) * Diagram.ARC_RADIUS*2
			- Diagram.ARC_RADIUS
		);
		new Path(x,y)
			.arc('se')
			.v(-(this._upperTrack - Diagram.ARC_RADIUS*2))
			.arc('wn')
			.h(upperSpan)
			.addTo(this);

		// lower track
		var lowerSpan = (sum(allButFirst, x=>x.width+(x.needsSpace?20:0))
			+ (this.items.length - 2) * Diagram.ARC_RADIUS*2
			+ (last.height > 0 ? Diagram.ARC_RADIUS : 0)
			- Diagram.ARC_RADIUS
		);
		var lowerStart = x + Diagram.ARC_RADIUS + first.width+(first.needsSpace?20:0) + Diagram.ARC_RADIUS*2;
		new Path(lowerStart, y+this._lowerTrack)
			.h(lowerSpan)
			.arc('se')
			.v(-(this._lowerTrack - Diagram.ARC_RADIUS*2))
			.arc('wn')
			.addTo(this);

		// Items
		for(const [i, item] of enumerate(this.items)) {
			// input track
			if(i === 0) {
				new Path(x,y)
					.h(Diagram.ARC_RADIUS)
					.addTo(this);
				x += Diagram.ARC_RADIUS;
			} else {
				new Path(x, y - this._upperTrack)
					.arc('ne')
					.v(this._upperTrack - Diagram.ARC_RADIUS*2)
					.arc('ws')
					.addTo(this);
				x += Diagram.ARC_RADIUS*2;
			}

			// item
			var itemWidth = item.width + (item.needsSpace?20:0);
			item.format(x, y, itemWidth).addTo(this);
			x += itemWidth;

			// output track
			if(i === this.items.length-1) {
				if(item.height === 0) {
					new Path(x,y)
						.h(Diagram.ARC_RADIUS)
						.addTo(this);
				} else {
					new Path(x,y+item.height)
					.arc('se')
					.addTo(this);
				}
			} else if(i === 0 && item.height > this._lowerTrack) {
				// Needs to arc up to meet the lower track, not down.
				if(item.height - this._lowerTrack >= Diagram.ARC_RADIUS*2) {
					new Path(x, y+item.height)
						.arc('se')
						.v(this._lowerTrack - item.height + Diagram.ARC_RADIUS*2)
						.arc('wn')
						.addTo(this);
				} else {
					// Not enough space to fit two arcs
					// so just bail and draw a straight line for now.
					new Path(x, y+item.height)
						.l(Diagram.ARC_RADIUS*2, this._lowerTrack - item.height)
						.addTo(this);
				}
			} else {
				new Path(x, y+item.height)
					.arc('ne')
					.v(this._lowerTrack - item.height - Diagram.ARC_RADIUS*2)
					.arc('ws')
					.addTo(this);
			}
		}
		return this;
	}


	var MultipleChoice = funcs.MultipleChoice = function MultipleChoice(normal, type, items) {
		if(!(this instanceof MultipleChoice)) return new MultipleChoice(normal, type, [].slice.call(arguments,2));
		DiagramMultiContainer.call(this, 'g', items);
		if( typeof normal !== "number" || normal !== Math.floor(normal) ) {
			throw new TypeError("The first argument of MultipleChoice() must be an integer.");
		} else if(normal < 0 || normal >= items.length) {
			throw new RangeError("The first argument of MultipleChoice() must be an index for one of the items.");
		} else {
			this.normal = normal;
		}
		if( type != "any" && type != "all" ) {
			throw new SyntaxError("The second argument of MultipleChoice must be 'any' or 'all'.");
		} else {
			this.type = type;
		}
		this.needsSpace = true;
		this.innerWidth = max(this.items, function(x){return x.width});
		this.width = 30 + Diagram.ARC_RADIUS + this.innerWidth + Diagram.ARC_RADIUS + 20;
		this.up = this.items[0].up;
		this.down = this.items[this.items.length-1].down;
		this.height = this.items[normal].height;
		for(var i = 0; i < this.items.length; i++) {
			var item = this.items[i];
			if(i == normal - 1 || i == normal + 1) var minimum = 10 + Diagram.ARC_RADIUS;
			else var minimum = Diagram.ARC_RADIUS;
			if(i < normal) {
				this.up += Math.max(minimum, item.height + item.down + Diagram.VERTICAL_SEPARATION + this.items[i+1].up);
			} else if(i > normal) {
				this.down += Math.max(minimum, item.up + Diagram.VERTICAL_SEPARATION + this.items[i-1].down + this.items[i-1].height);
			}
		}
		this.down -= this.items[normal].height; // already counted in this.height
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "multiplechoice"
		}
	}
	subclassOf(MultipleChoice, DiagramMultiContainer);
	MultipleChoice.prototype.format = function(x, y, width) {
		var gaps = determineGaps(width, this.width);
		Path(x, y).right(gaps[0]).addTo(this);
		Path(x + gaps[0] + this.width, y + this.height).right(gaps[1]).addTo(this);
		x += gaps[0];

		var normal = this.items[this.normal];

		// Do the elements that curve above
		for(var i = this.normal - 1; i >= 0; i--) {
			var item = this.items[i];
			if( i == this.normal - 1 ) {
				var distanceFromY = Math.max(10 + Diagram.ARC_RADIUS, normal.up + Diagram.VERTICAL_SEPARATION + item.down + item.height);
			}
			Path(x + 30,y)
				.up(distanceFromY - Diagram.ARC_RADIUS)
				.arc('wn').addTo(this);
			item.format(x + 30 + Diagram.ARC_RADIUS, y - distanceFromY, this.innerWidth).addTo(this);
			Path(x + 30 + Diagram.ARC_RADIUS + this.innerWidth, y - distanceFromY + item.height)
				.arc('ne')
				.down(distanceFromY - item.height + this.height - Diagram.ARC_RADIUS - 10)
				.addTo(this);
			if(i != 0) {
				distanceFromY += Math.max(Diagram.ARC_RADIUS, item.up + Diagram.VERTICAL_SEPARATION + this.items[i-1].down + this.items[i-1].height);
			}
		}

		Path(x + 30, y).right(Diagram.ARC_RADIUS).addTo(this);
		normal.format(x + 30 + Diagram.ARC_RADIUS, y, this.innerWidth).addTo(this);
		Path(x + 30 + Diagram.ARC_RADIUS + this.innerWidth, y + this.height).right(Diagram.ARC_RADIUS).addTo(this);

		for(var i = this.normal+1; i < this.items.length; i++) {
			var item = this.items[i];
			if(i == this.normal + 1) {
				var distanceFromY = Math.max(10+Diagram.ARC_RADIUS, normal.height + normal.down + Diagram.VERTICAL_SEPARATION + item.up);
			}
			Path(x + 30, y)
				.down(distanceFromY - Diagram.ARC_RADIUS)
				.arc('ws')
				.addTo(this);
			item.format(x + 30 + Diagram.ARC_RADIUS, y + distanceFromY, this.innerWidth).addTo(this)
			Path(x + 30 + Diagram.ARC_RADIUS + this.innerWidth, y + distanceFromY + item.height)
				.arc('se')
				.up(distanceFromY - Diagram.ARC_RADIUS + item.height - normal.height)
				.addTo(this);
			if(i != this.items.length - 1) {
				distanceFromY += Math.max(Diagram.ARC_RADIUS, item.height + item.down + Diagram.VERTICAL_SEPARATION + this.items[i+1].up);
			}
		}
		var text = FakeSVG('g', {"class": "diagram-text"}).addTo(this)
		FakeSVG('title', {}, (this.type=="any"?"take one or more branches, once each, in any order":"take all branches, once each, in any order")).addTo(text)
		FakeSVG('path', {
			"d": "M "+(x+30)+" "+(y-10)+" h -26 a 4 4 0 0 0 -4 4 v 12 a 4 4 0 0 0 4 4 h 26 z",
			"class": "diagram-text"
			}).addTo(text)
		FakeSVG('text', {
			"x": x + 15,
			"y": y + 4,
			"class": "diagram-text"
			}, (this.type=="any"?"1+":"all")).addTo(text)
		FakeSVG('path', {
			"d": "M "+(x+this.width-20)+" "+(y-10)+" h 16 a 4 4 0 0 1 4 4 v 12 a 4 4 0 0 1 -4 4 h -16 z",
			"class": "diagram-text"
			}).addTo(text)
		FakeSVG('path', {
			"d": "M "+(x+this.width-13)+" "+(y-2)+" a 4 4 0 1 0 6 -1 m 2.75 -1 h -4 v 4 m 0 -3 h 2",
			"style": "stroke-width: 1.75"
		}).addTo(text)
		return this;
	};

	var Optional = funcs.Optional = function Optional(item, skip) {
		if( skip === undefined )
			return Choice(1, Skip(), item);
		else if ( skip === "skip" )
			return Choice(0, Skip(), item);
		else
			throw "Unknown value for Optional()'s 'skip' argument.";
	}

	var OneOrMore = funcs.OneOrMore = function OneOrMore(item, rep) {
		if(!(this instanceof OneOrMore)) return new OneOrMore(item, rep);
		FakeSVG.call(this, 'g');
		rep = rep || (new Skip);
		this.item = wrapString(item);
		this.rep = wrapString(rep);
		this.width = Math.max(this.item.width, this.rep.width) + Diagram.ARC_RADIUS*2;
		this.height = this.item.height;
		this.up = this.item.up;
		this.down = Math.max(Diagram.ARC_RADIUS*2, this.item.down + Diagram.VERTICAL_SEPARATION + this.rep.up + this.rep.height + this.rep.down);
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "oneormore"
		}
	}
	subclassOf(OneOrMore, FakeSVG);
	OneOrMore.prototype.needsSpace = true;
	OneOrMore.prototype.format = function(x,y,width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y+this.height).h(gaps[1]).addTo(this);
		x += gaps[0];

		// Draw item
		Path(x,y).right(Diagram.ARC_RADIUS).addTo(this);
		this.item.format(x+Diagram.ARC_RADIUS,y,this.width-Diagram.ARC_RADIUS*2).addTo(this);
		Path(x+this.width-Diagram.ARC_RADIUS,y+this.height).right(Diagram.ARC_RADIUS).addTo(this);

		// Draw repeat arc
		var distanceFromY = Math.max(Diagram.ARC_RADIUS*2, this.item.height+this.item.down+Diagram.VERTICAL_SEPARATION+this.rep.up);
		Path(x+Diagram.ARC_RADIUS,y).arc('nw').down(distanceFromY-Diagram.ARC_RADIUS*2).arc('ws').addTo(this);
		this.rep.format(x+Diagram.ARC_RADIUS, y+distanceFromY, this.width - Diagram.ARC_RADIUS*2).addTo(this);
		Path(x+this.width-Diagram.ARC_RADIUS, y+distanceFromY+this.rep.height).arc('se').up(distanceFromY-Diagram.ARC_RADIUS*2+this.rep.height-this.item.height).arc('en').addTo(this);

		return this;
	}
	OneOrMore.prototype.walk = function(cb) {
		cb(this);
		this.item.walk(cb);
		this.rep.walk(cb);
	}

	var ZeroOrMore = funcs.ZeroOrMore = function ZeroOrMore(item, rep, skip) {
		return Optional(OneOrMore(item, rep), skip);
	}

	var Start = funcs.Start = function Start({type="simple", label}={}) {
		if(!(this instanceof Start)) return new Start({type, label});
		FakeSVG.call(this, 'g');
		this.width = 20;
		this.height = 0;
		this.up = 10;
		this.down = 10;
		this.type = type;
		if(label != undefined) {
			this.label = ""+label;
			this.width = Math.max(20, this.label.length * Diagram.CHAR_WIDTH + 10);
		}
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "start"
		}
	}
	subclassOf(Start, FakeSVG);
	Start.prototype.format = function(x,y) {
		let path = new Path(x, y-10);
		if (this.type === "complex") {
			path.down(20)
				.m(0, -10)
				.right(this.width)
				.addTo(this);
		} else {
			path.down(20)
				.m(10, -20)
				.down(20)
				.m(-10, -10)
				.right(this.width)
				.addTo(this);
		}
		if(this.label) {
			new FakeSVG('text', {x:x, y:y-15, style:"text-anchor:start"}, this.label).addTo(this);
		}
		return this;
	}

	var End = funcs.End = function End({type="simple"}={}) {
		if(!(this instanceof End)) return new End({type});
		FakeSVG.call(this, 'path');
		this.width = 20;
		this.height = 0;
		this.up = 10;
		this.down = 10;
		this.type = type;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "end"
		}
	}
	subclassOf(End, FakeSVG);
	End.prototype.format = function(x,y) {
		if (this.type === "complex") {
			this.attrs.d = 'M '+x+' '+y+' h 20 m 0 -10 v 20';
		} else {
			this.attrs.d = 'M '+x+' '+y+' h 20 m -10 -10 v 20 m 10 -20 v 20';
		}
		return this;
	}

	var Terminal = funcs.Terminal = function Terminal(text, {href, title}={}) {
		if(!(this instanceof Terminal)) return new Terminal(text, {href, title});
		FakeSVG.call(this, 'g', {'class': 'terminal'});
		this.text = ""+text;
		this.href = href;
		this.title = title;
		this.width = this.text.length * Diagram.CHAR_WIDTH + 20;
		this.height = 0;
		this.up = 11;
		this.down = 11;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "terminal"
		}
	}
	subclassOf(Terminal, FakeSVG);
	Terminal.prototype.needsSpace = true;
	Terminal.prototype.format = function(x, y, width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y).h(gaps[1]).addTo(this);
		x += gaps[0];

		FakeSVG('rect', {x:x, y:y-11, width:this.width, height:this.up+this.down, rx:10, ry:10}).addTo(this);
		var text = FakeSVG('text', {x:x+this.width/2, y:y+4}, this.text);
		if(this.href)
			FakeSVG('a', {'xlink:href': this.href}, [text]).addTo(this);
		else
			text.addTo(this);
		if(this.title)
			new FakeSVG('title', {}, this.title).addTo(this);
		return this;
	}

	var NonTerminal = funcs.NonTerminal = function NonTerminal(text, {href, title}={}) {
		if(!(this instanceof NonTerminal)) return new NonTerminal(text, {href, title});
		FakeSVG.call(this, 'g', {'class': 'non-terminal'});
		this.text = ""+text;
		this.href = href;
		this.title = title;
		this.width = this.text.length * Diagram.CHAR_WIDTH + 20;
		this.height = 0;
		this.up = 11;
		this.down = 11;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "nonterminal"
		}
	}
	subclassOf(NonTerminal, FakeSVG);
	NonTerminal.prototype.needsSpace = true;
	NonTerminal.prototype.format = function(x, y, width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y).h(gaps[1]).addTo(this);
		x += gaps[0];

		FakeSVG('rect', {x:x, y:y-11, width:this.width, height:this.up+this.down}).addTo(this);
		var text = FakeSVG('text', {x:x+this.width/2, y:y+4}, this.text);
		if(this.href)
			FakeSVG('a', {'xlink:href': this.href}, [text]).addTo(this);
		else
			text.addTo(this);
		if(this.title)
			new FakeSVG('title', {}, this.title).addTo(this);
		return this;
	}

	var Comment = funcs.Comment = function Comment(text, {href, title}={}) {
		if(!(this instanceof Comment)) return new Comment(text, {href, title});
		FakeSVG.call(this, 'g');
		this.text = ""+text;
		this.href = href;
		this.title = title;
		this.width = this.text.length * Diagram.COMMENT_CHAR_WIDTH + 10;
		this.height = 0;
		this.up = 11;
		this.down = 11;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "comment"
		}
	}
	subclassOf(Comment, FakeSVG);
	Comment.prototype.needsSpace = true;
	Comment.prototype.format = function(x, y, width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		Path(x,y).h(gaps[0]).addTo(this);
		Path(x+gaps[0]+this.width,y+this.height).h(gaps[1]).addTo(this);
		x += gaps[0];

		var text = FakeSVG('text', {x:x+this.width/2, y:y+5, class:'comment'}, this.text);
		if(this.href)
			FakeSVG('a', {'xlink:href': this.href}, [text]).addTo(this);
		else
			text.addTo(this);
		if(this.title)
			new FakeSVG('title', {}, this.title).addTo(this);
		return this;
	}

	var Skip = funcs.Skip = function Skip() {
		if(!(this instanceof Skip)) return new Skip();
		FakeSVG.call(this, 'g');
		this.width = 0;
		this.height = 0;
		this.up = 0;
		this.down = 0;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down
			this.attrs['data-type'] = "skip"
		}
	}
	subclassOf(Skip, FakeSVG);
	Skip.prototype.format = function(x, y, width) {
		Path(x,y).right(width).addTo(this);
		return this;
	}


	var Block = funcs.Block = function Block({width=50, up=15, height=25, down=15, needsSpace=true}={}) {
		if(!(this instanceof Block)) return new Block({width, up, height, down, needsSpace});
		FakeSVG.call(this, 'g');
		this.width = width;
		this.height = height;
		this.up = up;
		this.down = down;
		this.needsSpace = true;
		if(Diagram.DEBUG) {
			this.attrs['data-updown'] = this.up + " " + this.height + " " + this.down;
			this.attrs['data-type'] = "block"
		}
	}
	subclassOf(Block, FakeSVG);
	Block.prototype.format = function(x, y, width) {
		// Hook up the two sides if this is narrower than its stated width.
		var gaps = determineGaps(width, this.width);
		new Path(x,y).h(gaps[0]).addTo(this);
		new Path(x+gaps[0]+this.width,y).h(gaps[1]).addTo(this);
		x += gaps[0];

		new FakeSVG('rect', {x:x, y:y-this.up, width:this.width, height:this.up+this.height+this.down}).addTo(this);
		return this;
	}

	var root;
	if (typeof define === 'function' && define.amd) {
		// AMD. Register as an anonymous module.
		root = {};
		define([], function() {
			return root;
		});
	} else if (typeof exports === 'object') {
		// CommonJS for node
		root = exports;
	} else {
		// Browser globals (root is window)
		root = this;
	}

	for(var name in funcs) {
		root[name] = funcs[name];
	}
}).call(this,
	{
	VERTICAL_SEPARATION: 8,
	ARC_RADIUS: 10,
	DIAGRAM_CLASS: 'railroad-diagram',
	STROKE_ODD_PIXEL_LENGTH: true,
	INTERNAL_ALIGNMENT: 'center',
	CHAR_WIDTH: 8.5, // width of each monospace character. play until you find the right value for your font
	COMMENT_CHAR_WIDTH: 7, // comments are in smaller text by default
	}
);
