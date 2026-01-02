const API_KEY_STORAGE_KEY = 'takhin_api_key'

export const authService = {
  setApiKey(apiKey: string): void {
    localStorage.setItem(API_KEY_STORAGE_KEY, apiKey)
  },

  getApiKey(): string | null {
    return localStorage.getItem(API_KEY_STORAGE_KEY)
  },

  removeApiKey(): void {
    localStorage.removeItem(API_KEY_STORAGE_KEY)
  },

  isAuthenticated(): boolean {
    return this.getApiKey() !== null
  },

  getAuthHeader(): string | undefined {
    const apiKey = this.getApiKey()
    return apiKey ? `Bearer ${apiKey}` : undefined
  },
}
