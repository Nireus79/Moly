/**
 * Whether the person has already sent a message in this open panel. The profile form ("Tell me about yourself") is only
 * offered before that: it must never open over a conversation that has begun, for example when the app's slow answer to
 * the profile request arrives minutes later.
 */
const KEY = 'moly_chat_started';

export function markChatStarted(): void {
  try {
    sessionStorage.setItem(KEY, '1');
  } catch {
    /* storage unavailable */
  }
}

export function chatStarted(): boolean {
  try {
    return sessionStorage.getItem(KEY) === '1';
  } catch {
    return false;
  }
}
