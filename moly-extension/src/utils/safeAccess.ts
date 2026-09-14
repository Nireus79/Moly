/**
 * Safe access utilities to prevent crashes from null/undefined/empty arrays
 */

/**
 * Safely get first element of array
 */
export function getFirst<T>(arr: T[] | null | undefined): T | null {
  if (!arr || !Array.isArray(arr) || arr.length === 0) {
    return null;
  }
  return arr[0];
}

/**
 * Safely get element at index
 */
export function getAt<T>(arr: T[] | null | undefined, index: number): T | null {
  if (!arr || !Array.isArray(arr) || index < 0 || index >= arr.length) {
    return null;
  }
  return arr[index];
}

/**
 * Check if array has items
 */
export function hasItems<T>(arr: T[] | null | undefined): boolean {
  return Array.isArray(arr) && arr.length > 0;
}

/**
 * Get array length safely
 */
export function getLength<T>(arr: T[] | null | undefined): number {
  return Array.isArray(arr) ? arr.length : 0;
}

/**
 * Safely access nested object property
 */
export function getProperty<T>(obj: any, ...keys: string[]): T | null {
  if (!obj || typeof obj !== 'object') {
    return null;
  }

  let current: any = obj;
  for (const key of keys) {
    if (current == null) {
      return null;
    }
    current = current[key];
  }

  return current ?? null;
}

/**
 * Safely validate array has minimum length
 */
export function validateArrayLength<T>(arr: T[] | null | undefined, minLength: number): arr is T[] {
  return Array.isArray(arr) && arr.length >= minLength;
}

/**
 * Safely filter out null/undefined values
 */
export function filterNulls<T>(arr: (T | null | undefined)[]): T[] {
  return arr.filter((item): item is T => item != null);
}

/**
 * Map array safely (handles null input)
 */
export function mapSafe<T, U>(arr: T[] | null | undefined, fn: (item: T) => U): U[] {
  if (!Array.isArray(arr)) {
    return [];
  }
  return arr.map(fn);
}
