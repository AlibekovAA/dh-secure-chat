export async function encrypt(
  sessionKey: CryptoKey,
  plaintext: string,
): Promise<{ ciphertext: string; nonce: string }> {
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const encoded = new TextEncoder().encode(plaintext);

  const ciphertext = await crypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: nonce,
    },
    sessionKey,
    encoded,
  );

  const ciphertextArray = new Uint8Array(ciphertext);
  const ciphertextBase64 = btoa(
    String.fromCharCode.apply(null, Array.from(ciphertextArray)),
  );
  const nonceBase64 = btoa(String.fromCharCode.apply(null, Array.from(nonce)));

  return {
    ciphertext: ciphertextBase64,
    nonce: nonceBase64,
  };
}

export async function decrypt(
  sessionKey: CryptoKey,
  ciphertext: string,
  nonce: string,
): Promise<string> {
  const ciphertextBinary = Uint8Array.from(atob(ciphertext), (c) =>
    c.charCodeAt(0),
  );
  const nonceBinary = Uint8Array.from(atob(nonce), (c) => c.charCodeAt(0));

  const plaintext = await crypto.subtle.decrypt(
    {
      name: 'AES-GCM',
      iv: nonceBinary,
    },
    sessionKey,
    ciphertextBinary,
  );

  return new TextDecoder().decode(plaintext);
}
