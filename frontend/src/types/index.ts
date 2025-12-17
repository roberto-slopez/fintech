// User types
export interface User {
  id: string
  email: string
  full_name: string
  role: 'ADMIN' | 'ANALYST' | 'OPERATOR' | 'VIEWER'
  country_ids?: string[]
  is_active: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface LoginCredentials {
  email: string
  password: string
}

export interface LoginResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_at: string
}

// Country types
export interface Country {
  id: string
  code: string
  name: string
  currency: string
  timezone: string
  is_active: boolean
  config: CountryConfig
  created_at: string
  updated_at: string
}

export interface CountryConfig {
  min_loan_amount: number
  max_loan_amount: number
  min_income_required: number
  max_debt_to_income_ratio: number
  review_threshold: number
  min_credit_score: number
}

export interface DocumentType {
  id: string
  country_id: string
  code: string
  name: string
  validation_regex: string
  is_required: boolean
  created_at: string
}

export interface CountryRule {
  id: string
  country_id: string
  rule_type: string
  name: string
  description: string
  is_active: boolean
  priority: number
  config: Record<string, any>
  created_at: string
  updated_at: string
}

// Application types
export type ApplicationStatus = 
  | 'PENDING'
  | 'VALIDATING'
  | 'PENDING_BANK_INFO'
  | 'UNDER_REVIEW'
  | 'APPROVED'
  | 'REJECTED'
  | 'CANCELLED'
  | 'EXPIRED'
  | 'DISBURSED'

export interface CreditApplication {
  id: string
  country_id: string
  country?: Country
  full_name: string
  document_type: string
  document_number: string
  email: string
  phone?: string
  requested_amount: number
  monthly_income: number
  status: ApplicationStatus
  status_reason?: string
  requires_review: boolean
  banking_info?: BankingInfo
  validation_results?: ValidationResult[]
  risk_score?: number
  application_date: string
  processed_at?: string
  created_at: string
  updated_at: string
}

export interface BankingInfo {
  id: string
  application_id: string
  provider_id: string
  provider_name: string
  credit_score?: number
  total_debt?: number
  available_credit?: number
  payment_history?: string
  bank_accounts: number
  active_loans: number
  months_employed?: number
  retrieved_at: string
  expires_at: string
}

export interface ValidationResult {
  rule_id: string
  rule_name: string
  passed: boolean
  message: string
  requires_review: boolean
}

export interface StateTransition {
  id: string
  application_id: string
  from_status: ApplicationStatus
  to_status: ApplicationStatus
  reason?: string
  triggered_by: string
  triggered_by_id?: string
  created_at: string
}

export interface ApplicationFilter {
  country_id?: string
  country?: string
  status?: ApplicationStatus
  requires_review?: boolean
  from_date?: string
  to_date?: string
  min_amount?: number
  max_amount?: number
  search?: string
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'ASC' | 'DESC'
}

export interface ApplicationListResult {
  applications: CreditApplication[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// WebSocket types
export interface WebSocketMessage {
  type: string
  data: any
  country_id?: string
  timestamp: string
}

// API Response types
export interface ApiError {
  error: string
  message: string
}

