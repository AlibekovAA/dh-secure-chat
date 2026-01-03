const DB_NAME = 'secure-chat-db';
const DB_VERSION = 1;
const STORE_NAME = 'keys';

let dbPromise: Promise<IDBDatabase> | null = null;

function openDB(): Promise<IDBDatabase> {
  if (dbPromise) {
    return dbPromise;
  }

  dbPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onerror = () => {
      reject(new Error('Failed to open IndexedDB'));
    };

    request.onsuccess = () => {
      resolve(request.result);
    };

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        const objectStore = db.createObjectStore(STORE_NAME, {
          keyPath: 'id',
        });
        objectStore.createIndex('type', 'type', { unique: false });
      }
    };
  });

  return dbPromise;
}

export async function saveKey(
  id: string,
  keyData: string,
  type: string,
): Promise<void> {
  try {
    const db = await openDB();
    const transaction = db.transaction([STORE_NAME], 'readwrite');
    const store = transaction.objectStore(STORE_NAME);

    await new Promise<void>((resolve, reject) => {
      const request = store.put({ id, keyData, type });
      request.onsuccess = () => resolve();
      request.onerror = () => reject(new Error('Failed to save key'));
    });
  } catch (err) {
    throw new Error(
      `Failed to save key to IndexedDB: ${
        err instanceof Error ? err.message : String(err)
      }`,
    );
  }
}

export async function loadKey(id: string): Promise<string | null> {
  try {
    const db = await openDB();
    const transaction = db.transaction([STORE_NAME], 'readonly');
    const store = transaction.objectStore(STORE_NAME);

    return new Promise<string | null>((resolve, reject) => {
      const request = store.get(id);
      request.onsuccess = () => {
        const result = request.result;
        resolve(result ? result.keyData : null);
      };
      request.onerror = () => reject(new Error('Failed to load key'));
    });
  } catch (err) {
    return null;
  }
}

export async function clearAllKeys(): Promise<void> {
  try {
    const db = await openDB();
    const transaction = db.transaction([STORE_NAME], 'readwrite');
    const store = transaction.objectStore(STORE_NAME);

    await new Promise<void>((resolve, reject) => {
      const request = store.clear();
      request.onsuccess = () => resolve();
      request.onerror = () => reject(new Error('Failed to clear keys'));
    });
  } catch (err) {
    throw new Error(
      `Failed to clear keys from IndexedDB: ${
        err instanceof Error ? err.message : String(err)
      }`,
    );
  }
}
