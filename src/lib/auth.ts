/** Login state: username stored in a 1-day cookie. */

export function getUsername(): string | null {
  const match = document.cookie
    .split(";")
    .map((c) => c.trim())
    .find((c) => c.startsWith("username="));
  if (!match) return null;
  const value = decodeURIComponent(match.slice("username=".length));
  return value || null;
}

export function setUsername(username: string): void {
  const expires = new Date();
  expires.setDate(expires.getDate() + 1);
  document.cookie = `username=${encodeURIComponent(username)}; expires=${expires.toUTCString()}; path=/`;
}

export function clearUsername(): void {
  document.cookie = "username=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
}
