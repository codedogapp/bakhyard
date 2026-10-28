// Takes deepmerge's place for vite-plugin-svelte, which merges its options with it (decision 148), so it does only
// what the plugin asks: plain objects merge key by key into new ones, arrays go through arrayMerge, and any other
// value replaces. A __proto__ key is never written, so no merge sets a result's prototype.
type Options = { arrayMerge: (target: unknown[], source: unknown[]) => unknown[] };

type Plain = Record<string, unknown>;

function isPlain(value: unknown): value is Plain {
  if (typeof value !== 'object' || value === null) return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function mergeObject(target: Plain, source: Plain, options: Options): Plain {
  const merged: Plain = {};
  for (const [key, value] of [...Object.entries(target), ...Object.entries(source)]) {
    if (key !== '__proto__') merged[key] = deepmerge(merged[key], value, options);
  }
  return merged;
}

export default function deepmerge(target: unknown, source: unknown, options: Options): unknown {
  if (Array.isArray(source)) return options.arrayMerge(Array.isArray(target) ? target : [], source);
  if (!isPlain(source)) return source;
  return mergeObject(isPlain(target) ? target : {}, source, options);
}
