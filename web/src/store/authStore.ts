import { create } from "zustand";
import { api } from "../services/api";

interface User {
  user_id: string;
  org_id: string;
  username: string;
  display_name: string;
  email: string;
  roles: string[];
  must_change_password?: boolean;
  mfa_enabled?: boolean;
}

interface AuthState {
  token: string | null;
  user: User | null;
  login: (username: string, password: string, mfaCode?: string) => Promise<void>;
  logout: () => void;
  loadUser: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem("token"),
  user: null,

  login: async (username, password, mfaCode) => {
    const res = await api.post("/auth/login", { username, password, mfa_code: mfaCode });
    const { access_token, user } = res.data.data;
    localStorage.setItem("token", access_token);
    set({ token: access_token, user });
  },

  logout: () => {
    localStorage.removeItem("token");
    set({ token: null, user: null });
  },

  loadUser: async () => {
    try {
      const res = await api.get("/auth/me");
      set({ user: res.data.data });
    } catch {
      set({ token: null, user: null });
      localStorage.removeItem("token");
    }
  },
}));
