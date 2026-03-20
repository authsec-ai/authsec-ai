// WebAuthn utilities: base64url conversions and credential serialization

export const bufferToBase64Url = (buffer: ArrayBuffer): string =>
  btoa(String.fromCharCode(...new Uint8Array(buffer)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');

export const base64UrlToBuffer = (base64url: string): ArrayBuffer => {
  const base64 = (base64url || '').replace(/-/g, '+').replace(/_/g, '/');
  const padding = '='.repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(base64 + padding);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
};

export function publicKeyCredentialToJSON(cred: PublicKeyCredential) {
  const id = cred.id;
  const rawId = bufferToBase64Url(cred.rawId);
  const type = cred.type as 'public-key';
  const clientDataJSON = bufferToBase64Url(cred.response.clientDataJSON);

  if ('attestationObject' in cred.response) {
    const attestationObject = bufferToBase64Url((cred.response as AuthenticatorAttestationResponse).attestationObject);
    return {
      id,
      rawId,
      type,
      response: { clientDataJSON, attestationObject },
    };
  }

  const assertion = cred.response as AuthenticatorAssertionResponse;
  return {
    id,
    rawId,
    type,
    response: {
      clientDataJSON,
      authenticatorData: bufferToBase64Url(assertion.authenticatorData),
      signature: bufferToBase64Url(assertion.signature),
      userHandle: assertion.userHandle ? bufferToBase64Url(assertion.userHandle) : null,
    },
  };
}

export function extractPublicKeyOptions(obj: any) {
  if (!obj) return obj;
  let pk = obj.publicKey || obj;
  if (pk?.publicKey) pk = pk.publicKey;
  return pk;
}

