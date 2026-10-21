/** The phone app runs these same screens inside a native shell; its shim sets window.molyPhone. */
export interface PhoneBridge {
  /** Reads the clipboard; called only when the person taps a Paste button. */
  readClipboard(): Promise<string>;
  /** Text another app shared to Moly (the share sheet); returned once, then forgotten. Empty when there is none. */
  takeSharedText(): Promise<string>;
}

export function phoneBridge(): PhoneBridge | null {
  const b = (window as unknown as { molyPhone?: PhoneBridge }).molyPhone;
  return b && typeof b.readClipboard === 'function' ? b : null;
}

export const isPhoneApp = (): boolean => phoneBridge() !== null;
