export type InitServerRequest = {
  va_code: string;          // 3–4 letters
  prefix?: string;          // optional ≤10
  suffix?: string;          // optional ≤10
}