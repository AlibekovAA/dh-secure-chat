export type ChainKey = {
  key: CryptoKey;
  index: number;
};

export type MessageKey = {
  key: CryptoKey;
  index: number;
};

export type DoubleRatchetState = {
  sendingChain: ChainKey | null;
  receivingChain: ChainKey | null;
  dhRatchetKey: CryptoKeyPair | null;
  peerPublicKey: CryptoKey | null;
  rootKey: CryptoKey;
  messageKeys: Map<number, MessageKey>;
  maxSkip: number;
};

export async function generateDHKeyPair(): Promise<CryptoKeyPair> {
  return await crypto.subtle.generateKey(
    {
      name: 'ECDH',
      namedCurve: 'P-256',
    },
    true,
    ['deriveKey', 'deriveBits'],
  );
}

async function deriveChainKey(
  inputKey: CryptoKey,
  salt: BufferSource,
  info: string,
): Promise<CryptoKey> {
  const baseKey = await crypto.subtle.importKey(
    'raw',
    await crypto.subtle.deriveBits(
      {
        name: 'HKDF',
        hash: 'SHA-256',
        salt: salt,
        info: new TextEncoder().encode(info),
      },
      inputKey,
      256,
    ),
    {
      name: 'HKDF',
    },
    false,
    ['deriveKey'],
  );

  return await crypto.subtle.deriveKey(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: new Uint8Array(0),
      info: new TextEncoder().encode('chain-key'),
    },
    baseKey,
    {
      name: 'AES-GCM',
      length: 256,
    },
    false,
    ['encrypt', 'decrypt'],
  );
}

async function deriveMessageKey(chainKey: CryptoKey): Promise<MessageKey> {
  const messageKeyMaterial = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: new Uint8Array(0),
      info: new TextEncoder().encode('message-key'),
    },
    chainKey,
    256,
  );

  const messageKey = await crypto.subtle.importKey(
    'raw',
    messageKeyMaterial,
    {
      name: 'AES-GCM',
      length: 256,
    },
    false,
    ['encrypt', 'decrypt'],
  );

  return {
    key: messageKey,
    index: 0,
  };
}

export async function createDoubleRatchet(
  rootKey: CryptoKey,
  dhKeyPair?: CryptoKeyPair,
): Promise<DoubleRatchetState> {
  const dhKey = dhKeyPair || (await generateDHKeyPair());

  return {
    sendingChain: null,
    receivingChain: null,
    dhRatchetKey: dhKey,
    peerPublicKey: null,
    rootKey: rootKey,
    messageKeys: new Map(),
    maxSkip: 1000,
  };
}

export async function performDHRatchet(
  state: DoubleRatchetState,
  peerPublicKey: CryptoKey,
): Promise<void> {
  if (!state.dhRatchetKey) {
    throw new Error('DH ratchet key not initialized');
  }

  const sharedSecret = await crypto.subtle.deriveBits(
    {
      name: 'ECDH',
      public: peerPublicKey,
    },
    state.dhRatchetKey.privateKey,
    256,
  );

  const sharedSecretKey = await crypto.subtle.importKey(
    'raw',
    sharedSecret,
    {
      name: 'HKDF',
    },
    false,
    ['deriveKey'],
  );

  const salt = new Uint8Array(32);
  state.rootKey = await deriveChainKey(
    sharedSecretKey,
    salt,
    'root-key-update',
  );

  state.receivingChain = {
    key: await deriveChainKey(sharedSecretKey, salt, 'receiving-chain'),
    index: 0,
  };

  state.dhRatchetKey = await generateDHKeyPair();

  const nextSharedSecret = await crypto.subtle.deriveBits(
    {
      name: 'ECDH',
      public: peerPublicKey,
    },
    state.dhRatchetKey.privateKey,
    256,
  );

  const nextSharedSecretKey = await crypto.subtle.importKey(
    'raw',
    nextSharedSecret,
    {
      name: 'HKDF',
    },
    false,
    ['deriveKey'],
  );

  state.sendingChain = {
    key: await deriveChainKey(nextSharedSecretKey, salt, 'sending-chain'),
    index: 0,
  };

  state.peerPublicKey = peerPublicKey;
}

export async function encryptMessage(
  state: DoubleRatchetState,
  plaintext: string,
): Promise<{ ciphertext: string; nonce: string; publicKey: string }> {
  if (!state.sendingChain) {
    throw new Error('Sending chain not initialized');
  }

  const messageKey = await deriveMessageKey(state.sendingChain.key);

  const plaintextBytes = new TextEncoder().encode(plaintext);
  const nonce = crypto.getRandomValues(new Uint8Array(12));

  const ciphertext = await crypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: nonce,
    },
    messageKey.key,
    plaintextBytes,
  );

  state.sendingChain.index++;
  state.sendingChain.key = await deriveChainKey(
    state.sendingChain.key,
    new Uint8Array(0),
    'chain-advance',
  );

  let publicKeyBase64 = '';
  if (state.dhRatchetKey) {
    const exported = await crypto.subtle.exportKey(
      'spki',
      state.dhRatchetKey.publicKey,
    );
    publicKeyBase64 = btoa(String.fromCharCode(...new Uint8Array(exported)));
  }

  return {
    ciphertext: btoa(String.fromCharCode(...new Uint8Array(ciphertext))),
    nonce: btoa(String.fromCharCode(...nonce)),
    publicKey: publicKeyBase64,
  };
}

export async function decryptMessage(
  state: DoubleRatchetState,
  ciphertext: string,
  nonce: string,
  peerPublicKey?: string,
): Promise<string> {
  if (!state.receivingChain) {
    if (peerPublicKey) {
      const key = await importPublicKey(peerPublicKey);
      await performDHRatchet(state, key);
    } else {
      throw new Error('Receiving chain not initialized and no peer public key');
    }
  }

  if (!state.receivingChain) {
    throw new Error('Failed to initialize receiving chain');
  }

  const messageKey = await deriveMessageKey(state.receivingChain.key);

  const ciphertextBytes = Uint8Array.from(atob(ciphertext), (c) =>
    c.charCodeAt(0),
  );
  const nonceBytes = Uint8Array.from(atob(nonce), (c) => c.charCodeAt(0));

  try {
    const plaintext = await crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: nonceBytes,
      },
      messageKey.key,
      ciphertextBytes,
    );

    state.receivingChain.index++;
    state.receivingChain.key = await deriveChainKey(
      state.receivingChain.key,
      new Uint8Array(0),
      'chain-advance',
    );

    return new TextDecoder().decode(plaintext);
  } catch (err) {
    throw new Error(`Decryption failed: ${err}`);
  }
}

async function importPublicKey(base64: string): Promise<CryptoKey> {
  const binary = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
  return await crypto.subtle.importKey(
    'spki',
    binary,
    {
      name: 'ECDH',
      namedCurve: 'P-256',
    },
    true,
    [],
  );
}
