/**
 * Client for the Moly app's database-key gate (/api/keystore/*). Before the encrypted database can open, the app needs its
 * key: from the system keychain, or from what the person chose when there is no keychain. The key itself never reaches
 * the browser; these calls carry only the person's passphrase, TO the app.
 */

import { callBackend, type CallResult } from './llmSettingsAPI';

export type KeystoreState =
  | 'ready'
  | 'locked'
  | 'needs_setup'
  | 'keychain_unavailable'
  | 'unknown_key'
  | 'broken';

export interface KeystoreStatus {
  state: KeystoreState;
  backend?: string;
  reason?: string;
  keychainAvailable: boolean;
  options?: string[];
}

export const getKeystoreStatus = () => callBackend<KeystoreStatus>('GET', '/api/keystore/status');
export const recheckKeystore = () => callBackend<KeystoreStatus>('POST', '/api/keystore/recheck');
export const unlockKeystore = (passphrase: string) => callBackend<KeystoreStatus>('POST', '/api/keystore/unlock', { passphrase });
export const setupKeystore = (mode: 'passphrase' | 'file' | 'keychain', passphrase?: string, confirmInsecure?: boolean) =>
  callBackend<KeystoreStatus>('POST', '/api/keystore/setup', { mode, passphrase, confirmInsecure });

/** True once the app answers /api/status with 200 (the database is open and the handler is ready). */
export async function backendIsReady(): Promise<boolean> {
  const r: CallResult<{ status?: string }> = await callBackend('GET', '/api/status');
  return r.ok;
}
