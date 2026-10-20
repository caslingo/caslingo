import { useCallback, useEffect, useState } from "react";

/** Loads data once (and again on reload), keeping the error to show instead of the page. */
export function useLoad<T>(load: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setError("");
    load()
      .then(result => !cancelled && setData(result))
      .catch((e: Error) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
    // the caller lists what load depends on
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, version]);

  const reload = useCallback(() => setVersion(v => v + 1), []);
  return { data, error, reload, setData };
}
