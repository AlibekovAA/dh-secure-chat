import {
  VIDEO_THUMBNAIL_SIZE,
  VIDEO_THUMBNAIL_CACHE_PREFIX,
  VIDEO_THUMBNAIL_CACHE_DURATION_MS,
} from './constants';

interface CachedThumbnail {
  dataUrl: string;
  timestamp: number;
}

function getCacheKey(fileId: string): string {
  return `${VIDEO_THUMBNAIL_CACHE_PREFIX}${fileId}`;
}

export async function getCachedThumbnail(
  fileId: string,
): Promise<string | null> {
  try {
    const cached = localStorage.getItem(getCacheKey(fileId));
    if (!cached) return null;

    const parsed: CachedThumbnail = JSON.parse(cached);
    const now = Date.now();

    if (now - parsed.timestamp > VIDEO_THUMBNAIL_CACHE_DURATION_MS) {
      localStorage.removeItem(getCacheKey(fileId));
      return null;
    }

    return parsed.dataUrl;
  } catch {
    return null;
  }
}

export async function cacheThumbnail(
  fileId: string,
  dataUrl: string,
): Promise<void> {
  try {
    const cached: CachedThumbnail = {
      dataUrl,
      timestamp: Date.now(),
    };
    localStorage.setItem(getCacheKey(fileId), JSON.stringify(cached));
  } catch (err) {
    if (err instanceof Error && err.name !== 'QuotaExceededError') {
      console.warn('Failed to cache thumbnail:', err);
    }
  }
}

export async function generateVideoThumbnail(
  videoBlob: Blob,
  fileId: string,
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
      video.currentTime = 0.1;
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

        ctx.fillStyle = '#0a0a0a';
        ctx.fillRect(0, 0, size, size);

        ctx.beginPath();
        ctx.arc(size / 2, size / 2, size / 2, 0, 2 * Math.PI);
        ctx.clip();

        ctx.drawImage(video, offsetX, offsetY, drawWidth, drawHeight);

        const dataUrl = canvas.toDataURL('image/jpeg', 0.85);
        cacheThumbnail(fileId, dataUrl).catch(() => {});
        cleanup();
        resolve(dataUrl);
      } catch (err) {
        cleanup();
        reject(err);
      }
    });

    video.addEventListener('error', (e) => {
      cleanup();
      reject(new Error('Failed to load video for thumbnail'));
    });

    video.load();
  });
}
