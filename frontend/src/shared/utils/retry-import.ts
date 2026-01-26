export function retryImport<T>(
  importFn: () => Promise<T>,
  retries = 2,
  delay = 1000
): Promise<T> {
  return importFn().catch((error) => {
    if (retries > 0) {
      return new Promise((resolve) => {
        setTimeout(() => {
          resolve(retryImport(importFn, retries - 1, delay));
        }, delay);
      });
    }
    throw error;
  });
}
