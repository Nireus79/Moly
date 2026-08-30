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
  messageContext: 'dating' | 'professional' | 'social';
}

const PLATFORM_CONFIGS: Record<string, PlatformConfig> = {
  'tinder.com': {
    platform: 'tinder',
    messageContext: 'dating',
  },
  'bumble.com': {
    platform: 'bumble',
    messageContext: 'dating',
  },
  'hinge.com': {
    platform: 'hinge',
    messageContext: 'dating',
  },
  'match.com': {
    platform: 'match',
    messageContext: 'dating',
  },
  'okcupid.com': {
    platform: 'okcupid',
    messageContext: 'dating',
  },
  'fetlife.com': {
    platform: 'fetlife',
    messageContext: 'dating',
  },
  'facebook.com': {
    platform: 'facebook',
    messageContext: 'social',
  },
  'linkedin.com': {
    platform: 'linkedin',
    messageContext: 'professional',
  },
  'discord.com': {
    platform: 'discord',
    messageContext: 'social',
  },
  'slack.com': {
    platform: 'slack',
    messageContext: 'professional',
  },
  'twitter.com': {
    platform: 'twitter',
    messageContext: 'social',
  },
  'x.com': {
    platform: 'twitter',
    messageContext: 'social',
  },
  'web.whatsapp.com': {
    platform: 'whatsapp',
    messageContext: 'social',
  },
  'web.telegram.org': {
    platform: 'telegram',
    messageContext: 'social',
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
    messageContext: 'social',
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
