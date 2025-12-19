export type ApiResponse<T> = {
  data: T;
  error?: never;
};

export type ApiError = {
  error: string;
  data?: never;
};

export type ApiResult<T> = ApiResponse<T> | ApiError;

export function isApiError<T>(result: ApiResult<T>): result is ApiError {
  return 'error' in result && result.error !== undefined;
}

export function isApiSuccess<T>(
  result: ApiResult<T>,
): result is ApiResponse<T> {
  return 'data' in result && result.data !== undefined;
}
