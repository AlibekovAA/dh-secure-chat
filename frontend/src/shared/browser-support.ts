import { MESSAGES } from '@/shared/messages';

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
  return MESSAGES.common.browserSupport.unsupportedFeatures(features);
}

export function checkWebCryptoSupport(): void {
  const support = checkBrowserSupport();
  if (!support.webCrypto) {
    throw new Error(
      getBrowserSupportMessage(['Web Crypto API']) ||
        MESSAGES.common.browserSupport.webCryptoNotSupported
    );
  }
}

export function checkMediaRecorderSupport(): void {
  const support = checkBrowserSupport();
  if (!support.mediaRecorder || !support.getUserMedia) {
    throw new Error(
      getBrowserSupportMessage(['MediaRecorder API', 'getUserMedia API']) ||
        MESSAGES.common.browserSupport.audioRecordingNotSupported
    );
  }
}
