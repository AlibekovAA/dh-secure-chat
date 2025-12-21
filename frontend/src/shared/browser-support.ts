function checkBrowserSupport(): {
  webCrypto: boolean;
  mediaRecorder: boolean;
  getUserMedia: boolean;
} {
  const support = {
    webCrypto: false,
    mediaRecorder: false,
    getUserMedia: false,
  };

  if (typeof window !== 'undefined') {
    support.webCrypto = !!(window.crypto && window.crypto.subtle);
    support.mediaRecorder = typeof MediaRecorder !== 'undefined';
    const nav = navigator as Navigator & {
      getUserMedia?: unknown;
      webkitGetUserMedia?: unknown;
      mozGetUserMedia?: unknown;
    };
    support.getUserMedia =
      !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia) ||
      !!(nav.getUserMedia || nav.webkitGetUserMedia || nav.mozGetUserMedia);
  }

  return support;
}

function getBrowserSupportMessage(unsupported: string[]): string | null {
  if (unsupported.length === 0) {
    return null;
  }

  const features = unsupported.join(', ');
  return `Ваш браузер не поддерживает: ${features}. Используйте современный браузер (Chrome, Firefox, Safari, Edge).`;
}

export function checkWebCryptoSupport(): void {
  const support = checkBrowserSupport();
  if (!support.webCrypto) {
    throw new Error(
      getBrowserSupportMessage(['Web Crypto API']) ||
        'Web Crypto API не поддерживается',
    );
  }
}

export function checkMediaRecorderSupport(): void {
  const support = checkBrowserSupport();
  if (!support.mediaRecorder || !support.getUserMedia) {
    throw new Error(
      getBrowserSupportMessage(['MediaRecorder API', 'getUserMedia API']) ||
        'Запись аудио не поддерживается',
    );
  }
}
