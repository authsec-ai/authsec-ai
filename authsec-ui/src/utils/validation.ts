/**
 * Detects if a string is an IPv4 address
 * @param host - The host string to check
 * @returns true if host is an IPv4 address, false otherwise
 */
export const isIPv4Address = (host: string): boolean => {
  if (!host) return false;
  const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
  const trimmedHost = host.trim();

  if (!ipv4Regex.test(trimmedHost)) return false;

  // Validate each octet is 0-255
  const octets = trimmedHost.split('.');
  return octets.every(octet => {
    const num = parseInt(octet, 10);
    return num >= 0 && num <= 255;
  });
};

/**
 * Validates port number
 * @param port - Port number as string
 * @returns true if port is valid (1-65535), false otherwise
 */
export const isValidPort = (port: string): boolean => {
  if (!port) return false;
  const portNum = parseInt(port, 10);
  return !isNaN(portNum) && portNum >= 1 && portNum <= 65535;
};

/**
 * Builds host string for API payload
 * @param host - Domain or IP address
 * @param port - Port number (optional)
 * @returns Formatted host string (IP:port or domain)
 */
export const buildHostString = (host: string, port: string): string => {
  const trimmedHost = host.trim();

  if (isIPv4Address(trimmedHost) && port) {
    return `${trimmedHost}:${port.trim()}`;
  }

  return trimmedHost;
};
