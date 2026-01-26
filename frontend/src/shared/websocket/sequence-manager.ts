import { MAX_SEQUENCES_TO_KEEP } from '@/shared/constants';

type MessageWithSequence = {
  sequence: number;
  message: unknown;
  timestamp: number;
};

type ReorderBufferOptions = {
  maxBufferSize: number;
  maxWaitTime: number;
  onMissingSequence?: (expected: number, received: number) => void;
};

export class SequenceManager {
  private nextSequence = 0;
  private receivedSequences = new Set<number>();
  private buffer = new Map<number, MessageWithSequence>();
  private lastDeliveredSequence = -1;
  private readonly options: Required<ReorderBufferOptions>;

  constructor(options: Partial<ReorderBufferOptions> = {}) {
    this.options = {
      maxBufferSize: options.maxBufferSize ?? 1000,
      maxWaitTime: options.maxWaitTime ?? 5000,
      onMissingSequence: options.onMissingSequence ?? (() => {}),
    };
  }

  getNextSequence(): number {
    const seq = this.nextSequence;
    this.nextSequence = (this.nextSequence + 1) % Number.MAX_SAFE_INTEGER;
    return seq;
  }

  addMessage(
    sequence: number,
    message: unknown
  ): { messages: unknown[]; hasGap: boolean } {
    const now = Date.now();
    const msg: MessageWithSequence = {
      sequence,
      message,
      timestamp: now,
    };

    if (this.receivedSequences.has(sequence)) {
      return { messages: [], hasGap: false };
    }

    this.receivedSequences.add(sequence);

    if (sequence === this.lastDeliveredSequence + 1) {
      this.lastDeliveredSequence = sequence;
      this.cleanupOldSequences();
      return { messages: [message], hasGap: false };
    }

    if (sequence <= this.lastDeliveredSequence) {
      return { messages: [], hasGap: false };
    }

    this.buffer.set(sequence, msg);
    this.cleanupExpired();

    if (this.buffer.size > this.options.maxBufferSize) {
      this.flushBuffer();
    }

    return this.deliverOrdered();
  }

  private deliverOrdered(): { messages: unknown[]; hasGap: boolean } {
    const messages: unknown[] = [];
    let hasGap = false;
    let expected = this.lastDeliveredSequence + 1;

    for (;;) {
      const msg = this.buffer.get(expected);
      if (!msg) {
        if (this.buffer.size > 0) {
          const minBufferSeq = Math.min(...Array.from(this.buffer.keys()));
          if (minBufferSeq > expected) {
            hasGap = true;
            this.options.onMissingSequence(expected, minBufferSeq);
          }
        }
        break;
      }

      messages.push(msg.message);
      this.buffer.delete(expected);
      this.lastDeliveredSequence = expected;
      expected++;
    }

    this.cleanupOldSequences();
    return { messages, hasGap };
  }

  private cleanupExpired(): void {
    const now = Date.now();
    const expired: number[] = [];

    for (const [seq, msg] of this.buffer.entries()) {
      if (now - msg.timestamp > this.options.maxWaitTime) {
        expired.push(seq);
      }
    }

    for (const seq of expired) {
      this.buffer.delete(seq);
      const nextExpected = this.lastDeliveredSequence + 1;
      if (seq > nextExpected) {
        this.options.onMissingSequence(nextExpected, seq);
      }
    }
  }

  private flushBuffer(): void {
    const sorted = Array.from(this.buffer.keys()).sort((a, b) => a - b);
    const nextExpected = this.lastDeliveredSequence + 1;

    for (const seq of sorted) {
      if (seq === nextExpected) {
        this.lastDeliveredSequence = seq;
        this.buffer.delete(seq);
      } else if (seq > nextExpected) {
        break;
      }
    }
  }

  private cleanupOldSequences(): void {
    const maxToKeep = MAX_SEQUENCES_TO_KEEP;
    if (this.receivedSequences.size <= maxToKeep) {
      return;
    }

    const toRemove: number[] = [];
    let count = 0;
    for (const seq of this.receivedSequences) {
      if (seq <= this.lastDeliveredSequence - 1000) {
        toRemove.push(seq);
        count++;
        if (count >= this.receivedSequences.size - maxToKeep) {
          break;
        }
      }
    }

    for (const seq of toRemove) {
      this.receivedSequences.delete(seq);
    }
  }

  reset(): void {
    this.nextSequence = 0;
    this.receivedSequences.clear();
    this.buffer.clear();
    this.lastDeliveredSequence = -1;
  }
}
