import { VOICE_MIME_TYPES, VIDEO_MIME_TYPES } from './constants';

export function extractDurationFromFilename(filename: string): number {
  const voiceMatch = filename.match(/voice-(\d+)s/);
  const videoMatch = filename.match(/video-(\d+)s/);
  const match = voiceMatch || videoMatch;
  const extracted = match ? parseInt(match[1], 10) : 0;
  return extracted > 0 ? extracted : 0;
}

export function isVoiceFile(mimeType: string): boolean {
  return VOICE_MIME_TYPES.some((type) => mimeType.includes(type));
}

export function isVideoFile(mimeType: string): boolean {
  return VIDEO_MIME_TYPES.some((type) => mimeType.includes(type));
}
