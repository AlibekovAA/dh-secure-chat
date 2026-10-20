export function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export function extractDurationFromFilename(filename: string): number {
  const voiceMatch = filename.match(/voice-(\d+)s/);
  const videoMatch = filename.match(/video-(\d+)s/);
  const match = voiceMatch || videoMatch;
  const extracted = match ? parseInt(match[1], 10) : 0;
  return extracted > 0 ? extracted : 0;
}
