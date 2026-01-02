// Re-export the main API client for backwards compatibility
export { takhinApi as default, takhinApi, TakhinApiClient } from './takhinApi'
export { authService } from './auth'
export { TakhinApiError, handleApiError } from './errors'
export * from './types'
