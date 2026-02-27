export type HealthApiResponse = {
    status: string;
    up_since: string;
    uptime: string;
    services: {
        [key: string]: {
            status: string;
            details?: string;
        }
    }
}

export type InitRegistrationResponse = {
    ifc_id: string;
    status: boolean;
    message?: string;
    steps: RegistrationStep[]
}

export interface InitServerResponse {
  va_code: string;
  status: boolean;
  message?: string;
  steps: RegistrationStep[];
}

export interface InitServerResult {
  success: boolean;
  message: string;
  va_code: string;
  va_id: string;
}

type RegistrationStep = {
    name: string;
    status: boolean;
    message: string;
}

export type ApiResponse<T> = {
    status: string;
    message?: string;
    responseTimeMs?: number;
    result?: T;
};

export interface RegistrationResult {
    success: boolean;
    message: string;
    is_va_registered: boolean;
}

export type RegistrationApiResponse = ApiResponse<RegistrationResult>;

export type FlightHistoryRecord = {
    origin: string;
    dest: string;
    timestamp: string;
    endtime: string;
    landings: number;
    server: string;
    equipment: string;
    mapUrl: string;
    callsign: string;
    violations: number;
    duration: string;      // HH:MM
    aircraft?: string;     // older API null-safety
    livery?: string;
    username?: string;     // Pilot's IFC username
};
  
  export type FlightHistoryPage = {
    page: number;
    records: FlightHistoryRecord[];
    error: string;
  };
  
export interface LiveFlightRecord {
  flight_id: string;
  callsign: string;
  username: string;
  session_name: string;
  latitude: number;
  longitude: number;
  altitude: number;
  speed: number;
  track: number;
  vertical_speed: number;
  aircraft_name: string;
  livery_name: string;
  phase: string;
  takeoff_time: string;
  origin: string;
  destination: string;
  last_updated: string;
  last_report: string;
  last_flight_plan_fetch: string;
  max_altitude?: number;
  max_speed?: number;
}

export interface VARole {
  va_id: string;
  va_name: string;
  va_code: string;
  role: string;
  is_active: boolean;
  joined_at: string;
}

export interface CurrentVA {
  is_member: boolean;
  role: string;
  is_active: boolean;
}

export interface UserDetailsData {
  user_id: string;
  discord_id: string;
  if_community_id: string;
  if_api_id: string;
  is_active: boolean;
  created_at: string;
  affiliations: VARole[];
  current_va: CurrentVA;
}

export type UserDetailsResponse = ApiResponse<UserDetailsData>;

export interface PilotStatsData {
  game_stats?: {
    flight_time?: number;
    online_flights?: number;
    landing_count?: number;
    xp?: number;
    grade?: string;
    violations?: number;
  };
  career_mode_data?: {
    airline?: string;
    aircraft?: string;
    total_cm_hours?: number;
    required_hours_to_next?: number;
    last_career_mode_flight?: string;
    assigned_routes?: string[];
    last_activity_cm?: string;
  };
  provider_data?: {
    join_date?: string;
    last_activity?: string;
    region?: string;
    additional_fields?: {
      callsign?: string;
      category?: string;
      cm_status?: string;
      [key: string]: any;
    };
    [key: string]: any;
  };
  metadata: {
    provider_configured: boolean;
    last_fetched: string;
    cached: boolean;
    va_name: string;
  };
}

export type PilotStatsResponse = ApiResponse<PilotStatsData>;

// PIREP Configuration Response Types
export interface FormField {
  name: string;
  type: "text" | "textarea" | "number";
  label: string;
  required: boolean;
}

export interface ModeResponse {
  mode_id: string;
  display_name: string;
  status: "valid" | "invalid";
  requires_route_selection: boolean;
  autofill_route?: string;
  fields: FormField[];
  available_routes?: RouteOption[];
  auto_route?: RouteOption;
  error_reason?: string;
}

export interface RouteOption {
  route_id: string;
  name: string;
  multiplier: number;
}

export interface UserInfo {
  callsign: string;
  ifc_username: string;
  current_aircraft: string;
  current_livery: string;
  current_route: string;
  current_flight_status: string;
  current_altitude?: number;
  current_speed?: number;
}

export interface PirepConfigData {
  user_info: UserInfo;
  available_modes: ModeResponse[];
}

export type PirepConfigResponse = ApiResponse<PirepConfigData>;

export interface PirepSubmitRequest {
  mode: string;
  route_id?: string;
  flight_time: string;
  pilot_remarks?: string;
  fuel_kg?: number;
  cargo_kg?: number;
  passengers?: number;
}

export interface PirepSubmitData {
  success: boolean;
  message: string;
  pirep_id?: string;
  error_type?: string;
  error_message?: string;
}

export type PirepSubmitResponse = ApiResponse<PirepSubmitData>;

export interface MembershipJoinResult {
  success: boolean;
  message: string;
  user_id: string;
  va_id: string;
  callsign: string;
  role: string;
}

export type MembershipJoinResponse = ApiResponse<MembershipJoinResult>;

// Event Response Types
export interface EventLegResponse {
  id: string;
  event_id: string;
  leg_number: number;
  origin: string;
  destination: string;
  route_at_id?: string;
  description?: string;
  thumbnail_url?: string;
  additional_data?: Record<string, any>;
  created_at: string;
  updated_at: string;
  created_by_id?: string;
  updated_by_id?: string;
}

export interface EventResponse {
  id: string;
  name: string;
  description?: string;
  status: string;
  start_date?: string;
  end_date?: string;
  is_active: boolean;
  legs?: EventLegResponse[];
  created_at: string;
  updated_at: string;
  created_by_id?: string;
  updated_by_id?: string;
}

export type EventsResponse = ApiResponse<EventResponse[]>;
