import { describe, expect, it } from 'vitest';

import deepmerge from './index.ts';

type Merged = Record<string, unknown>;

// The options vite-plugin-svelte passes: an array replaces the one it merges into.
const replace = { arrayMerge: (target: unknown[], source: unknown[]) => source ?? target };
const concat = { arrayMerge: (target: unknown[], source: unknown[]) => [...target, ...source] };

const merge = (target: unknown, source: unknown, options = replace) => deepmerge(target, source, options) as Merged;

class Preprocessor {
  readonly name = 'class';
}

describe('deepmerge', () => {
  it('merges plain objects key by key, all the way down', () => {
    const target = { a: 1, nested: { b: 1, c: 1, deeper: { d: 1 } } };
    const source = { nested: { c: 2, deeper: { e: 2 } }, f: 2 };

    expect(merge(target, source)).toEqual({ a: 1, nested: { b: 1, c: 2, deeper: { d: 1, e: 2 } }, f: 2 });
  });

  it('merges objects with no prototype as plain ones, into objects that have one', () => {
    const source = Object.assign(Object.create(null), { a: 1 });

    const merged = merge({}, source);

    expect(merged).toEqual({ a: 1 });
    expect(Object.getPrototypeOf(merged)).toBe(Object.prototype);
  });

  it("hands arrays to the options' arrayMerge, with an empty target where there was no array", () => {
    expect(merge({ list: [1] }, { list: [2] }, concat)).toEqual({ list: [1, 2] });
    expect(merge({ list: { a: 1 } }, { list: [2] }, concat)).toEqual({ list: [2] });
    expect(merge({ list: [1] }, {}, concat)).toEqual({ list: [1] });
    expect(merge({ list: [1] }, { list: [2] })).toEqual({ list: [2] });
  });

  it.each([
    ['undefined', undefined],
    ['null', null],
    ['a string', 'text'],
    ['a function', () => 'run'],
    ['a RegExp', /\.svelte$/],
    ['a Date', new Date(0)],
    ['a Map', new Map([['a', 1]])],
    ['a class instance', new Preprocessor()],
  ])('replaces the value with %s, as it is', (_, value) => {
    expect(merge({ value: { kept: true } }, { value }).value).toBe(value);
  });

  it('replaces a value that is not a plain object with a copy of the plain object', () => {
    expect(merge({ value: new Preprocessor() }, { value: { a: 1 } })).toEqual({ value: { a: 1 } });
  });

  it('changes neither input and copies the objects inside them', () => {
    const target = { both: { a: 1 }, mine: { b: 1 } };
    const source = { both: { c: 2 }, theirs: { d: 2 } };

    const merged = merge(target, source);

    expect(target).toEqual({ both: { a: 1 }, mine: { b: 1 } });
    expect(source).toEqual({ both: { c: 2 }, theirs: { d: 2 } });
    expect(merged.mine).toEqual(target.mine);
    expect(merged.mine).not.toBe(target.mine);
    expect(merged.theirs).toEqual(source.theirs);
    expect(merged.theirs).not.toBe(source.theirs);
  });

  describe('with a __proto__ key, as JSON.parse makes one', () => {
    const poison = '{"__proto__":{"admin":true},"a":1}';

    it.each([
      ['the source', () => merge({}, JSON.parse(poison))],
      ['the target', () => merge(JSON.parse(poison), {})],
      ['an object the source nests', () => merge({}, { inner: JSON.parse(poison) }).inner as Merged],
      // deepmerge 4.3.1 sets this result's prototype (GHSA-gcqx-w48m-w2h4).
      [
        'an object the source nests, merged into one with no prototype',
        () => merge({ inner: Object.create(null) }, { inner: JSON.parse(poison) }).inner as Merged,
      ],
    ])('in %s leaves the prototype alone', (_, merged) => {
      const result = merged();

      expect(result).toEqual({ a: 1 });
      expect(Object.getPrototypeOf(result)).toBe(Object.prototype);
      expect(result.admin).toBeUndefined();
      expect(Object.prototype).not.toHaveProperty('admin');
    });
  });
});
