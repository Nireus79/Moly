/** The phone app runs these same screens inside a native shell; its shim sets window.molyPhone. */
export interface PhoneBridge {
  /** The loopback port of this phone's engine (random, chosen once per install). Absent: the desktop default. */
  enginePort?: number;
  /** Reads the clipboard; called only when the person taps a Paste button. */
  readClipboard(): Promise<string>;
  /** How long the app was off the screen before it last came back, in milliseconds; 0 when not known. */
  awayMs?(): Promise<number>;
  /** Text another app shared to Moly (the share sheet); returned once, then forgotten. Empty when there is none. */
  takeSharedText(): Promise<string>;
  /** A plain-text report of how the app is doing (phone model, when the engine started and stopped, error lines); nothing is sent by it. */
  buildReport?(): Promise<string>;
  /** Opens the phone's share sheet with this text, so the person picks where it goes (e-mail, a messenger, a note). */
  shareReport?(text: string): Promise<void>;
  /** Opens the phone's "save as" picker so the person chooses where the file goes. true: saved; false: they cancelled. */
  saveFile?(name: string, text: string): Promise<boolean>;
  /** Opens the phone's file picker and returns the text of the chosen file; null when they cancelled. */
  pickFile?(): Promise<string | null>;
}

export function phoneBridge(): PhoneBridge | null {
  const b = (window as unknown as { molyPhone?: PhoneBridge }).molyPhone;
  return b && typeof b.readClipboard === 'function' ? b : null;
}

export const isPhoneApp = (): boolean => phoneBridge() !== null;

/** The port the Moly engine listens on: the phone's own, or the desktop's fixed one. */
export function enginePort(): number {
  return phoneBridge()?.enginePort ?? 11436;
}
