/**
 * Platform Detection
 * Identifies which website the user is on
 */

export type Platform =
  | 'tinder'
  | 'bumble'
  | 'hinge'
  | 'match'
  | 'okcupid'
  | 'fetlife'
  | 'facebook'
  | 'linkedin'
  | 'discord'
  | 'slack'
  | 'twitter'
  | 'whatsapp'
  | 'telegram'
  | 'unknown';

export interface PlatformConfig {
  platform: Platform;
  messageSelector?: string[];
  senderSelector?: string[];
}

const PLATFORM_CONFIGS: Record<string, PlatformConfig> = {
  'tinder.com': {
    platform: 'tinder',
    messageSelector: [
      '[data-qa="messageItem"]',
      '.Bubble__bubble',
      '[role="article"]',
      '.message-bubble',
    ],
    senderSelector: [
      '[data-qa="messageSender"]',
      '.Bubble__senderName',
      '.message-sender',
    ],
  },
  'bumble.com': {
    platform: 'bumble',
    messageSelector: [
      '[data-testid="message-item"]',
      '.message-container',
      '[role="article"]',
      '.bubble',
    ],
    senderSelector: [
      '[data-testid="message-sender"]',
      '.message-sender-name',
      '.sender-name',
    ],
  },
  'hinge.com': {
    platform: 'hinge',
    messageSelector: [
      '[data-testid="chatMessage"]',
      '.chat-message',
      '[role="article"]',
      '.message-item',
    ],
    senderSelector: [
      '[data-testid="senderName"]',
      '.chat-sender',
      '.message-sender',
    ],
  },
  'match.com': {
    platform: 'match',
    messageSelector: [
      '[data-qa="message"]',
      '.chat-bubble',
      '[role="article"]',
      '.message-box',
    ],
    senderSelector: [
      '[data-qa="sender"]',
      '.bubble-name',
      '.message-from',
    ],
  },
  'okcupid.com': {
    platform: 'okcupid',
    messageSelector: [
      '[data-testid="message-bubble"]',
      '.message',
      '[role="article"]',
      '.chat-item',
    ],
    senderSelector: [
      '[data-testid="message-sender"]',
      '.username',
      '.sender',
    ],
  },
  'fetlife.com': {
    platform: 'fetlife',
    messageSelector: [
      '.message-item',
      '.pm-item',
      '[role="article"]',
      '.bubble',
    ],
    senderSelector: [
      '.message-author',
      '.pm-from',
      '.sender-name',
    ],
  },
  'facebook.com': {
    platform: 'facebook',
    messageSelector: [
      '[data-qa="message_container"]',
      '.msg',
      '[role="article"]',
      '[data-testid="message"]',
    ],
    senderSelector: [
      '[data-qa="message_sender"]',
      '.message__author',
      '.msg-meta',
    ],
  },
  'linkedin.com': {
    platform: 'linkedin',
    messageSelector: [
      '[data-qa="message-item"]',
      '.msg-s-message-list__item',
      '[role="article"]',
      '.message-item',
    ],
    senderSelector: [
      '[data-qa="message-sender"]',
      '.msg-s-message-list__item-text__sender',
      '.sender-name',
    ],
  },
  'discord.com': {
    platform: 'discord',
    messageSelector: [
      '[data-testid="message"]',
      '[id^="chat-messages-"] [role="article"]',
      '.messageContent-2qWWxC',
      '[role="article"]',
    ],
    senderSelector: [
      '[data-testid="message-author-username"]',
      '.username-2d0OO5',
      '.author',
    ],
  },
  'slack.com': {
    platform: 'slack',
    messageSelector: [
      '[data-qa-type="message"]',
      '[role="article"]',
      '.c-virtual_list__item',
      '.message-in',
    ],
    senderSelector: [
      '[data-qa="message_sender"]',
      '.c-message__sender_profile_name',
      '.message_sender',
    ],
  },
  'twitter.com': {
    platform: 'twitter',
    messageSelector: [
      '[data-testid="tweet"]',
      '[role="article"]',
      '.Tweet',
      '[data-qa="tweet"]',
    ],
    senderSelector: [
      '[data-testid="Tweet-User-Name"]',
      '.tweet-author',
      '.fullname',
    ],
  },
  'x.com': {
    platform: 'twitter',
    messageSelector: [
      '[data-testid="tweet"]',
      '[role="article"]',
      '.Tweet',
      '[data-qa="tweet"]',
    ],
    senderSelector: [
      '[data-testid="Tweet-User-Name"]',
      '.tweet-author',
      '.fullname',
    ],
  },
  'web.whatsapp.com': {
    platform: 'whatsapp',
    messageSelector: [
      '[data-testid="message"]',
      '[role="article"]',
      '.message-in',
      '.bubble',
    ],
    senderSelector: [
      '[data-testid="message-sender"]',
      '.message-meta',
      '.sender',
    ],
  },
  'web.telegram.org': {
    platform: 'telegram',
    messageSelector: [
      '[data-mid]',
      '.message',
      '[role="article"]',
      '.bubble',
    ],
    senderSelector: [
      '.from_name',
      '.message-author',
      '.sender-name',
    ],
  },
};

export function detectPlatform(): PlatformConfig {
  const url = window.location.href;

  // Check exact domain matches first
  for (const [domain, config] of Object.entries(PLATFORM_CONFIGS)) {
    if (url.includes(domain)) {
      return config;
    }
  }

  // Return generic config for unknown platforms
  return {
    platform: 'unknown',
  };
}

export function getPlatformName(platform: Platform): string {
  const names: Record<Platform, string> = {
    tinder: 'Tinder',
    bumble: 'Bumble',
    hinge: 'Hinge',
    match: 'Match',
    okcupid: 'OkCupid',
    fetlife: 'FetLife',
    facebook: 'Facebook',
    linkedin: 'LinkedIn',
    discord: 'Discord',
    slack: 'Slack',
    twitter: 'Twitter',
    whatsapp: 'WhatsApp',
    telegram: 'Telegram',
    unknown: 'Unknown',
  };
  return names[platform];
}
