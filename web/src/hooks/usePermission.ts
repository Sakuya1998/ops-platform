import { useState, useCallback } from "react";
import { api } from "../services/api";

export function usePermission() {
  const [permissions] = useState<string[]>([]);

  const hasPermission = useCallback(
    (code: string) => permissions.includes(code),
    [permissions]
  );

  return { hasPermission, permissions };
}

export function useFetch(url: string) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get(url);
      setData(res.data.data);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [url]);

  return { data, loading, error, fetchData };
}
