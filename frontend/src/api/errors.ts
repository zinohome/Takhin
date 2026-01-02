import { AxiosError } from 'axios'
import type { ApiError } from './types'

export class TakhinApiError extends Error {
  statusCode?: number
  apiError?: ApiError

  constructor(message: string, statusCode?: number, apiError?: ApiError) {
    super(message)
    this.name = 'TakhinApiError'
    this.statusCode = statusCode
    this.apiError = apiError
  }
}

export function handleApiError(error: unknown): TakhinApiError {
  if (error instanceof AxiosError) {
    const statusCode = error.response?.status
    const apiError = error.response?.data as ApiError | undefined

    if (statusCode === 401) {
      return new TakhinApiError('Unauthorized. Please check your API key.', statusCode, apiError)
    }

    if (statusCode === 404) {
      return new TakhinApiError(
        apiError?.error || 'Resource not found.',
        statusCode,
        apiError
      )
    }

    if (statusCode === 400) {
      return new TakhinApiError(
        apiError?.error || 'Bad request. Please check your input.',
        statusCode,
        apiError
      )
    }

    if (statusCode === 500) {
      return new TakhinApiError(
        apiError?.error || 'Internal server error.',
        statusCode,
        apiError
      )
    }

    if (statusCode === 503) {
      return new TakhinApiError(
        'Service unavailable. The server is not ready.',
        statusCode,
        apiError
      )
    }

    return new TakhinApiError(
      apiError?.error || error.message || 'An unexpected error occurred.',
      statusCode,
      apiError
    )
  }

  if (error instanceof Error) {
    return new TakhinApiError(error.message)
  }

  return new TakhinApiError('An unexpected error occurred.')
}
