export interface ReadingPoint {
  ts: string;
  value: number;
  unit: string;
  source: string;
  note?: string;
  eventType?: string;
}

export interface MetricPoint {
  TS: string;
  Value: number;
  Unit: string;
  Source: string;
  Note?: string;
}

export interface DailyValue {
  date: string;
  value: number;
  unit: string;
}

export interface DomainValue {
  value: number | null;
  unit: string;
  costEur: number | null;
  pricePerUnit: number | null;
  priceUnit: string;
  intervals: number;
  coveredHours: number;
  estimate: boolean;
  status: string;
}

export interface DomainResponse {
  currentReading?: ReadingPoint | null;
  currentPower?: ReadingPoint | null;
  consumed: DomainValue;
  readings?: ReadingPoint[];
  powerReadings?: ReadingPoint[];
  daily: DailyValue[];
}

export interface Purchase {
  ts: string;
  quantityKg: number | null;
  totalCostEur: number | null;
  costPerKgEur: number | null;
  note?: string;
}

export interface HeatingResponse {
  currentStock: ReadingPoint | null;
  consumed: DomainValue;
  lifetimeConsumedKg: number | null;
  averageDaily30DaysKg: number | null;
  consumedLast30DaysKg: number | null;
  last30DaysIntervals: number;
  last30DaysCoveredHours: number;
  forecastNextHourKg: number | null;
  forecastStatus: string;
  averagePurchasePriceEurPerKg: number | null;
  stockReadings: ReadingPoint[];
  daily: DailyValue[];
  purchases: Purchase[];
}

export interface OverviewResponse {
  water: {
    consumed: DomainValue;
    currentReading: ReadingPoint | null;
  };
  electricity: {
    consumed: DomainValue;
    currentPower: ReadingPoint | null;
  };
  heating: {
    currentStock: ReadingPoint | null;
    consumedLast30DaysKg: number | null;
    averageDaily30DaysKg: number | null;
    forecastNextHourKg: number | null;
    forecastStatus: string;
  };
  totalCost: {
    totalEur: number | null;
    electricityEur: number | null;
    waterEur: number | null;
    heatingEur: number | null;
    complete: boolean;
    missingDomains: string[];
  };
}

export interface MeasurementPayload {
  ts: string;
  type: string;
  value: number;
  unit: string;
  source: string;
  note?: string;
}
