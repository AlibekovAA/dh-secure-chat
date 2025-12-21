import { checkMediaRecorderSupport } from '../browser-support';

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

  constructor(private options: AudioRecorderOptions = {}) {
    this.maxDuration = options.maxDuration || 5 * 60 * 1000;
  }

  async start(): Promise<void> {
    checkMediaRecorderSupport();

    if (this.state === 'recording') {
      throw new Error('Запись уже начата');
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
      this.state = 'recording';
      this.startTime = Date.now();
      this.duration = 0;

      this.mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          this.chunks.push(event.data);
        }
      };

      this.mediaRecorder.onerror = (event) => {
        this.state = 'error';
        this.stop();
      };

      this.mediaRecorder.onstop = () => {
        this.duration = Date.now() - this.startTime;
        this.stopStream();
      };

      this.mediaRecorder.start(1000);

      this.startDurationTimer();
    } catch (error) {
      this.state = 'error';
      this.stopStream();
      throw error;
    }
  }

  stop(): Blob | null {
    if (this.state !== 'recording' || !this.mediaRecorder) {
      return null;
    }

    this.stopDurationTimer();

    if (this.startTime > 0) {
      this.duration = Date.now() - this.startTime;
    }

    if (this.mediaRecorder.state === 'recording') {
      this.mediaRecorder.requestData();
      this.mediaRecorder.stop();
    }

    this.state = 'stopped';
    this.stopStream();

    if (this.chunks.length === 0) {
      return null;
    }

    const mimeType = this.getSupportedMimeType();
    return new Blob(this.chunks, { type: mimeType });
  }

  cancel(): void {
    if (this.state === 'recording') {
      this.stop();
    }
    this.chunks = [];
    this.state = 'idle';
    this.duration = 0;
  }

  getDuration(): number {
    if (this.state === 'recording' && this.startTime > 0) {
      return Math.floor((Date.now() - this.startTime) / 1000);
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
    this.durationTimer = window.setInterval(() => {
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
    this.cancel();
    this.stopStream();
  }
}
