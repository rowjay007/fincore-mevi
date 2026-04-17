import { create } from 'zustand';

interface DashboardState {
  tps: number;
  totalVolume: string;
  activeUsers: number;
  fraudAlerts: number;
  systemHealth: 'healthy' | 'warning' | 'critical';
  setTPS: (tps: number) => void;
  incrementAlerts: () => void;
}

export const useDashboardStore = create<DashboardState>((set) => ({
  tps: 0,
  totalVolume: "0.00",
  activeUsers: 0,
  fraudAlerts: 0,
  systemHealth: 'healthy',
  setTPS: (tps) => set({ tps }),
  incrementAlerts: () => set((state) => ({ fraudAlerts: state.fraudAlerts + 1 })),
}));
