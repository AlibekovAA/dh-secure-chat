export function formatTime(
  timestamp: number,
  locale: string = 'ru-RU'
): string {
  return new Date(timestamp).toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
  });
}
