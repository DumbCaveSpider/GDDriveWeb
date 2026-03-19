export const API = 'http://localhost:3030/api'

export function setCookie(name: string, value: string, days = 365) {
  const expires = new Date(Date.now() + days * 864e5).toUTCString()
  document.cookie = `${name}=${encodeURIComponent(value)}; expires=${expires}; path=/; SameSite=Strict`
}

export function getCookie(name: string): string {
  return document.cookie.split('; ').reduce((acc, c) => {
    const [k, v] = c.split('=')
    return k === name ? decodeURIComponent(v ?? '') : acc
  }, '')
}

export function deleteCookie(name: string) {
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/`
}

export function credHeaders(): Record<string, string> {
  return {
    'X-GD-Username':   getCookie('gdd_username'),
    'X-GD-GJP2':       getCookie('gdd_gjp2'),
    'X-GD-AccountID':  getCookie('gdd_account_id'),
  }
}
