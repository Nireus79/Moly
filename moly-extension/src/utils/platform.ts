/** The phone app runs these same screens inside a native shell; its shim sets window.molyPhone. */
export interface PhoneBridge {
  /** Reads the clipboard; called only when the person taps a Paste button. */
  readClipboard(): Promise<string>;
  /** Text another app shared to Moly (the share sheet); returned once, then forgotten. Empty when there is none. */
  takeSharedText(): Promise<string>;
  /** A plain-text report of how the app is doing (phone model, when the engine started and stopped, error lines); nothing is sent by it. */
  buildReport?(): Promise<string>;
  /** Opens the phone's share sheet with this text, so the person picks where it goes (e-mail, a messenger, a note). */
  shareReport?(text: string): Promise<void>;
}

export function phoneBridge(): PhoneBridge | null {
  const b = (window as unknown as { molyPhone?: PhoneBridge }).molyPhone;
  return b && typeof b.readClipboard === 'function' ? b : null;
}

export const isPhoneApp = (): boolean => phoneBridge() !== null;
