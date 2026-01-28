import { checkMediaRecorderSupport } from '@/shared/browser-support';
import { MESSAGES } from '@/shared/messages';

export type RecordingState = 'idle' | 'recording' | 'stopped' | 'error';

export type AudioRecorderOptions = {
  mimeType?: string;
  audioBitsPerSecond?: number;
  maxDuration?: number;
};

export class AudioRecorder {
  private mediaRecorder: MediaRecorder | null = null;
  private stream: MediaStream | null = null;
  private chunks: Blob[] = [];
  private state: RecordingState = 'idle';
  private startTime: number = 0;
  private duration: number = 0;
  private maxDuration: number;
  private durationTimer: number | null = null;
  private stopResolve: ((blob: Blob | null) => void) | null = null;
  private startResolve: (() => void) | null = null;

  constructor(private options: AudioRecorderOptions = {}) {
    this.maxDuration = options.maxDuration || 5 * 60 * 1000;
  }

  async start(): Promise<void> {
    checkMediaRecorderSupport();

    if (this.state === 'recording') {
      throw new Error(MESSAGES.common.audioRecorder.errors.alreadyStarted);
    }

    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mimeType = this.getSupportedMimeType();
      const audioBitsPerSecond = this.options.audioBitsPerSecond || 32000;

      this.mediaRecorder = new MediaRecorder(this.stream, {
        mimeType,
        audioBitsPerSecond,
      });

      this.chunks = [];
      this.duration = 0;

      const startPromise = new Promise<void>((resolve, reject) => {
        this.startResolve = resolve;

        const timeout = setTimeout(() => {
          if (this.startResolve === resolve) {
            this.startResolve = null;
            reject(
              new Error(MESSAGES.common.audioRecorder.errors.startTimeout)
            );
          }
        }, 5000);

        const markAsStarted = () => {
          if (this.startTime > 0) {
            return;
          }
          clearTimeout(timeout);
          if (this.startResolve === resolve) {
            this.startResolve = null;
            this.state = 'recording';
            this.startTime = Date.now();
            this.startDurationTimer();
            resolve();
          }
        };

        this.mediaRecorder!.onstart = () => {
          markAsStarted();
        };

        this.mediaRecorder!.onerror = (_event) => {
          clearTimeout(timeout);
          if (this.startResolve === resolve) {
            this.startResolve = null;
            this.state = 'error';
            this.stopStream();
            reject(
              new Error(
                MESSAGES.common.audioRecorder.errors.mediaRecorderStartError
              )
            );
          }
        };
      });

      this.mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          this.chunks.push(event.data);
        }
      };

      this.mediaRecorder.onstop = () => {
        const savedStartTime = this.startTime;
        this.duration = savedStartTime > 0 ? Date.now() - savedStartTime : 0;
        this.stopStream();
        const mime = this.getSupportedMimeType();
        const blob =
          this.chunks.length > 0 ? new Blob(this.chunks, { type: mime }) : null;
        this.state = 'stopped';
        const resolve = this.stopResolve;
        this.stopResolve = null;
        if (resolve) {
          resolve(blob);
        }
      };

      try {
        this.mediaRecorder.start(100);

        await Promise.race([
          startPromise,
          new Promise<void>((resolve) => {
            setTimeout(() => {
              if (
                this.mediaRecorder &&
                this.mediaRecorder.state === 'recording' &&
                this.startTime === 0
              ) {
                if (this.startResolve) {
                  const resolveFn = this.startResolve;
                  this.startResolve = null;
                  this.state = 'recording';
                  this.startTime = Date.now();
                  this.startDurationTimer();
                  resolveFn();
                }
                resolve();
              } else {
                resolve();
              }
            }, 100);
          }),
        ]);
      } catch (startError) {
        this.state = 'error';
        this.stopStream();
        throw startError instanceof Error
          ? startError
          : new Error(
              MESSAGES.common.audioRecorder.errors.failedToStart(
                String(startError)
              )
            );
      }
    } catch (error) {
      this.state = 'error';
      this.stopStream();
      throw error;
    }
  }

  stop(): Promise<Blob | null> {
    if (this.state !== 'recording' || !this.mediaRecorder) {
      return Promise.resolve(null);
    }

    this.stopDurationTimer();

    if (this.startTime > 0) {
      this.duration = Date.now() - this.startTime;
    }

    const states: Array<'recording' | 'paused'> = ['recording', 'paused'];
    if (states.includes(this.mediaRecorder.state as 'recording' | 'paused')) {
      const promise = new Promise<Blob | null>((resolve) => {
        this.stopResolve = resolve;
      });

      try {
        if (this.mediaRecorder.state === 'recording') {
          this.mediaRecorder.requestData();
        }
        this.mediaRecorder.stop();
      } catch (error) {
        this.state = 'error';
        this.stopStream();
        const resolve = this.stopResolve;
        this.stopResolve = null;
        if (resolve) {
          resolve(null);
        }
        return Promise.resolve(null);
      }

      return promise;
    }

    this.state = 'error';
    return Promise.resolve(null);
  }

  cancel(): void {
    if (this.startResolve) {
      this.startResolve = null;
    }
    if (this.state === 'recording' && this.mediaRecorder) {
      this.mediaRecorder.stop();
    }
    this.chunks = [];
    this.state = 'idle';
    this.duration = 0;
    this.startTime = 0;
  }

  getDuration(): number {
    if (this.state === 'recording' && this.startTime > 0) {
      const elapsed = Math.floor((Date.now() - this.startTime) / 1000);
      return Math.max(0, elapsed);
    }
    return Math.floor(this.duration / 1000);
  }

  private getSupportedMimeType(): string {
    const preferredTypes = [
      'audio/webm;codecs=opus',
      'audio/webm',
      'audio/ogg;codecs=opus',
      'audio/ogg',
      'audio/mp4',
    ];

    if (
      this.options.mimeType &&
      MediaRecorder.isTypeSupported(this.options.mimeType)
    ) {
      return this.options.mimeType;
    }

    for (const type of preferredTypes) {
      if (MediaRecorder.isTypeSupported(type)) {
        return type;
      }
    }

    return 'audio/webm';
  }

  private startDurationTimer(): void {
    if (this.durationTimer !== null) {
      clearInterval(this.durationTimer);
    }
    this.durationTimer = window.setInterval(() => {
      if (this.startTime === 0) {
        return;
      }
      const currentDuration = Date.now() - this.startTime;
      if (currentDuration >= this.maxDuration) {
        this.stop();
      }
    }, 100);
  }

  private stopDurationTimer(): void {
    if (this.durationTimer !== null) {
      clearInterval(this.durationTimer);
      this.durationTimer = null;
    }
  }

  private stopStream(): void {
    if (this.stream) {
      this.stream.getTracks().forEach((track) => track.stop());
      this.stream = null;
    }
  }

  cleanup(): void {
    if (this.state === 'recording') {
      this.stop().catch(() => {});
      return;
    }
    if (this.startResolve) {
      this.startResolve = null;
    }
    this.cancel();
    this.stopStream();
  }
}
