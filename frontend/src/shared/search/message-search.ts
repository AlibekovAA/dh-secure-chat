export type SearchableMessage = {
  id: string;
  content: string;
  sender: string;
  timestamp: number;
  [key: string]: unknown;
};

type SearchResult = {
  message: SearchableMessage;
  score: number;
  matches: number;
};

export class MessageSearchEngine {
  private messages: SearchableMessage[] = [];
  private indexed: boolean = false;

  indexMessages(messages: SearchableMessage[]): void {
    this.messages = messages;
    this.indexed = true;
  }

  addMessage(message: SearchableMessage): void {
    this.messages.push(message);
  }

  removeMessage(messageId: string): void {
    this.messages = this.messages.filter((m) => m.id !== messageId);
  }

  clear(): void {
    this.messages = [];
    this.indexed = false;
  }

  search(
    query: string,
    options?: { threshold?: number; limit?: number },
  ): SearchableMessage[] {
    if (!this.indexed || this.messages.length === 0) {
      return [];
    }

    const threshold = options?.threshold ?? 0.3;
    const limit = options?.limit ?? 50;

    if (!query || query.trim().length === 0) {
      return [];
    }

    const queryLower = query.toLowerCase().trim();
    const queryWords = queryLower.split(/\s+/).filter((w) => w.length > 0);

    const results: SearchResult[] = [];

    for (const message of this.messages) {
      const content = message.content.toLowerCase();
      const sender = message.sender.toLowerCase();

      let score = 0;
      let matches = 0;

      if (content.includes(queryLower) || sender.includes(queryLower)) {
        score = 1.0;
        matches = queryLower.length;
      } else {
        let matchedWords = 0;
        for (const word of queryWords) {
          if (content.includes(word) || sender.includes(word)) {
            matchedWords++;
            matches += word.length;
          }
        }

        if (matchedWords > 0) {
          score = matchedWords / queryWords.length;
        }
      }

      if (score >= threshold) {
        results.push({
          message,
          score,
          matches,
        });
      }
    }

    results.sort((a, b) => {
      if (Math.abs(a.score - b.score) > 0.01) {
        return b.score - a.score;
      }
      return b.matches - a.matches;
    });

    return results.slice(0, limit).map((r) => r.message);
  }

  searchBySender(sender: string): SearchableMessage[] {
    const senderLower = sender.toLowerCase();
    return this.messages.filter((m) =>
      m.sender.toLowerCase().includes(senderLower),
    );
  }

  searchByTimeRange(startTime: number, endTime: number): SearchableMessage[] {
    return this.messages.filter(
      (m) => m.timestamp >= startTime && m.timestamp <= endTime,
    );
  }

  getMessageCount(): number {
    return this.messages.length;
  }
}
