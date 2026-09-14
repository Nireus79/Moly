import { useEffect, useState } from 'react';
import { getBackendManager } from '@/api/backendManager';

/**
 * Hook to get the auto-detected backend URL
 * Returns null if backend is not available
 */
export function useBackendUrl(): string {
  const [url, setUrl] = useState<string>(() => {
    const manager = getBackendManager();
    return manager.getBackendUrl();
  });

  useEffect(() => {
    const manager = getBackendManager();

    // Update on status changes
    const handleStatusChange = (status: any) => {
      if (status.running) {
        setUrl(status.url);
      }
    };

    manager.onStatusChange(handleStatusChange);

    return () => {
      // Cleanup if needed
    };
  }, []);

  return url;
}
