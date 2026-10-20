import {
  VIDEO_THUMBNAIL_SIZE,
  VIDEO_THUMBNAIL_CACHE_PREFIX,
  VIDEO_THUMBNAIL_CACHE_DURATION_MS,
  VIDEO_THUMBNAIL_SEEK_TIME,
  JPEG_QUALITY,
  VIDEO_THUMBNAIL_BG_COLOR,
} from '@/shared/constants';

interface CachedThumbnail {
  dataUrl: string;
  timestamp: number;
  timeoutId: number;
}

const thumbnailCache = new Map<string, CachedThumbnail>();

function clearPersistedThumbnails(): void {
  try {
    for (let index = localStorage.length - 1; index >= 0; index--) {
      const key = localStorage.key(index);
      if (key?.startsWith(VIDEO_THUMBNAIL_CACHE_PREFIX)) {
        localStorage.removeItem(key);
      }
    }
  } catch {
    void 0;
  }
}

clearPersistedThumbnails();

export async function getCachedThumbnail(
  fileId: string
): Promise<string | null> {
  const cached = thumbnailCache.get(fileId);
  if (!cached) return null;
  if (Date.now() - cached.timestamp > VIDEO_THUMBNAIL_CACHE_DURATION_MS) {
    window.clearTimeout(cached.timeoutId);
    thumbnailCache.delete(fileId);
    return null;
  }
  return cached.dataUrl;
}

export async function cacheThumbnail(
  fileId: string,
  dataUrl: string
): Promise<void> {
  const existing = thumbnailCache.get(fileId);
  if (existing) {
    window.clearTimeout(existing.timeoutId);
  }

  const timeoutId = window.setTimeout(() => {
    thumbnailCache.delete(fileId);
  }, VIDEO_THUMBNAIL_CACHE_DURATION_MS);
  thumbnailCache.set(fileId, { dataUrl, timestamp: Date.now(), timeoutId });
}

export function clearVideoThumbnailCache(): void {
  for (const cached of thumbnailCache.values()) {
    window.clearTimeout(cached.timeoutId);
  }
  thumbnailCache.clear();
  clearPersistedThumbnails();
}

export async function generateVideoThumbnail(
  videoBlob: Blob,
  fileId: string
): Promise<string> {
  const cached = await getCachedThumbnail(fileId);
  if (cached) {
    return cached;
  }

  return new Promise((resolve, reject) => {
    const video = document.createElement('video');
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d', { willReadFrequently: true });

    if (!ctx) {
      reject(new Error('Failed to get canvas context'));
      return;
    }

    const url = URL.createObjectURL(videoBlob);
    video.src = url;
    video.muted = true;
    video.playsInline = true;
    video.preload = 'metadata';

    const cleanup = () => {
      URL.revokeObjectURL(url);
      video.remove();
      canvas.remove();
    };

    video.addEventListener('loadedmetadata', () => {
      video.currentTime = VIDEO_THUMBNAIL_SEEK_TIME;
    });

    video.addEventListener('seeked', () => {
      try {
        const size = VIDEO_THUMBNAIL_SIZE;
        canvas.width = size;
        canvas.height = size;

        const videoAspect = video.videoWidth / video.videoHeight;
        let drawWidth = size;
        let drawHeight = size;
        let offsetX = 0;
        let offsetY = 0;

        if (videoAspect > 1) {
          drawHeight = size / videoAspect;
          offsetY = (size - drawHeight) / 2;
        } else {
          drawWidth = size * videoAspect;
          offsetX = (size - drawWidth) / 2;
        }

        ctx.fillStyle = VIDEO_THUMBNAIL_BG_COLOR;
        ctx.fillRect(0, 0, size, size);

        ctx.beginPath();
        ctx.arc(size / 2, size / 2, size / 2, 0, 2 * Math.PI);
        ctx.clip();

        ctx.drawImage(video, offsetX, offsetY, drawWidth, drawHeight);

        const dataUrl = canvas.toDataURL('image/jpeg', JPEG_QUALITY);
        cacheThumbnail(fileId, dataUrl).catch(() => {});
        cleanup();
        resolve(dataUrl);
      } catch (err) {
        cleanup();
        reject(err);
      }
    });

    video.addEventListener('error', (_e) => {
      cleanup();
      reject(new Error('Failed to load video for thumbnail'));
    });

    video.load();
  });
}
